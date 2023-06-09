package packets

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"log"
	"math"
	"strings"
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
	D_TEXT      = 1
	D_TEXT_ACK  = 2
	D_FILE_INFO = 3
	D_FILE_HAVE = 4
	D_FILE      = 5
	D_FILE_ACK  = 6
)

type Packet interface {
	Create | Search | Contact | ContactList | ContactOption | Text | TextAck | Header | ContactInfo | FileInfo | FileHave
}

type Header struct {
	Category  byte
	Type      byte
	UserId    uint32
	MessageId uint32
}

type Create struct {
	Username string
}

type Search struct {
	UserIdentifier string
}

type Contact struct {
	UserId   uint32
	Username string
}

type ContactList struct {
	Contacts []Contact
}

type Option struct {
	Type  string
	Value string
}

type ContactOption struct {
	ContactUserId uint32
	Options       []Option
}

type ContactInfo struct {
	Username    string
	ContactIds  []uint32
	ImageBytes  uint32
	ImageFormat string
	Image       []byte
}

type Text struct {
	ContactUserId uint32
	Timestamp     string
	Message       string
}

type TextAck struct {
	ContactUserId uint32
	Timestamp     string
}

type FileInfo struct {
	ContactUserId    uint32
	Timestamp        string
	FileType         string
	FileName         string
	FileLength       uint32
	Compression      string
	CompressedLength uint32
	FileHash         int64
}

type FileHave struct {
	FileOffset uint64
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
		case D_FILE_INFO:
			log.Print("File Info")
			return CAT_DATA, D_FILE_INFO, nil
		case D_FILE_HAVE:
			log.Print("File Have")
			return CAT_DATA, D_FILE_HAVE, nil
		case D_FILE:
			log.Print("File")
			return CAT_DATA, D_FILE, nil
		case D_FILE_ACK:
			log.Print("File Ack")
			return CAT_DATA, D_FILE_ACK, nil
		default:
			log.Printf("Unknown type %d", typ)
			return NONE, NONE, errors.New("unknown type")
		}
	default:
		log.Print("Unknown category")
		return NONE, NONE, errors.New("unknown category")
	}
}

