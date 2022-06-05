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
	if len(packet) > 65536 {
		log.Println("Packet longer than 16 kB are not supported")
		return nil, errors.New("Packet too long")
	}
	size := uint16(len(packet))
	buffer := make([]byte, size+2)
	binary.BigEndian.PutUint16(buffer, size)
	copied := copy(buffer[2:], packet)
	if copied < len(packet) {
		log.Printf("Something went wrong during packet creation! Should copy %d, but copied only %d", size, copied)
		return nil, errors.New("Failed to copy all data")
	}
	log.Printf("Packet: %02x", buffer)
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
			log.Println("Failed to read information from client!")
			continue
		}

		log.Printf("Read %d bytes size information from the client", read)

		size := binary.BigEndian.Uint16(sizeBuf)

		log.Printf("Expecting %d bytes of data", size)

		// ADAPT SIZE INFORMATION IF SIZE FIELD CHANGES
		contentBuf := make([]byte, size-2)
		read, err = reader.Read(contentBuf)

		if err != nil {
			if read == 0 {
				log.Println("Connection closed by remote host")
				return
			}
			log.Println("Failed to read information from the client!")
			continue
		}

		category, typ, err := packets.PacketType(contentBuf)
		// packet, err := packets.DeseralizePacket(contentBuf)

		if err != nil {
			log.Printf("Got unknown packet type! %s", err.Error())
			log.Printf("Closing client connection...")
			return
		}

		switch category {
		case packets.CAT_CONTACT:
			switch typ {
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
				// Sending back the ID to the client
				create.UserId = newUserId
				var encoded []byte
				encoded, err = json.Marshal(create)
				if err != nil {
					log.Printf("Failed to encoded json! %s", err)
					return
				}
				connection.Write(encoded)
			case packets.CON_SEARCH:
				var search packets.Search
				search, err = packets.DeseralizePacket[packets.Search](contentBuf)
				users := database.SearchUsers(search.UserIdentifier)
				if len(users) == 0 {
					log.Printf("No users for identifier \"%s\" found", search.UserIdentifier)
				}
				contactList := packets.ContactList{
					Category:  packets.CAT_CONTACT,
					Type:      packets.CON_CONTACTS,
					UserId:    search.UserId,
					MessageId: search.MessageId + 1,
					Contacts:  users,
				}
				var packet []byte
				packet, err = json.Marshal(contactList)
				if err != nil {
					log.Printf("Failed to encode json! %s", err)
				}
				connection.Write(packet)
			case packets.CON_CONTACTS:
				// packet, err = packets.DeseralizePacket[packets.ContactList](contentBuf)
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
				HandleContactOption(option)
			default:
				log.Printf("Incorrect packet type: %d", typ)
				return
			}
		case packets.CAT_DATA:
			switch typ {
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
				forward, err := CreatePacket(contentBuf)
				user.Connection.Write(forward)
				textAck := packets.TextAck{
					Category:      packets.CAT_DATA,
					Type:          packets.D_TEXT_ACK,
					UserId:        text.UserId,
					MessageId:     text.UserId,
					ContactUserId: text.ContactUserId,
					Timestamp:     text.Timestamp,
					AckPart:       text.Part,
				}
				var ack []byte
				ack, err = json.Marshal(textAck)
				if err != nil {
					log.Printf("Failed to serialize json! %s", err)
					return
				}
				connection.Write(ack)
			case packets.D_TEXT_ACK:
				// packet, err = packets.DeseralizePacket[packets.TextAck](contentBuf)
				log.Println("Received text ack! Should never be received on the server side! Closing connection...")
				return
			default:
				log.Printf("Incorrect packet type %d", typ)
			}
		default:
			log.Printf("Incorrect packet category: %d", category)
		}

		if err != nil {
			log.Println("Could not parse packet! Closing connection to client!")
			return
		}

	}
}

func HandleContactOption(option packets.ContactOption) {
	for _, v := range option.Options {
		log.Printf("Option: {%s, %s}", v.Type, v.Value)
		switch v.Type {
		case "Add":
			break
		case "Remove":
			break
		default:
			log.Printf("Unknown contact option type \"%s\"", v.Type)
			return
		}
	}
}
