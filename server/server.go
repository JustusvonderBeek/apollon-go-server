package server

import (
	"Loxias/apollon"
	"fmt"
	"log"
	"net"
)

func Start() {
	fmt.Println("Starting the server...")

	listen, err := net.Listen("tcp", ":50000")
	defer listen.Close()

	if err != nil {
		// Already closes the program (if not handled)
		log.Fatalf("Failed to connect to localhost: %s", err.Error())
	}

	for {
		log.Println("Waiting for connecting client...")
		conn, err := listen.Accept()

		if err != nil {
			log.Printf("Failed to accept client: %s", err.Error())
			continue
		}

		log.Println("Client accepted")
		// This method is generic enough (only one param, the net.Conn) so that many different functionalites can be used and implemented with this simple code snippet
		go apollon.HandleClient(conn)
	}
}
