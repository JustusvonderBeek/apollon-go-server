package apollon

import (
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

	"anzu.cloudsheeptech.com/database"
	"anzu.cloudsheeptech.com/packets"
)

var MESSAGE_QUEUE_SIZE int = 50

type StoreMessage struct {
	MessageID uint32
	Type      int16
}

func HandleOldMessages(id uint32, connection net.Conn) {
	log.Printf("Handling messages for \"%d\"", id)
	messages, err := database.ReadMessagesFromFile(fmt.Sprint(id) + ".json")
	if err != nil {
		log.Printf("No messages for client \"%d\" found", id)
	} else {
		log.Printf("Sending %d messages from %d", len(messages), id)
		random := rand.NewSource(time.Now().UnixNano())
		initMessageId := uint32(random.Int63())
		for _, v := range messages {
			// TODO: Fix this, for now this is only to fix the compiliation error
			log.Print("Sending next packet...")
			header := packets.Header{
				Category:  packets.CAT_DATA,
				Type:      packets.D_TEXT,
				UserId:    v.ContactUserId,
				MessageId: initMessageId,
			}
			v.ContactUserId = id
			initMessageId += 1
			raw, err := packets.SerializePacket(header, v)
			if err != nil {
				log.Printf("Failed to serialize packet: %s", err)
				continue
			}
			log.Printf("Sending:\n%s", hex.Dump(raw))
			connection.Write(raw)
		}
	}
	// TODO: For now only rename the file but later make sure the packets arrived and then remove from the file // or delete the file completly
	os.Rename(fmt.Sprint(id)+".json", "_"+fmt.Sprint(id)+".json")
}

func MessageIDExists(messageId uint32, lastMessageIDs []StoreMessage) int {
	for i, v := range lastMessageIDs {
		if v.MessageID == messageId {
			return i
		}
	}
	return -1
}

func AlreadySeen(category byte, pType byte, existing int16) bool {
	packet := (int16(category) << 8) | int16(pType)
	return packet == existing
}

func AddMessageId(messageId uint32, category byte, pType byte, count *int, lastMessageIDs *[]StoreMessage) {
	(*lastMessageIDs)[*count] = StoreMessage{MessageID: messageId, Type: (int16(category) << 8) | int16(pType)}
	*count = (*count + 1) % MESSAGE_QUEUE_SIZE
}

