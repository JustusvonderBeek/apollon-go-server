package packets

import (
	"encoding/json"
	"log"
	"time"
)

// Categories
const (
	CAT_NONE    = 0
	CAT_CONTACT = 1
	CAT_DATA    = 2
)

// Contact types
const (
	CON_NONE     = 0
	CON_CREATE   = 1
	CON_SEARCH   = 2
	CON_CONTACTS = 3
	CON_OPTION   = 4
)

// Data types
const (
	D_NONE     = 0
	D_TEXT     = 1
	D_TEXT_ACK = 2
)

type Packet interface {
	Create | Search | Contact | ContactList | ContactOption | Text | TextAck
}

type Create struct {
	Category  byte
	Type      byte
	UserId    uint32
	MessageId uint32
	Username  string
}

type Search struct {
	Category       byte
	Type           byte
	UserId         uint32
	MessageId      uint32
	UserIdentifier string
}

type Contact struct {
	UserId   uint32
	Username string
}

type ContactList struct {
	Category  byte
	Type      byte
	UserId    uint32
	MessageId uint32
	Contacts  []Contact
}

type Option struct {
	Type  string
	Value string
}

type ContactOption struct {
	Category      byte
	Type          byte
	UserId        uint32
	MessageId     uint32
	ContactUserId uint32
	Options       []Option
}

type Text struct {
	Category      byte
	Type          byte
	UserId        uint32
	MessageId     uint32
	ContactUserId uint32
	Timestamp     string
	Part          uint16
	Message       string
}

type TextAck struct {
	Category      byte
	Type          byte
	UserId        uint32
	MessageId     uint32
	ContactUserId uint32
	Timestamp     string
	AckPart       uint16
}

func PacketType(packet []byte) (uint, uint, error) {
	valid := json.Valid(packet)
	if valid != true {
		log.Println("The received packet does not contain a valid JSON format")
		// TODO: Create error
		return CAT_NONE, CON_NONE, nil
	}

	var parsed Text
	err := json.Unmarshal(packet, &parsed)
	if err != nil {
		log.Println("Failed to parse packet!")
		return CAT_NONE, CON_NONE, nil
	}
	c := uint(parsed.Category)
	t := uint(parsed.Type)
	return c, t, nil
}

func DeseralizePacket[V Packet](packet []byte) (V, error) {
	log.Printf("Got packet:\n%s", string(packet))
	// log.Printf("Got packet:\n%s\n%02x", string(packet), packet)

	valid := json.Valid(packet)
	if valid != true {
		log.Printf("The received packet is not valid JSON!")
		// TODO: create new error
		// return V, nil
	}

	var parsed V
	err := json.Unmarshal(packet, &parsed)
	if err != nil {
		log.Printf("Failed to parse text: %s", err.Error())
		// return nil, err
	}

	return parsed, nil
}

func CreateTextAck(messageId uint32, part uint16) TextAck {
	// TODO: Fix the hardcoded fields
	ack := TextAck{
		Category:      CAT_DATA,
		Type:          D_TEXT_ACK,
		MessageId:     messageId,
		UserId:        123456,
		ContactUserId: 123456,
		Timestamp:     time.Now().Format("mm:yyyy"),
		AckPart:       part,
	}
	return ack
}
