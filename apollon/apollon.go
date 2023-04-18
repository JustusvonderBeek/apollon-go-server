package apollon

import (
	"Loxias/database"
	"Loxias/packets"
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
	"time"
)

func CreatePacket(content any) ([]byte, error) {
	packet, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}
	if len(packet) > 65534 {
		log.Println("Packet longer than 16 kB are not supported")
		return nil, errors.New("Packet too long")
	}
	size := uint16(len(packet) + 2)
	buffer := make([]byte, size)
	binary.BigEndian.PutUint16(buffer, size)
	copied := copy(buffer[2:], packet)
	if copied < len(packet) {
		log.Printf("Something went wrong during packet creation! Should copy %d, but copied only %d", size, copied)
		return nil, errors.New("Failed to copy all data")
	}
	// log.Printf("Packet: %02x", buffer)
	return buffer, nil
}

// func HandleIncoming(connection net.Conn, read chan []byte) {
// 	log.Println("Starting incoming listener...")
// 	for {
// 		// Must be already completly read packets (including size info)
// 		packet := <-read
// 		connection.Write(packet)
// 	}
// }

func HandleOldMessages(id uint32, connection net.Conn) {
	log.Printf("Handling messages for \"%d\"", id)
	messages, err := database.ReadMessagesFromFile(fmt.Sprint(id) + ".json")
	if err != nil {
		log.Printf("No messages for client \"%d\" found", id)
	} else {
		for _, v := range messages {
			raw, err := CreatePacket(v)
			if err != nil {
				continue
			}
			connection.Write(raw)
		}
	}
	// TODO: For now only rename the file but later make sure the packets arrived and then remove from the file // or delete the file completly
	os.Rename(fmt.Sprint(id)+".json", "_"+fmt.Sprint(id)+".json")
}

