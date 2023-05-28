package apollon_test

import (
	"Loxias/configuration"
	"Loxias/server"
	"bytes"
	"encoding/binary"
	"errors"
	"log"
	"math/rand"
	"net"
	"testing"
	"time"

	"apollon.chat.com/packets"
)

func TestMain(m *testing.M) {
	StartServer()
	time.Sleep(100 * time.Millisecond)
	m.Run()
	server.Stop()
}

func StartServer() {
	configuration := configuration.Config{
		Secure:             false,
		ListenAddr:         "0.0.0.0",
		ListenPort:         "50000",
		SecureListenPort:   "50001",
		Logfile:            "server.log",
		ClearDatabase:      false,
		CertificateFile:    "resources/apollon.crt",
		CertificateKeyfile: "resources/apollon.key",
		DatabaseFile:       "../resources/test_database.json",
		DatabaseNoWrite:    true,
	}
	go server.Start(configuration)
}

func TestLogin(t *testing.T) {
	userId := uint32(0)
	// Create the login package and send it to the other end
	loginHeader := packets.CreateLogin(userId, rand.Uint32())
	packet, err := packets.SerializePacket(loginHeader, nil)
	if err != nil {
		log.Printf("Internal Failure while serializing the packet!")
		t.FailNow()
	}
	addr := "127.0.0.1" + ":" + "50000"
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("Failed to connect to the server!")
		t.FailNow()
	}
	defer conn.Close()
	wrote, err := conn.Write(packet)
	if err != nil {
		log.Printf("Failed to write to server!\n%s", err)
		t.FailNow()
	}
	if wrote != len(packet) {
		log.Printf("Failed to write all %d bytes to server! Wrote only %d", len(packet), wrote)
		t.FailNow()
	}

	// Waiting for server to close connection
	time.Sleep(100 * time.Millisecond)

	// Reading to check if the connection is available
	testBuffer := make([]byte, 10)
	read, err := conn.Read(testBuffer)
	if err == nil && read == len(testBuffer) {
		log.Print("Can still write to server! Connection should be closed by now!")
		t.FailNow()
	}

	// Testing with a non 0 User ID (but unknown)
	userId = uint32(1)
	// Create the login package and send it to the other end
	loginHeader = packets.CreateLogin(userId, rand.Uint32())
	packet, err = packets.SerializePacket(loginHeader, nil)
	if err != nil {
		log.Printf("Internal Failure while serializing the packet!")
		t.FailNow()
	}
	conn, err = net.Dial("tcp", addr)
	if err != nil {
		log.Printf("Failed to connect to the server!")
		t.FailNow()
	}
	defer conn.Close()
	wrote, err = conn.Write(packet)
	if err != nil {
		log.Printf("Failed to write to server!\n%s", err)
		t.FailNow()
	}
	if wrote != len(packet) {
		log.Printf("Failed to write all %d bytes to server! Wrote only %d", len(packet), wrote)
		t.FailNow()
	}
	// Waiting for server to close connection
	time.Sleep(100 * time.Millisecond)

	read, err = conn.Read(testBuffer)
	if err == nil && read == len(testBuffer) {
		log.Print("Can still write to server! Connection should be closed by now!")
		t.FailNow()
	}

	// Testing with a known User ID (after testing once)
	userId = uint32(1293812414)
	// Create the login package and send it to the other end
	loginHeader = packets.CreateLogin(userId, rand.Uint32())
	packet, err = packets.SerializePacket(loginHeader, nil)
	if err != nil {
		log.Printf("Internal Failure while serializing the packet!")
		t.FailNow()
	}
	conn, err = net.Dial("tcp", addr)
	if err != nil {
		log.Printf("Failed to connect to the server!")
		t.FailNow()
	}
	defer conn.Close()
	wrote, err = conn.Write(packet)
	if err != nil {
		log.Printf("Failed to write to server!\n%s", err)
		t.FailNow()
	}
	if wrote != len(packet) {
		log.Printf("Failed to write all %d bytes to server! Wrote only %d", len(packet), wrote)
		t.FailNow()
	}
	// Waiting for server to close connection
	time.Sleep(100 * time.Millisecond)

	// TODO: Check connection by sending text or packet that expects response
	// nullBuffer := make([]byte, 10)
	// read, err = conn.Read(nullBuffer)
	// if err != nil && read != len(nullBuffer) {
	// 	log.Print("Can still write to server! Connection should be closed by now!")
	// 	t.FailNow()
	// }
	// if wrote != len(packet) {
	// 	log.Print("Did not write whole packet to server!")
	// 	t.FailNow()
	// }
}

