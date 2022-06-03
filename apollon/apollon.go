package apollon

import (
	"Loxias/packets"
	"bufio"
	"encoding/binary"
	"log"
	"math/rand"
	"net"
	"time"
)

type User struct {
	Username   string
	UserId     uint32
	connection net.Conn
}

var database = make(map[uint32]User)

func StoreInDatabase(user packets.Create, connection net.Conn) {
	log.Println("Storing user in database")

	newUser := User{
		Username:   user.Username,
		UserId:     user.UserId,
		connection: connection,
	}

	_, exists := database[user.UserId]

	if exists {
		log.Printf("User with ID %d already exists", user.UserId)
		return
	}

	database[user.UserId] = newUser

	log.Printf("Stored user %s with id %d", user.Username, user.UserId)
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
					_, exists := database[newUserId]
					if !exists {
						break
					}
					newUserId = rand.Uint32()
				}
				create.UserId = newUserId
				// Store new user in some sort of database
				StoreInDatabase(create, connection)
			case packets.CON_SEARCH:
				// packet, err = packets.DeseralizePacket[packets.Search](contentBuf)
			case packets.CON_CONTACTS:
				// packet, err = packets.DeseralizePacket[packets.ContactList](contentBuf)
			case packets.CON_OPTION:
				// packet, err = packets.DeseralizePacket[packets.ContactOption](contentBuf)
			default:
				log.Printf("Incorrect packet type: %d", typ)
				return
			}
		case packets.CAT_DATA:
			switch typ {
			case packets.D_TEXT:
				var packet packets.Text
				packet, err = packets.DeseralizePacket[packets.Text](contentBuf)
				log.Printf("Got %s", packet.Message)
			case packets.D_TEXT_ACK:
				// packet, err = packets.DeseralizePacket[packets.TextAck](contentBuf)
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
