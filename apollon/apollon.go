package apollon

import (
	"Loxias/packets"
	"bufio"
	"encoding/binary"
	"log"
	"net"
)

func HandleClient(connection net.Conn) {
	log.Println("Handling client...")

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

		cat, typ, err := packets.PacketType(contentBuf)

		if err != nil {
			log.Printf("Got unknown packet type!")
			return
		}

		switch cat {
		case packets.CAT_CONTACT:
			log.Println("Category: contact")
			switch typ {
			case packets.CON_CREATE:
				log.Println("Type: Create")
			case packets.CON_SEARCH:
				log.Println("Type: Search")
			case packets.CON_CONTACTS:
				log.Println("Type: Contacts")
			case packets.CON_OPTION:
				log.Println("Type: Option")
			}
		case packets.CAT_DATA:
			log.Println("Category: data")

		}

		// text, err := packets.DeseralizePacket[pType](contentBuf)

		if err != nil {
			log.Println("Could not parse packet! Closing connection to client!")
			return
		}

		// log.Printf("Found text: \"%s\"", text.Message)
	}
}