func SerializePacket(header Header, content any) ([]byte, error) {
	// Convert the json string into byte form and add the packet length
	headerBuffer := new(bytes.Buffer)
	err := binary.Write(headerBuffer, binary.BigEndian, header)
	if err != nil {
		log.Printf("Failed to encode given header to binary")
		return nil, err
	}
	if content != nil {
		payload, err := json.Marshal(content)
		if err != nil {
			return nil, err
		}
		if len(payload) > math.MaxInt32 {
			log.Println("Packet longer than 4 GB are not supported")
			return nil, errors.New("Packet too long")
		}
		packet := append(headerBuffer.Bytes(), payload...)
		// We need to add the newline for the other end to be able to scan for this
		packet = append(packet, []byte("\n")...)
		return packet, nil
	}
	// log.Printf("Packet: %02x", buffer)
	return headerBuffer.Bytes(), nil
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

func CreateLogin(userId uint32, messageId uint32) Header {
	header := Header{
		Category:  CAT_CONTACT,
		Type:      CON_LOGIN,
		UserId:    userId,
		MessageId: messageId,
	}
	// login := Login{
	// 	Category:  CAT_CONTACT,
	// 	Type:      CON_LOGIN,
	// 	UserId:    userId,
	// 	MessageId: messageId,
	// }
	return header
}

func CreateAccount(messageId uint32, username string) (Header, Create) {
	header := Header{
		Category:  CAT_CONTACT,
		Type:      CON_CREATE,
		UserId:    0,
		MessageId: messageId,
	}
	create := Create{
		Username: username,
	}
	return header, create
}

func CreateContactInfo(userId uint32, messageId uint32, username string, image []byte, contactList []uint32) (Header, ContactInfo) {
	// Divisor should be sized so that the MTU is kept
	// divisor := 1000.
	// packets := int(math.Ceil(float64(len(image)) / divisor))
	// log.Printf("# Packets: %d", packets)
	// contactInfoPackets := make([]ContactInfo, packets)
	// for i := 0; i < packets; i++ {
	header := Header{
		Category:  CAT_CONTACT,
		Type:      CON_CONTACT_INFO,
		UserId:    userId,
		MessageId: messageId,
	}
	contactInfoStruct := ContactInfo{
		Username:    username,
		ContactIds:  contactList,
		ImageBytes:  uint32(len(image)),
		ImageFormat: "jpeg",
		Image:       image,
	}
	// contactInfoPackets[i] = contactInfoStruct
	// }
	// return contactInfoPackets
	return header, contactInfoStruct
}

func CreateText(userId uint32, messageId uint32, contactId uint32, text string) (Header, Text) {
	header := Header{
		Category:  CAT_DATA,
		Type:      D_TEXT,
		UserId:    userId,
		MessageId: messageId,
	}
	textStruct := Text{
		ContactUserId: contactId,
		Timestamp:     time.Now().Format(time.RFC3339),
		Message:       text,
	}
	return header, textStruct
}

func CreateTextAck(userId uint32, messageId uint32, contactId uint32) (Header, TextAck) {
	header := Header{
		Category:  CAT_DATA,
		Type:      D_TEXT_ACK,
		UserId:    userId,
		MessageId: messageId,
	}
	ack := TextAck{
		ContactUserId: contactId,
		Timestamp:     time.Now().Format("mm:yyyy"),
	}
	return header, ack
}

func CreateContactInfoAck(userId uint32, messageId uint32) Header {
	header := Header{
		Category:  CAT_CONTACT,
		Type:      CON_CONTACT_ACK,
		UserId:    userId,
		MessageId: messageId,
	}
	return header
}

func CreateContactList(userId uint32, messageId uint32, contacts []Contact) (Header, ContactList) {
	log.Println("Creating contact list packet")
	header := Header{
		Category:  CAT_CONTACT,
		Type:      CON_CONTACTS,
		UserId:    userId,
		MessageId: messageId,
	}
	contactList := ContactList{
		Contacts: contacts,
	}
	return header, contactList
}

func ConvertContactInfoToClientContactInfo(contactInfo ContactInfo) (ContactInfo, error) {
	log.Print("Converting the contact information from client to server format, removing all contact IDs")
	// TODO: Maybe leave the client ID inside (no benefit for now)
	contactInfo.ContactIds = make([]uint32, 0)
	return contactInfo, nil
}

func CreateFileInfo(userId uint32, messageId uint32, fileName string, fileLength uint32, fileHash int64, fileCompression string, compressionLength uint32) (Header, FileInfo) {
	log.Print("Creating file info")
	header := Header{
		Category:  CAT_DATA,
		Type:      D_FILE_INFO,
		UserId:    userId,
		MessageId: messageId,
	}
	fileType := "FILE"
	if strings.HasSuffix(fileName, ".png") || strings.HasSuffix(fileName, ".jpeg") {
		fileType = "IMAGE"
	}
	if strings.HasSuffix(fileName, ".mp4") || strings.HasSuffix(fileName, ".wmv") {
		fileType = "VIDEO"
	}
	fileInfo := FileInfo{
		FileName:         fileName,
		FileType:         fileType,
		FileLength:       fileLength,
		Compression:      fileCompression,
		CompressedLength: compressionLength,
		FileHash:         fileHash,
	}
	return header, fileInfo
}

func CreateFileHave(userId uint32, messageId uint32, offset uint64) (Header, FileHave) {
	header := Header{
		Category:  CAT_DATA,
		Type:      D_FILE_HAVE,
		UserId:    userId,
		MessageId: messageId,
	}
	fileHave := FileHave{
		FileOffset: offset,
	}
	return header, fileHave
}

func CreateFile(userId uint32, messageId uint32) Header {
	header := Header{
		Category:  CAT_DATA,
		Type:      D_FILE,
		UserId:    userId,
		MessageId: messageId,
	}
	return header
}

func CreateFileAck(userId uint32, messageId uint32) Header {
	header := Header{
		Category:  CAT_DATA,
		Type:      D_FILE_ACK,
		UserId:    userId,
		MessageId: messageId,
	}
	return header
}
