package apollon

import (
	"Loxias/database"
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
	"time"

	"apollon.chat.com/packets"
)

func HandleOldMessages(id uint32, connection net.Conn) {
	log.Printf("Handling messages for \"%d\"", id)
	messages, err := database.ReadMessagesFromFile(fmt.Sprint(id) + ".json")
	if err != nil {
		log.Printf("No messages for client \"%d\" found", id)
	} else {
		for _, v := range messages {
			// TODO: Fix this, for now this is only to fix the compiliation error
			header := packets.Header{
				Category:  packets.CAT_DATA,
				Type:      packets.D_TEXT,
				UserId:    id,
				MessageId: messages[0].ContactUserId,
			}
			raw, err := packets.SerializePacket(header, v)
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

	defer connection.Close()

	// incoming := make(chan []byte)
	// go HandleIncoming(connection, incoming)

	// Init the random number generator
	rand.Seed(time.Now().UnixNano())
	reader := bufio.NewReader(connection)
	var id uint32
	id = 0
	headerBuffer := make([]byte, 10)

	for {
		// read, err := reader.ReadString('\n') // TODO: Switch to this version.
		read, err := reader.Read(headerBuffer)

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

		if read != 10 {
			log.Fatal("The first packet was not enough to fit the header!")
		}

		// log.Printf("Raw: %x", headerBuffer)
		// log.Printf("Raw String: %s", string(headerBuffer))

		// Decode the header information
		var header packets.Header
		newReader := bytes.NewReader(headerBuffer)
		err = binary.Read(newReader, binary.BigEndian, &header)
		if err != nil {
			log.Println("Failed to extract header information from packet")
			// TODO: Is this rather due to an transmission error or because the client send wrong information? This should NORMALLY only happen if the client is malicous and sends incorrect data as the first part of the packet -> return
			log.Printf("Set client %d to nil", id)
			delete(db, id)
			return
		}
		id = header.UserId
		log.Printf("Header:\n%s", hex.Dump(headerBuffer))

		switch header.Category {
		case packets.CAT_CONTACT:
			switch header.Type {
			case packets.CON_CREATE:
				// Reading the actual payload
				payload, err := reader.ReadSlice('\n')
				if err != nil {
					log.Printf("Failed to read payload of create packet!\n%s", err)
					continue
				}
				var create packets.Create
				create, err = packets.DeseralizePacket[packets.Create](payload)

				if err != nil {
					// This only happens if incorrect JSON was send
					delete(db, id)
					return
				}
				// Generate new user id
				// TODO: Make faster in case most IDs are used
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
				header.UserId = newUserId
				// Store new user in some sort of database
				err = database.StoreInDatabase(newUserId, create.Username)
				if err != nil {
					// Failed to insert user into database
					continue
				}
				// Sending back the ID to the client
				encoded, err := packets.SerializePacket(header, nil)
				if err != nil {
					log.Println("Failed to encode answer")
					continue
				}
				connection.Write(encoded)
			case packets.CON_SEARCH:
				payload, err := reader.ReadSlice('\n')

				if err != nil {
					log.Printf("Failed to read payload of search packet!%s\n", err)
					continue
				}

				var search packets.Search
				search, err = packets.DeseralizePacket[packets.Search](payload)
				if err != nil {
					log.Println("Failed to deserialize search payload")
					delete(db, id)
					return
				}
				users := database.SearchUsers(search.UserIdentifier)
				log.Printf("%d users for identifier \"%s\" found", len(users), search.UserIdentifier)
				header, contactList := packets.CreateContactList(header.UserId, header.MessageId, users)

				encoded, err := packets.SerializePacket(header, contactList)
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
				payload, err := reader.ReadSlice('\n')
				if err != nil {
					log.Printf("Failed to read payload of contact option packet!%s\n", err)
					continue
				}
				option, err := packets.DeseralizePacket[packets.ContactOption](payload)
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
				err = HandleContactOption(header, option, connection, forwardCon)
				if err != nil {
					delete(db, id)
					return
				}
			case packets.CON_LOGIN:
				// Nothing else to do here. The client is already inserted into the database and the login packet seems to have the correct format!
				log.Printf("Login from user %d", header.UserId)
				if header.UserId == 0 {
					log.Printf("Unknown user %d! Killing connection!", header.UserId)
					delete(db, header.UserId)
					return
				}
				_, err := database.SearchUserId(header.UserId)
				if err != nil {
					log.Printf("Failed to find user with ID %d! Killing connection", header.UserId)
					delete(db, header.UserId)
					return
				}
				// Proceed to handle user otherwise
				db[id] = connection
				go HandleOldMessages(id, connection)
			case packets.CON_CONTACT_INFO:
				_, ex := db[id]
				if !ex {
					log.Printf("User %d not logged in. Cannot process anything without registration and login!", id)
					return
				}

				payload, err := reader.ReadSlice('\n')
				if err != nil {
					log.Printf("Failed to read payload of contact info packet!%s\n", err)
					continue
				}
				contact, err := packets.DeseralizePacket[packets.ContactInfo](payload)
				if err != nil {
					log.Println("Failed to deserialize contact information packet!")
					delete(db, id)
					return
				}
				// log.Printf("Got contact information: %s", string(contentBuf))
				// Acknowledge that we received the packet
				infoAck := packets.CreateContactInfoAck(header.UserId, header.MessageId)
				rawInfoAck, err := packets.SerializePacket(infoAck, nil)
				if err != nil {
					log.Printf("Failed to serialize acknowledgement header!\n%s", err)
					continue
				}
				connection.Write(rawInfoAck)

				// Forwarding to first friend currently TODO: Fix this
				for _, v := range contact.ContactIds {
					forwardCon, ex := db[v]
					if !ex {
						log.Printf("Contact %du not online\n", contact.ContactIds[0])
						continue
					}
					forward, err := packets.SerializePacket(header, contact)
					if err != nil {
						log.Println("Failed to serialize contact packet")
						continue
					}
					forwardCon.Write(forward)
					log.Printf("Forwarded image to %du\n", contact.ContactIds[0])
				}
			default:
				log.Printf("Incorrect packet type: %d\n", header.Type)
				delete(db, id)
				return
			}
		case packets.CAT_DATA:
			switch header.Type {
			case packets.D_TEXT:
				_, ex := db[id]
				if !ex {
					log.Printf("User %d not logged in. Cannot process anything without registration and login!", id)
					return
				}

				payload, err := reader.ReadSlice('\n')
				if err != nil {
					log.Printf("Failed to read payload of contact info packet!%s\n", err)
					continue
				}
				var text packets.Text
				text, err = packets.DeseralizePacket[packets.Text](payload)
				log.Printf("Got \"%s\" forwarding to \"%d\"\n", text.Message, text.ContactUserId)
				if err != nil {
					log.Println("Failed to deserialize text packet")
					delete(db, id)
					return
				}

				// First write the ack back to the sending client (later on save the text and send to client when it comes back online)
				ackHeader, textAck := packets.CreateTextAck(header.UserId, header.MessageId, text.ContactUserId)
				ack, err := packets.SerializePacket(ackHeader, textAck)
				if err != nil {
					log.Println("Failed to create ack packet")
					continue
				}
				connection.Write(ack)
				// log.Printf("Wrote textAck back to %d\n", header.UserId)

				// Continue with forwarding the text
				forwardCon, ex := db[text.ContactUserId]
				// TODO: fix this with using delete(db, id)
				if !ex {
					log.Printf("Contact %d not online", text.ContactUserId)
					database.SaveMessagesToFile(text, fmt.Sprint(text.ContactUserId)+".json")
					continue
				}
				log.Printf("Text before sending: %v", text)
				forward, err := packets.SerializePacket(header, text)
				if err != nil {
					log.Printf("Failed to create forward packet!")
					continue
				}
				log.Printf("Sending:\n%s", hex.Dump(forward))
				forwardCon.Write(forward)
			case packets.D_TEXT_ACK:
				// TODO: When this is received send it further to acked client so that he can show the "received" flag
				payload, err := reader.ReadSlice('\n')
				if err != nil {
					log.Printf("Failed to read payload of contact info packet!%s\n", err)
					continue
				}
				var textAck packets.TextAck
				textAck, err = packets.DeseralizePacket[packets.TextAck](payload)

				// Lookup the contacted user and forward
				forwardCon, ex := db[textAck.ContactUserId]
				if !ex {
					log.Printf("Contact %d not online", textAck.ContactUserId)
					database.SaveTextAckToFile(textAck, fmt.Sprint(textAck.ContactUserId)+".json")
					continue
				}
				forward, err := packets.SerializePacket(header, textAck)
				if err != nil {
					log.Printf("Failed to create forward packet!")
					continue
				}
				forwardCon.Write(forward)
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

func HandleContactOption(header packets.Header, option packets.ContactOption, connection net.Conn, forwardCon net.Conn) error {
	for _, v := range option.Options {
		log.Printf("Option: {%s, %s}", v.Type, v.Value)
		switch v.Type {
		case "Question":
			switch v.Value {
			case "Add":
				log.Printf("User %d wants to add %d", header.UserId, option.ContactUserId)
				_, err := database.GetUser(option.ContactUserId)
				if err != nil {
					log.Printf("%s", err)
					break
				}

				// TODO: Add forwarding the request to be able to automatically add the user into the list
				if forwardCon != nil {
					forwardPacket, err := packets.SerializePacket(header, option)
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
					Value: option.Options[len(option.Options)-1].Value,
				}
				options := make([]packets.Option, 2)
				options[0] = answerOption
				options[1] = nameOption
				answerHeader := packets.Header{
					Category:  packets.CAT_CONTACT,
					Type:      packets.CON_OPTION,
					UserId:    option.ContactUserId,
					MessageId: header.MessageId,
				}
				accept := packets.ContactOption{
					ContactUserId: header.UserId,
					Options:       options,
				}
				packet, err := packets.SerializePacket(answerHeader, accept)
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
				answerHeader := packets.Header{
					Category:  packets.CAT_CONTACT,
					Type:      packets.CON_OPTION,
					UserId:    option.ContactUserId,
					MessageId: header.MessageId,
				}
				ack := packets.ContactOption{
					ContactUserId: header.UserId,
					Options:       options,
				}
				packet, err := packets.SerializePacket(answerHeader, ack)
				if err != nil {
					log.Printf("Failed to create next packet")
					break
				}
				connection.Write(packet)
			default:
				log.Printf("Unknown or incorrect contact option value \"%s\". Closing connection...", v.Value)
				return errors.New("unknown contact value")
			}
		case "Add":
			log.Printf("User is adding the contact and sending name: %s", v.Value)

		default:
			log.Printf("Unknown contact option type \"%s\"", v.Type)
			return errors.New("unknown contact type")
		}
	}
	return nil
}
