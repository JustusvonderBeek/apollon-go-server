package apollon

import (
	"Loxias/apollontypes"
	"Loxias/database"
	"Loxias/packets"
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"net"
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

func HandleClient(connection net.Conn) {
	log.Println("Handling client...")

	// Init the random number generator
	rand.Seed(time.Now().UnixNano())
	reader := bufio.NewReader(connection)

	for {

		sizeBuf := make([]byte, 2)
		read, err := reader.Read(sizeBuf)

		if err != nil {
			if read == 0 {
				log.Println("Connection closed by remote host")
				return
			}
			log.Println("Failed to read size information from client!")
			continue
		}

		log.Printf("Read %d bytes size information from the client", read)

		size := binary.BigEndian.Uint16(sizeBuf)

		log.Printf("Expecting %d bytes of data", size)
		if size <= 4 {
			log.Println("Keep alive packet")
			continue
		}

		// ADAPT SIZE INFORMATION IF SIZE FIELD CHANGES
		contentBuf := make([]byte, size-2)
		read, err = reader.Read(contentBuf)

		if err != nil {
			if read == 0 {
				log.Println("Connection closed by remote host")
				return
			}
			log.Println("Failed to read data from client!")
			continue
		}

		// category, typ, err := packets.PacketType(contentBuf)
		header, err := packets.DeseralizePacket[packets.Header](contentBuf)
		if err != nil {
			log.Println("Failed to extract header information from packet")
			// TODO: Is this rather due to an transmission error or because the client send wrong information? This should NORMALLY only happen if the client is malicous and sends incorrect data as the first part of the packet -> return
			return
		} else {
			user, err := database.GetUser(header.UserId)
			// Add the newly connected user to the online users
			// First check for not already added, second for registered user
			if err != nil && user.UserId != 0 {
				user.Connection = connection
				err = database.StoreUserInDatabase(user)
				log.Printf("User \"%d\" connected and online", user.UserId)
			} else {
				// Allows to switch the connection per packet
				user.Connection = connection
			}
		}

		switch header.Category {
		case packets.CAT_CONTACT:
			switch header.Type {
			case packets.CON_CREATE:
				var create packets.Create
				create, err = packets.DeseralizePacket[packets.Create](contentBuf)
				// Generate new user id
				newUserId := rand.Uint32()
				for {
					exists := database.IdExists(newUserId)
					if !exists {
						break
					}
					newUserId = rand.Uint32()
				}
				create.UserId = newUserId
				// Store new user in some sort of database
				err = database.StoreInDatabase(create, connection)
				if err != nil {
					// Failed to insert user into database
					return
				}
				database.SaveToFile("./database.json")
				// Sending back the ID to the client
				create.UserId = newUserId
				encoded, err := CreatePacket(create)
				if err != nil {
					log.Println("Failed to encode answer")
					continue
				}
				connection.Write(encoded)
			case packets.CON_SEARCH:
				var search packets.Search
				search, err = packets.DeseralizePacket[packets.Search](contentBuf)
				users := database.SearchUsers(search.UserIdentifier)
				if len(users) == 0 {
					log.Printf("No users for identifier \"%s\" found", search.UserIdentifier)
					// What to do in this case according to protocol?
					// I guess just send an empty list back
				}
				contactList, err := packets.CreateContactList(search, users)
				if err != nil {
					// Internal error, tried our best
					continue
				}
				packet, err := CreatePacket(contactList)
				if err != nil {
					continue
				}
				connection.Write(packet)
			case packets.CON_CONTACTS:
				// packet, err = packets.DeseralizePacket[packets.ContactList](contentBuf)
				// Should never be sent to the server
				log.Println("Received contact list! Should not be received on the server side! Closing connection!")
				return
			case packets.CON_OPTION:
				var option packets.ContactOption
				option, err = packets.DeseralizePacket[packets.ContactOption](contentBuf)
				if err != nil {
					log.Println("Failed to deserialize packet!")
					return
				}
				log.Println("Received contact option")
				err = HandleContactOption(option, connection)
				if err != nil {
					return
				}
			default:
				log.Printf("Incorrect packet type: %d", header.Type)
				return
			}
		case packets.CAT_DATA:
			switch header.Type {
			case packets.D_TEXT:
				var text packets.Text
				text, err = packets.DeseralizePacket[packets.Text](contentBuf)
				log.Printf("Got \"%s\" forwarding to \"%d\"", text.Message, text.ContactUserId)
				var user apollontypes.User
				user, err = database.GetUser(text.ContactUserId)
				if err != nil {
					log.Printf("%s! Closing connection...", err)
					// return
				}
				forward, err := CreatePacket(text)
				if err != nil {
					log.Printf("Failed to create forward packet!")
					continue
				}
				if user.Connection == nil {
					log.Printf("User %d currently not online", text.ContactUserId)
					continue
				}
				user.Connection.Write(forward)
				// Sending the ack! TODO: Normally this should only be send after the receiving side gives their OK - therefore no method is implemented in the packet library
				textAck := packets.TextAck{
					Category:      packets.CAT_DATA,
					Type:          packets.D_TEXT_ACK,
					UserId:        text.UserId,
					MessageId:     text.UserId,
					ContactUserId: text.ContactUserId,
					Timestamp:     text.Timestamp,
					AckPart:       text.Part,
				}
				ack, err := CreatePacket(textAck)
				connection.Write(ack)
			case packets.D_TEXT_ACK:
				// packet, err = packets.DeseralizePacket[packets.TextAck](contentBuf)
				log.Println("Received text ack! Should never be received on the server side! Closing connection...")
				return
			default:
				log.Printf("Incorrect packet type %d", header.Type)
			}
		default:
			log.Printf("Incorrect packet category: %d", header.Category)
		}

		if err != nil {
			log.Println("Could not parse packet! Closing connection to client!")
			return
		}

	}
}

func HandleContactOption(option packets.ContactOption, connection net.Conn) error {
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
				break
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
				break
			default:
				log.Printf("Unknown or incorrect contact option value \"%s\". Closing connection...", v.Value)
				return errors.New("Unknown contact value")
			}
			break
		default:
			log.Printf("Unknown contact option type \"%s\"", v.Type)
			return errors.New("Unknown contact type")
		}
	}
	return nil
}
