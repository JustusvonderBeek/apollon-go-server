package packets

import (
	"encoding/json"
	"errors"
	"log"
	"time"
)

const NONE = 0

// Categories
const (
	CAT_CONTACT = 1
	CAT_DATA    = 2
)

// Contact types
const (
	CON_CREATE       = 1
	CON_SEARCH       = 2
	CON_CONTACTS     = 3
	CON_OPTION       = 4
	CON_LOGIN        = 5
	CON_CONTACT_INFO = 6
	CON_CONTACT_ACK  = 7
)

// Data types
const (
	D_TEXT     = 1
	D_TEXT_ACK = 2
)

type Packet interface {
	Create | Search | Contact | ContactList | ContactOption | Text | TextAck | Header | Login | ContactInfo | ContactAck
}

type Header struct {
	Category  byte
	Type      byte
	UserId    uint32
	MessageId uint32
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

type Login struct {
	Category  byte
	Type      byte
	UserId    uint32
	MessageId uint32
}

type ContactInfo struct {
	Category    byte
	Type        byte
	UserId      uint32
	MessageId   uint32
	Username    string
	ContactIds  []uint32
	ImageBytes  uint32
	ImageFormat string
}

type ContactAck struct {
	Category  byte
	Type      byte
	UserId    uint32
	MessageId uint32
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

func PacketType(packet []byte) (int, int, error) {
	valid := json.Valid(packet)
	if !valid {
		log.Print("Incorrect JSON")
		return NONE, NONE, errors.New("invalid JSON")
	}

	var parsed map[string]interface{}
	err := json.Unmarshal(packet, &parsed)
	if err != nil {
		log.Printf("Failed to parse packet: %s", err.Error())
		return NONE, NONE, errors.New("failed to parse JSON")
	}

	cat := parsed["Category"].(float64)
	category := int(cat)
	t := parsed["Type"].(float64)
	typ := int(t)

	switch category {
	case CAT_CONTACT:
		log.Print("Contact")
		switch typ {
		case CON_CREATE:
			log.Print("Create")
			return CAT_CONTACT, CON_CREATE, nil
		case CON_SEARCH:
			log.Print("Search")
			return CAT_CONTACT, CON_SEARCH, nil
		case CON_CONTACTS:
			log.Print("Contacts")
			return CAT_CONTACT, CON_CONTACTS, nil
		case CON_OPTION:
			log.Print("Option")
			return CAT_CONTACT, CON_OPTION, nil
		case CON_LOGIN:
			log.Print("Login")
			return CAT_CONTACT, CON_LOGIN, nil
		case CON_CONTACT_INFO:
			log.Print("Contact Information")
			return CAT_CONTACT, CON_CONTACT_INFO, nil
		case CON_CONTACT_ACK:
			log.Print("Info")
			return CAT_CONTACT, CON_CONTACT_ACK, nil
		default:
			log.Printf("Unknown type %d", typ)
			return NONE, NONE, errors.New("unknown type")
		}
	case CAT_DATA:
		log.Print("Data")
		switch typ {
		case D_TEXT:
			log.Print("Text")
			return CAT_DATA, D_TEXT, nil
		case D_TEXT_ACK:
			log.Print("Text Ack")
			return CAT_DATA, D_TEXT_ACK, nil
		default:
			log.Printf("Unknown type %d", typ)
			return NONE, NONE, errors.New("unknown type")
		}
	default:
		log.Print("Unknown category")
		return NONE, NONE, errors.New("unknown category")
	}
}

func DeseralizePacket[T Packet](packet []byte) (T, error) {
	// log.Printf("Got packet:\n%s", string(packet))
	// log.Printf("Got packet:\n%s\n%02x", string(packet), packet)

	valid := json.Valid(packet)
	if !valid {
		log.Printf("The received packet is not valid JSON!")
		return *new(T), errors.New("invalid JSON")
	}

	var parsed T
	err := json.Unmarshal(packet, &parsed)
	if err != nil {
		log.Printf("Failed to parse text: %s", err.Error())
		return *new(T), errors.New("failed to parse JSON")
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

func CreateContactList(search Search, contacts []Contact) (ContactList, error) {
	log.Println("Creating contact list packet")
	contactList := ContactList{
		Category:  CAT_CONTACT,
		Type:      CON_CONTACTS,
		UserId:    search.UserId,
		MessageId: search.MessageId + 1,
		Contacts:  contacts,
	}
	return contactList, nil
}

func ConvertContactInfoToClientContactInfo(contactInfo ContactInfo) (ContactInfo, error) {
	log.Print("Converting the contact information from client to server format, removing all contact IDs")
	// TODO: Maybe leave the client ID inside (no benefit for now)
	contactInfo.ContactIds = make([]uint32, 0)
	return contactInfo, nil
}