func HandleClient(connection net.Conn, db map[uint32]net.Conn) {
	log.Println("Handling client...")

	// incoming := make(chan []byte)
	// go HandleIncoming(connection, incoming)

	// Init the random number generator
	rand.Seed(time.Now().UnixNano())
	reader := bufio.NewReader(connection)
	var id uint32
	id = 0
	sizeBuf := make([]byte, 2)

	for {
		read, err := reader.Read(sizeBuf)

		// log.Printf("%T %+v", err, err) // checking err type
		if read == 0 {
			log.Printf("Connection \"%d\" closed by remote host", id)
			delete(db, id)
			return
		}

		if err != nil {
			log.Println("Failed to read size information from client!")
			continue
		}

		// log.Printf("Read %d bytes size information from the client", read)

		size := binary.BigEndian.Uint16(sizeBuf)
		log.Printf("Expecting %d bytes of data", size)
		if size <= 4 {
			// TODO: Define specific keep alive packet
			log.Println("Keep alive packet")
			continue
		}

		// ADAPT SIZE INFORMATION IF SIZE FIELD CHANGES
		contentBuf := make([]byte, size-2)
		read, err = reader.Read(contentBuf)

		if read == 0 {
			log.Printf("Connection \"%d\" closed by remote host", id)
			delete(db, id)
			return
		}

		if err != nil {
			log.Println("Failed to read data from client!")
			continue
		}

		// log.Printf("Raw: %x", contentBuf)
		// log.Printf("Raw String: %s", string(contentBuf))

		// category, typ, err := packets.PacketType(contentBuf)
		// packet, err := packets.DeseralizePacket(contentBuf)
		var header packets.Header
		err = json.Unmarshal(contentBuf, &header)
		if err != nil {
			log.Println("Failed to extract header information from packet")
			if len(contentBuf) > 100 {
				log.Printf("Content (first 100 bytes): %x", contentBuf[:100])
			} else {
				log.Printf("Content: %x", contentBuf[:])
			}
			// TODO: Is this rather due to an transmission error or because the client send wrong information? This should NORMALLY only happen if the client is malicous and sends incorrect data as the first part of the packet -> return
			log.Printf("Set client %d to nil", id)
			delete(db, id)
			return
		}
		id = header.UserId
		log.Printf("Packet:\n%s", hex.Dump(contentBuf))

		if id > 0 {
			db[id] = connection
			// TODO: In order to trigger this right at the beginning we would need something as a login message with an id in it
			go HandleOldMessages(id, connection)
		}

		log.Printf("User \"%d\" connected and online", id)

		switch header.Category {
		case packets.CAT_CONTACT:
			switch header.Type {
			case packets.CON_CREATE:
				var create packets.Create
				create, err = packets.DeseralizePacket[packets.Create](contentBuf)
				if err != nil {
					// This only happens if incorrect JSON was send
					delete(db, id)
					return
				}
				// Generate new user id
				newUserId := rand.Uint32()
				safeCounter := math.MaxInt32
				for {
					exists := database.IdExists(newUserId)
					if !exists || safeCounter <= 0 {
						break
					}
					safeCounter--
					newUserId = rand.Uint32()
				}
				create.UserId = newUserId
				// Store new user in some sort of database
				err = database.StoreInDatabase(create)
				if err != nil {
					// Failed to insert user into database
					continue
				}
				// Sending back the ID to the client
				encoded, err := CreatePacket(create)
				if err != nil {
					log.Println("Failed to encode answer")
					continue
				}
				connection.Write(encoded)
			case packets.CON_SEARCH:
				var search packets.Search
				search, err = packets.DeseralizePacket[packets.Search](contentBuf)
				if err != nil {
					log.Println("Failed to deserialize packet")
					delete(db, id)
					return
				}
				users := database.SearchUsers(search.UserIdentifier)
				log.Printf("%d users for identifier \"%s\" found", len(users), search.UserIdentifier)
				contactList, err := packets.CreateContactList(search, users)
				if err != nil {
					// Internal error, tried our best
					log.Println("Contact list could not be created")
					continue
				}

				encoded, err := CreatePacket(contactList)
				if err != nil {
					log.Println("Failed to encode contact list")
					continue
				}
				// log.Printf("Raw: %s", string(encoded))
				connection.Write(encoded)
			case packets.CON_CONTACTS:
				// Should never be sent to the server
				log.Println("Received contact list! Should not be received on the server side! Closing connection!")
				delete(db, id)
				return
			case packets.CON_OPTION:
				option, err := packets.DeseralizePacket[packets.ContactOption](contentBuf)
				if err != nil {
					log.Println("Failed to deserialize packet!")
					delete(db, id)
					return
				}
				// log.Println("Received contact option")
				forwardCon, ex := db[option.ContactUserId]
				if !ex {
					// TODO: Save question to file
					forwardCon = nil
				}
				err = HandleContactOption(option, connection, forwardCon)
				if err != nil {
					delete(db, id)
					return
				}
			case packets.CON_LOGIN:
				login, err := packets.DeseralizePacket[packets.Login](contentBuf)
				if err != nil {
					log.Println("Failed to deserialize login packet!")
					delete(db, id)
					return
				}
				// Nothing else to do here. The client is already inserted into the database and the login packet seems to have the correct format!
				log.Printf("Login from user %d", login.UserId)
			case packets.CON_CONTACT_INFO:
				contact, err := packets.DeseralizePacket[packets.ContactInfo](contentBuf)
				if err != nil {
					log.Println("Failed to deserialize contact information packet!")
					delete(db, id)
					return
				}
				// log.Printf("Got contact information: %s", string(contentBuf))
				log.Printf("Got data from %d, expecting image of size %d", contact.UserId, contact.ImageBytes)
				imageBuffer := make([]byte, contact.ImageBytes)
				// Variant to wait for the full image
				// TODO: Create a timeout for this method, otherwise one could make this wait infinite
				read, err = io.ReadFull(reader, imageBuffer)
				if err != nil {
					log.Printf("Failed to read image from remote!\n%s", err)
				}
				log.Printf("Got %d bytes, first 100 image bytes %x", read, hex.Dump(imageBuffer[:100]))
			default:
				log.Printf("Incorrect packet type: %d", header.Type)
				delete(db, id)
				return
			}
		case packets.CAT_DATA:
			switch header.Type {
			case packets.D_TEXT:
				var text packets.Text
				text, err = packets.DeseralizePacket[packets.Text](contentBuf)
				log.Printf("Got \"%s\" forwarding to \"%d\"", text.Message, text.ContactUserId)
				if err != nil {
					log.Println("Failed to deserialize text packet")
					delete(db, id)
					return
				}

				// First write the ack back to the sending client (later on save the text and send to client when it comes back online)
				textAck := packets.TextAck{
					Category:      packets.CAT_DATA,
					Type:          packets.D_TEXT_ACK,
					UserId:        text.ContactUserId,
					MessageId:     text.MessageId,
					ContactUserId: text.UserId,
					Timestamp:     text.Timestamp,
					AckPart:       text.Part,
				}
				ack, err := CreatePacket(textAck)
				if err != nil {
					log.Println("Failed to create ack packet")
					continue
				}
				connection.Write(ack)
				log.Printf("Wrote textAck back to %d", text.UserId)

				// Continue with forwarding the text
				forwardCon, ex := db[text.ContactUserId]
				// TODO: fix this with using delete(db, id)
				if !ex {
					log.Printf("Contact %d not online", text.ContactUserId)
					database.SaveMessagesToFile(text, fmt.Sprint(text.ContactUserId)+".json")
					continue
				}
				forward, err := CreatePacket(text)
				if err != nil {
					log.Printf("Failed to create forward packet!")
					continue
				}
				forwardCon.Write(forward)
			case packets.D_TEXT_ACK:
				// TODO: When this is received send it further to acked client so that he can show the "received" flag
				// textAck, err = packets.DeseralizePacket[packets.TextAck](contentBuf)

			default:
				log.Printf("Incorrect packet type %d", header.Type)
				delete(db, id)
				return
			}
		default:
			log.Printf("Incorrect packet category: %d", header.Category)
			delete(db, id)
			return
		}
	}
}

