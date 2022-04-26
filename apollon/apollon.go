package apollon

import (
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

		contentBuf := make([]byte, size)
		read, err = reader.Read(contentBuf)

		if err != nil {
			if read == 0 {
				log.Println("Connection closed by remote host")
				return
			}
			log.Println("Failed to read information from the client!")
			continue
		}

	}
}
