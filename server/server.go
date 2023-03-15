package server

import (
	"Loxias/apollon"
	"crypto/tls"
	"log"
	"net"
)

func Start(addr string, port string, secure bool) {
	log.Println("Starting the server...")

	compAddr := addr + ":" + port
	var listen net.Listener
	var err error
	if secure {
		log.Println("Loading server certificate and key")
		var cert tls.Certificate
		cert, err = tls.LoadX509KeyPair("./resources/apollon.crt", "./resources/server.key")
		if err != nil {
			log.Printf("Failed to load certificate: %s", err)
			return
		}

		config := tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		listen, err = tls.Listen("tcp", compAddr, &config)
	} else {
		listen, err = net.Listen("tcp", compAddr)
	}

	// Listing on the TLS socket
	if err != nil {
		// Already closes the program (if not handled)
		log.Fatalf("Failed to connect to localhost: %s", err.Error())
	}
	defer listen.Close()
	log.Printf("Listing on '%s'", compAddr)

	// database.ReadFromFile("database.json")
	// dbWriteChannel := make(chan apollontypes.User)
	// go database.UpdateDatabase(dbWriteChannel)

	db := make(map[uint32]net.Conn)
	// var db database.Database
	// if err != nil {
	// 	log.Printf("%s", err)
	// 	return
	// }

	for {
		log.Println("Waiting for connecting client...")
		conn, err := listen.Accept()

		if err != nil {
			log.Printf("Failed to accept client: %s", err.Error())
			continue
		}

		log.Printf("Client from %s accepted", conn.RemoteAddr().String())
		// This method is generic enough (only one param, the net.Conn) so that many different functionalites can be used and implemented with this simple code snippet
		go apollon.HandleClient(conn, db)
	}
}