func HandleContactOption(option packets.ContactOption, connection net.Conn, forwardCon net.Conn) error {
	for _, v := range option.Options {
		log.Printf("Option: {%s, %s}", v.Type, v.Value)
		switch v.Type {
		case "Question":
			switch v.Value {
			case "Add":
				log.Printf("User %d wants to add %d", option.UserId, option.ContactUserId)
				_, err := database.GetUser(option.ContactUserId)
				if err != nil {
					log.Printf("%s", err)
					break
				}

				// TODO: Add forwarding the request to be able to automatically add the user into the list
				if forwardCon != nil {
					forwardPacket, err := CreatePacket(option)
					if err != nil {
						log.Print("Failed to create Option packet to forward!")
						break
					}
					forwardCon.Write(forwardPacket)
				}
				log.Print("The questioned client is not online!")

				// Forwarding the request to the other user
				// if user.Connection == nil {
				// 	log.Printf("User \"%d\" is currently not online!", option.ContactUserId)
				// 	// TODO: Save the request and send it as soon as the other client comes online
				// 	return nil
				// }
				// packet, err := CreatePacket(option)
				// if err != nil {
				// 	log.Println("Failed to create next packet!")
				// 	return nil
				// }
				// connection.Write(packet)

				// Because we currently don't have the request implemented on the other client we just send the accept answer back (for testing purposes)
				answerOption := packets.Option{
					Type:  "Answer",
					Value: "Accept",
				}
				nameOption := packets.Option{
					Type:  "Name",
					Value: "username",
				}
				options := make([]packets.Option, 2)
				options[0] = answerOption
				options[1] = nameOption
				accept := packets.ContactOption{
					Category:      packets.CAT_CONTACT,
					Type:          packets.CON_OPTION,
					UserId:        option.ContactUserId,
					MessageId:     option.MessageId,
					ContactUserId: option.UserId,
					Options:       options,
				}
				packet, err := CreatePacket(accept)
				if err != nil {
					log.Println("Failed to create answer packet!")
					break
				}
				connection.Write(packet)
			case "Remove":
				// TODO: Implement the acknowledgement on the client side before sending out the ack.
				// For testing purposes the ack is send so that the client is successfully removed
				removeAck := packets.Option{
					Type:  "Answer",
					Value: "RemoveAck",
				}
				options := make([]packets.Option, 1)
				options[0] = removeAck
				ack := packets.ContactOption{
					Category:      packets.CAT_CONTACT,
					Type:          packets.CON_OPTION,
					UserId:        option.ContactUserId,
					MessageId:     option.MessageId,
					ContactUserId: option.UserId,
					Options:       options,
				}
				packet, err := CreatePacket(ack)
				if err != nil {
					log.Printf("Failed to create next packet")
					break
				}
				connection.Write(packet)
			default:
				log.Printf("Unknown or incorrect contact option value \"%s\". Closing connection...", v.Value)
				return errors.New("unknown contact value")
			}
		default:
			log.Printf("Unknown contact option type \"%s\"", v.Type)
			return errors.New("unknown contact type")
		}
	}
	return nil
}