func TestCreateAccount(t *testing.T) {
	messageId := uint32(1293812414)
	username := "Neuer Nutzer"
	// Create the login package and send it to the other end
	createHeader, create := packets.CreateAccount(messageId, username)
	packet, err := packets.SerializePacket(createHeader, create)
	if err != nil {
		log.Printf("Internal Failure while serializing the packet!")
		t.FailNow()
	}
	addr := "127.0.0.1" + ":" + "50000"
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("Failed to connect to the server!")
		t.FailNow()
	}
	defer conn.Close()
	conn.Write(packet)

	// Now the important part: Expecting the answer with only a header containing our user ID != 0
	headerBuffer := make([]byte, 10)
	// Timeout after 2 seconds
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	read, err := conn.Read(headerBuffer)
	if err != nil {
		log.Printf("Failed to receive answer back!")
		t.FailNow()
	}
	if read != 10 {
		log.Printf("Failed to receive answer back!")
		t.FailNow()
	}
	// Convert header into struct form and check contents
	var ackHeader packets.Header
	newReader := bytes.NewReader(headerBuffer)
	err = binary.Read(newReader, binary.BigEndian, &ackHeader)
	if err != nil {
		log.Printf("Failed to decode header: %s", err)
		t.FailNow()
	}

	if ackHeader.MessageId != messageId {
		t.FailNow()
	}
	if ackHeader.UserId == 0 {
		t.FailNow()
	}
	if ackHeader.Category != packets.CAT_CONTACT {
		t.FailNow()
	}
	if ackHeader.Type != packets.CON_CREATE {
		t.FailNow()
	}
}

func TestSendingMessage(t *testing.T) {
	userId := uint32(1293812414)
	contactId := uint32(3718291512)
	messageId := rand.Uint32()
	text := "Testing sending message"
	// First only sending text (MUST FAIL!)
	createHeader, create := packets.CreateText(userId, messageId, contactId, text)
	textPacket, err := packets.SerializePacket(createHeader, create)
	if err != nil {
		log.Printf("Internal Failure while serializing the packet!")
		t.FailNow()
	}
	addr := "127.0.0.1" + ":" + "50000"
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("Failed to connect to the server!")
		t.FailNow()
	}
	defer conn.Close()
	conn.Write(textPacket)

	time.Sleep(100 * time.Millisecond)

	// Expecting to fail!
	headerBuffer := make([]byte, 10)
	// Timeout after 2 seconds
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	read, err := conn.Read(headerBuffer)
	if err == nil && read == len(headerBuffer) {
		log.Printf("Received answer back!")
		t.FailNow()
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		log.Printf("Timeout instead of connection closed!")
		t.FailNow()
	}

	// Now make it correctly. Sending login + text
	userId = uint32(1293812414)
	// Create the login package and send it to the other end
	loginHeader := packets.CreateLogin(userId, rand.Uint32())
	packet, err := packets.SerializePacket(loginHeader, nil)
	if err != nil {
		log.Printf("Internal Failure while serializing the packet!")
		t.FailNow()
	}
	conn, err = net.Dial("tcp", addr)
	if err != nil {
		log.Printf("Failed to connect to the server!")
		t.FailNow()
	}
	defer conn.Close()
	wrote, err := conn.Write(packet)
	if err != nil {
		log.Printf("Failed to write to server!\n%s", err)
		t.FailNow()
	}
	if wrote != len(packet) {
		log.Printf("Failed to write all %d bytes to server! Wrote only %d", len(packet), wrote)
		t.FailNow()
	}
	// Waiting for server to close connection
	time.Sleep(100 * time.Millisecond)

	conn.Write(textPacket)

	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	read, err = conn.Read(headerBuffer)
	if err != nil && errors.As(err, &netErr) && netErr.Timeout() {
		log.Printf("Timeout instead of connection closed!")
		t.FailNow()
	}
	if err != nil || read != len(headerBuffer) {
		log.Printf("Failed to receive answer!")
		t.FailNow()
	}
	var ackHeader packets.Header
	newReader := bytes.NewReader(headerBuffer)
	err = binary.Read(newReader, binary.BigEndian, &ackHeader)
	if err != nil {
		log.Printf("Failed to decode header: %s", err)
		t.FailNow()
	}

	if ackHeader.Category != packets.CAT_DATA {
		t.FailNow()
	}
	if ackHeader.Type != packets.D_TEXT_ACK {
		t.FailNow()
	}
	if ackHeader.UserId != userId {
		t.FailNow()
	}
	if ackHeader.MessageId != messageId {
		t.FailNow()
	}
}
