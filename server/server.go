package server

import (
	"crypto/tls"
	"log"
	"net"
	"os"

	"anzu.cloudsheeptech.com/apollon"
	"anzu.cloudsheeptech.com/configuration"
	"anzu.cloudsheeptech.com/database"
	"anzu.cloudsheeptech.com/restapi"
)

var listen net.Listener
var running bool
var db map[uint32]net.Conn

func Start(config configuration.Config) {
	log.Println("Starting the server...")

	database.SetDatabaseLocation(config.DatabaseFile)
	database.SetDatabaseNoWrite(config.DatabaseNoWrite)
	if config.ClearDatabase {
		database.Delete()
		log.Print("Cleared the database")
	}

	defaultAddr := config.ListenAddr + ":" + config.ListenPort
	secureAddr := config.ListenAddr + ":" + config.SecureListenPort
	// restAddr := config.ListenAddr + ":" + config.RestApiPort
	running = true
	var err error
	if config.Secure {
		log.Println("Loading server certificate and key")
		var cert tls.Certificate
		cert, err = tls.LoadX509KeyPair(config.CertificateFile, config.CertificateKeyfile)
		if err != nil {
			log.Printf("Failed to load certificate: %s", err)
			return
		}

		config := tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		listen, err = tls.Listen("tcp", secureAddr, &config)
		log.Printf("Listing on '%s'", secureAddr)
	} else {
		listen, err = net.Listen("tcp", defaultAddr)
		log.Printf("Listing on '%s'", defaultAddr)
	}

	// Listing on the TLS socket
	if err != nil {
		// Already closes the program (if not handled)
		log.Fatalf("Failed to connect to localhost: %s", err.Error())
	}
	defer listen.Close()

	// database.ReadFromFile("database.json")
	// dbWriteChannel := make(chan apollontypes.User)
	// go database.UpdateDatabase(dbWriteChannel)

	if config.RestApi {
		go restapi.RunRestApi()
	}

	db = make(map[uint32]net.Conn)
	forwardC := make(chan apollon.ForwardMessage, 20)
	newConnC := make(chan apollon.ConnMessage, 10)
	onlineC := make(chan apollon.OnlineMessage, 10)
	go apollon.ModifyOnlineUsers(newConnC, db)
	go apollon.ForwardingPackets(forwardC, db)
	go apollon.CheckUserOnline(onlineC, db)

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
			assertClose := err.(*net.OpError)
			if assertClose.Err == net.ErrClosed {
				log.Printf("Server was stopped...")
				break
			}
			continue
		}

		log.Printf("Client from %s accepted", conn.RemoteAddr().String())
		// This method is generic enough (only one param, the net.Conn) so that many different functionalites can be used and implemented with this simple code snippet
		go apollon.HandleClient(conn, forwardC, newConnC, onlineC)

		if !running {
			break
		}
	}
}

func Stop() {
	// Stop listening
	running = false
	listen.Close()
	// Closing all running clients the hard way (TODO: fix this)
	os.Exit(0)
}