func HandleClient(connection net.Conn, db map[uint32]net.Conn) {
	log.Println("Handling client...")

	defer connection.Close()

	// incoming := make(chan []byte)
	// go HandleIncoming(connection, incoming)

	// Init the random number generator
	rand.New(rand.NewSource(time.Now().UnixNano()))
	reader := bufio.NewReader(connection)
	largePacketBuffer := make([]byte, 0)
	var id uint32
	id = 0
	// Keeping track of the last n messageIDs for this client
	lastMessageId := make([]StoreMessage, MESSAGE_QUEUE_SIZE)
	count := 0

	// headerBuffer := make([]byte, 10)

	for {
		// Blocking call... but then how to handle data that should be forwarded?
		// Idea: own thread that is only responsible for forwarding data
		inBuffer, err := reader.ReadSlice('\n')
		// read, err := reader.Read(headerBuffer)

		// log.Printf("%T %+v", err, err) // checking err type
		if len(inBuffer) == 0 {
			log.Printf("Connection \"%d\" closed by remote host", id)
			delete(db, id)
			return
		}

		// In case we received larger packets, append to existing data
		if err == bufio.ErrBufferFull {
			// log.Printf("Failed to read all data in one go")
			largePacketBuffer = append(largePacketBuffer, inBuffer...)
			continue
		}

		if err != nil {
			log.Printf("Failed to read size information from client: %s", err)
			continue
		}

		if len(largePacketBuffer) > 0 {
			inBuffer = append(largePacketBuffer, inBuffer...)
			largePacketBuffer = make([]byte, 0)
		}

		if len(inBuffer) < 10 {
			log.Fatal("The first packet was not enough to fit the header!")
			// Flush the pipe or wait?
			continue
		}

		// log.Printf("Raw: %x", headerBuffer)
		// log.Printf("Raw String: %s", string(headerBuffer))

		// Decode the header information (first 10 bytes, checked that available)
		var header packets.Header
		newReader := bytes.NewReader(inBuffer[:10])
		err = binary.Read(newReader, binary.BigEndian, &header)
		if err != nil {
			log.Println("Failed to extract header information from packet")
			// TODO: Is this rather due to an transmission error or because the client send wrong information? This should NORMALLY only happen if the client is malicous and sends incorrect data as the first part of the packet -> return
			log.Printf("Set client %d to nil", id)
			delete(db, id)
			return
		}
		id = header.UserId
		payload := inBuffer[10:]
		log.Printf("Header:\n%s", hex.Dump(inBuffer[:10]))

		switch header.Category {
		case packets.CAT_CONTACT:
			switch header.Type {
			case packets.CON_CREATE:
				if count != 0 {
					log.Printf("Create packet after connection establishment!")
					delete(db, id)
					return
				}

				AddMessageId(header.MessageId, header.Category, header.Type, &count, &lastMessageId)

				// Already read the whole payload, no more need to do this!
				// // Reading the actual payload
				// payload, err := reader.ReadSlice('\n')
				// if err != nil {
				// 	log.Printf("Failed to read payload of create packet!\n%s", err)
				// 	continue
				// }

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
				// Logging in the client
				db[newUserId] = connection

				// Sending back the ID to the client
				encoded, err := packets.SerializePacket(header, nil)
				if err != nil {
					log.Println("Failed to encode answer")
					continue
				}
				log.Printf("Writing create ack back:\n%s", hex.Dump(encoded))
				connection.Write(encoded)
			case packets.CON_SEARCH:
				_, ex := db[id]
				if !ex {
					log.Printf("User %d not logged in. Cannot process anything without registration and login!", id)
					return
				}

				if index := MessageIDExists(header.MessageId, lastMessageId); index > -1 {
					log.Printf("MessageID has already been seen!")
					stored := lastMessageId[index]
					if !AlreadySeen(header.Category, header.Type, stored.Type) {
						delete(db, id)
						return
					} else {
						// This packet is a duplicate, continue
						// But first "clean the pipe"
						// _, _ = reader.ReadSlice('\n')
						continue
					}
				}

				AddMessageId(header.MessageId, header.Category, header.Type, &count, &lastMessageId)

				// payload, err := reader.ReadSlice('\n')
				// if err != nil {
				// 	log.Printf("Failed to read payload of search packet!%s\n", err)
				// 	continue
				// }

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
				_, ex := db[id]
				if !ex {
					log.Printf("User %d not logged in. Cannot process anything without registration and login!", id)
					return
				}

				if index := MessageIDExists(header.MessageId, lastMessageId); index > -1 {
					log.Printf("MessageID has already been seen!")
					stored := lastMessageId[index]
					if !AlreadySeen(header.Category, header.Type, stored.Type) {
						delete(db, id)
						return
					} else {
						// This packet is a duplicate, continue
						// _, _ = reader.ReadSlice('\n')
						continue
					}
				}

				AddMessageId(header.MessageId, header.Category, header.Type, &count, &lastMessageId)

				// payload, err := reader.ReadSlice('\n')
				// if err != nil {
				// 	log.Printf("Failed to read payload of contact option packet!%s\n", err)
				// 	continue
				// }

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
					// database.SaveContactOption(option, fmt.Sprint(option.ContactUserId)+".json")
					forwardCon = nil
					continue
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

				if count != 0 {
					log.Print("Login in incorrect (established) state!")
					return
				}

				AddMessageId(header.MessageId, header.Category, header.Type, &count, &lastMessageId)

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

				if index := MessageIDExists(header.MessageId, lastMessageId); index > -1 {
					log.Printf("MessageID has already been seen!")
					stored := lastMessageId[index]
					if !AlreadySeen(header.Category, header.Type, stored.Type) {
						delete(db, id)
						return
					} else {
						// This packet is a duplicate, continue
						// _, _ = reader.ReadSlice('\n')
						continue
					}
				}

				AddMessageId(header.MessageId, header.Category, header.Type, &count, &lastMessageId)

				// payload, err := reader.ReadSlice('\n')
				// if err != nil {
				// 	log.Printf("Failed to read payload of contact info packet!%s\n", err)
				// 	continue
				// }

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

				forward, err := packets.SerializePacket(header, contact)
				if err != nil {
					log.Println("Failed to serialize contact packet")
					continue
				}
				for _, v := range contact.ContactIds {
					forwardCon, ex := db[v]
					if !ex {
						log.Printf("Contact %du not online\n", v)
						// database.SaveContactInfoToFile(contact, fmt.Sprint(v)+".json")
						continue
					}
					forwardCon.Write(forward)
					log.Printf("Forwarded contact info to %du\n", v)
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

				if index := MessageIDExists(header.MessageId, lastMessageId); index > -1 {
					log.Printf("MessageID has already been seen!")
					stored := lastMessageId[index]
					if !AlreadySeen(header.Category, header.Type, stored.Type) {
						delete(db, id)
						return
					} else {
						// This packet is a duplicate, continue
						// _, _ = reader.ReadSlice('\n')
						continue
					}
				}

				AddMessageId(header.MessageId, header.Category, header.Type, &count, &lastMessageId)

				// payload, err := reader.ReadSlice('\n')
				// if err != nil {
				// 	log.Printf("Failed to read payload of contact info packet!%s\n", err)
				// 	continue
				// }

				var text packets.Text
				text, err = packets.DeseralizePacket[packets.Text](payload)
				log.Printf("Got \"%s\" from \"%d\" forwarding to \"%d\"\n", text.Message, header.UserId, text.ContactUserId)
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
				log.Printf("Wrote textAck (%s) back to %d\n", hex.Dump(ack), header.UserId)

				// Continue with forwarding the text
				forwardCon, ex := db[text.ContactUserId]
				if !ex {
					log.Printf("Contact %d not online", text.ContactUserId)
					database.SaveMessagesToFile(text, header.UserId, fmt.Sprint(text.ContactUserId)+".json")
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

				// Difficult to check. Contains the same ID as the text, so cannot really check twice
				// if MessageIDExists(header.MessageId, lastMessageId) > 0 {
				// 	log.Printf("MessageID in text ack has already been seen!")
				// 	// not going to forward
				// 	continue
				// }
				// AddMessageId(header.MessageId, &count, &lastMessageId)

				// payload, err := reader.ReadSlice('\n')
				// if err != nil {
				// 	log.Printf("Failed to read payload of contact info packet!%s\n", err)
				// 	continue
				// }

				var textAck packets.TextAck
				textAck, err = packets.DeseralizePacket[packets.TextAck](payload)
				if err != nil {
					log.Printf("Failed to deserialize text ack!")
					// We cannot decode, so also not store the answer...
					// database.SaveTextAckToFile(textAck, fmt.Sprint(textAck.ContactUserId)+".json")
					continue
				}

				// Lookup the contacted user and forward
				forwardCon, ex := db[textAck.ContactUserId]
				if !ex {
					log.Printf("Contact %d not online", textAck.ContactUserId)
					// database.SaveTextAckToFile(textAck, fmt.Sprint(textAck.ContactUserId)+".json")
					continue
				}
				forward, err := packets.SerializePacket(header, textAck)
				if err != nil {
					log.Printf("Failed to create forward packet!")
					continue
				}
				forwardCon.Write(forward)
			case packets.D_FILE_INFO:
				log.Printf("Received file information")

				_, ex := db[id]
				if !ex {
					log.Printf("User %d not logged in. Cannot process anything without registration and login!", id)
					return
				}

				if index := MessageIDExists(header.MessageId, lastMessageId); index > -1 {
					log.Printf("MessageID has already been seen!")
					stored := lastMessageId[index]
					if !AlreadySeen(header.Category, header.Type, stored.Type) {
						delete(db, id)
						return
					} else {
						// This packet is a duplicate, continue
						// _, _ = reader.ReadSlice('\n')
						continue
					}
				}

				AddMessageId(header.MessageId, header.Category, header.Type, &count, &lastMessageId)

				// payload, err := reader.ReadSlice('\n')
				// if err != nil {
				// 	log.Printf("Failed to read payload of contact info packet!%s\n", err)
				// 	continue
				// }

				var fileInfo packets.FileInfo
				fileInfo, err = packets.DeseralizePacket[packets.FileInfo](payload)
				log.Printf("Got \"%s\" forwarding to \"%d\"\n", fileInfo.FileName, fileInfo.ContactUserId)
				if err != nil {
					log.Println("Failed to deserialize text packet")
					delete(db, id)
					return
				}

				forwardCon, ex := db[fileInfo.ContactUserId]
				if !ex {
					log.Printf("Contact %d not online", fileInfo.ContactUserId)
					// database.SaveTextAckToFile(textAck, fmt.Sprint(textAck.ContactUserId)+".json")
					continue
				}
				forward, err := packets.SerializePacket(header, fileInfo)
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
		case "Username":
			log.Printf("Username: %s", option.Options[0].Value)
		default:
			log.Printf("Unknown contact option type \"%s\"", v.Type)
			return errors.New("unknown contact type")
		}
	}
	return nil
}
