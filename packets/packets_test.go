package packets_test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"anzu.cloudsheeptech.com/packets"
)

func TestCreationPacket(t *testing.T) {
	id := uint32(1234)
	messageID := uint32(4321)
	header := packets.CreateLogin(id, messageID)

	if header.Category != packets.CAT_CONTACT {
		t.Fail()
	}

	if header.Type != packets.CON_LOGIN {
		t.Fail()
	}

	if header.UserId != id {
		t.Fail()
	}

	if header.MessageId != messageID {
		t.Fail()
	}

	headerRaw, _ := packets.SerializePacket(header, nil)
	raw := []byte{0x01, 0x05, 0x00, 0x00, 0x04, 0xD2, 0x00, 0x00, 0x10, 0xE1}

	if !reflect.DeepEqual(headerRaw, raw) {
		fmt.Printf("Headers not equal:\nExpected: %s\nGot: %s\n", hex.Dump(raw), hex.Dump(headerRaw))
		t.Fail()
	}
}

func TestAccountPacket(t *testing.T) {
	username := "Cloudsheep"
	messageID := uint32(4321)
	header, create := packets.CreateAccount(messageID, username)

	if header.Category != packets.CAT_CONTACT {
		t.Fail()
	}
	if header.Type != packets.CON_CREATE {
		t.Fail()
	}
	if header.UserId != 0 {
		t.Fail()
	}
	if header.MessageId != messageID {
		t.Fail()
	}
	if create.Username != username {
		fmt.Printf("Expected: %s, Got: %s\n", username, create.Username)
		t.Fail()
	}

	headerRaw, _ := packets.SerializePacket(header, nil)
	raw := []byte{0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0xE1}

	if !reflect.DeepEqual(headerRaw, raw) {
		fmt.Printf("Headers not equal:\nExpected: %s\nGot: %s\n", hex.Dump(raw), hex.Dump(headerRaw))
		t.Fail()
	}
}

func TestContactInfoPacket(t *testing.T) {
	id := uint32(1234)
	messageID := uint32(4321)
	username := "Cloudsheep"
	image := []byte{1, 2, 3, 4}
	contacts := []uint32{16, 32, 64}
	header, contact := packets.CreateContactInfo(id, messageID, username, image, contacts)

	if header.Category != packets.CAT_CONTACT {
		t.Fail()
	}
	if header.Type != packets.CON_CONTACT_INFO {
		t.Fail()
	}
	if header.UserId != id {
		t.Fail()
	}
	if header.MessageId != messageID {
		t.Fail()
	}

	if contact.Username != username {
		fmt.Printf("Expected: %s, Got: %s\n", username, contact.Username)
		t.Fail()
	}
	if !reflect.DeepEqual(contact.ContactIds, contacts) {
		fmt.Printf("Contact IDs do not match!\n")
		t.Fail()
	}
	if !reflect.DeepEqual(contact.Image, image) {
		fmt.Printf("Image bytes do not match!")
		t.Fail()
	}
	if contact.ImageBytes != uint32(len(image)) {
		fmt.Printf("Length of image does not match!\n")
		t.Fail()
	}

	headerRaw, _ := packets.SerializePacket(header, nil)
	raw := []byte{0x01, 0x06, 0x00, 0x00, 0x04, 0xD2, 0x00, 0x00, 0x10, 0xE1}
	if !reflect.DeepEqual(headerRaw, raw) {
		fmt.Printf("Headers not equal:\nExpected: %s\nGot: %s\n", hex.Dump(raw), hex.Dump(headerRaw))
		t.Fail()
	}
}

func TestText(t *testing.T) {
	id := uint32(1234)
	messageID := uint32(4321)
	contactID := uint32(9988)
	message := "Testing the text packet"
	header, text := packets.CreateText(id, messageID, contactID, message)

	if header.Category != packets.CAT_DATA {
		t.Fail()
	}
	if header.Type != packets.D_TEXT {
		t.Fail()
	}
	if header.UserId != id {
		t.Fail()
	}
	if header.MessageId != messageID {
		t.Fail()
	}
	headerRaw, _ := packets.SerializePacket(header, nil)
	raw := []byte{0x02, 0x01, 0x00, 0x00, 0x04, 0xD2, 0x00, 0x00, 0x10, 0xE1}
	if !reflect.DeepEqual(headerRaw, raw) {
		fmt.Printf("Headers not equal:\nExpected: %s\nGot: %s\n", hex.Dump(raw), hex.Dump(headerRaw))
		t.Fail()
	}
	if text.Message != message {
		fmt.Printf("Message does not match!\n")
		t.Fail()
	}
	if text.ContactUserId != contactID {
		t.Fail()
	}
	if text.Timestamp == "" {
		t.Fail()
	}
}

func TestTextAckPacket(t *testing.T) {
	id := uint32(1234)
	messageID := uint32(4321)
	contactID := uint32(9988)
	header, ack := packets.CreateTextAck(id, messageID, contactID)
	if header.Category != packets.CAT_DATA {
		t.Fail()
	}
	if header.Type != packets.D_TEXT_ACK {
		t.Fail()
	}
	if header.UserId != id {
		t.Fail()
	}
	if header.MessageId != messageID {
		t.Fail()
	}
	headerRaw, _ := packets.SerializePacket(header, nil)
	raw := []byte{0x02, 0x02, 0x00, 0x00, 0x04, 0xD2, 0x00, 0x00, 0x10, 0xE1}
	if !reflect.DeepEqual(headerRaw, raw) {
		fmt.Printf("Headers not equal:\nExpected: %s\nGot: %s\n", hex.Dump(raw), hex.Dump(headerRaw))
		t.Fail()
	}
	if ack.ContactUserId != contactID {
		t.Fail()
	}
	if ack.Timestamp == "" {
		t.Fail()
	}
}

func TestContactListPacket(t *testing.T) {
	id := uint32(1234)
	messageID := uint32(4321)
	contacts := []packets.Contact{{1234, "Test"}, {4321, "Haha"}}
	header, list := packets.CreateContactList(id, messageID, contacts)
	if header.Category != packets.CAT_CONTACT {
		t.Fail()
	}
	if header.Type != packets.CON_CONTACTS {
		t.Fail()
	}
	if header.UserId != id {
		t.Fail()
	}
	if header.MessageId != messageID {
		t.Fail()
	}
	headerRaw, _ := packets.SerializePacket(header, nil)
	raw := []byte{0x01, 0x03, 0x00, 0x00, 0x04, 0xD2, 0x00, 0x00, 0x10, 0xE1}
	if !reflect.DeepEqual(headerRaw, raw) {
		fmt.Printf("Headers not equal:\nExpected: %s\nGot: %s\n", hex.Dump(raw), hex.Dump(headerRaw))
		t.Fail()
	}
	if !reflect.DeepEqual(list.Contacts, contacts) {
		fmt.Printf("Contact list is not the same!\n")
		t.Fail()
	}
}

func TestFileInfoPacket(t *testing.T) {
	fileName := "image.png"
	fileType := "IMAGE"
	fileLength := 20120313
	compression := "None"
	compressionLength := 20120313
	fileHash := 1293102301203

	userId := uint32(1234)
	messageId := uint32(4321)

	header, info := packets.CreateFileInfo(userId, messageId, fileName, uint32(fileLength), int64(fileHash), compression, uint32(compressionLength))
	if header.Category != packets.CAT_DATA {
		t.FailNow()
	}
	if header.Type != packets.D_FILE_INFO {
		t.FailNow()
	}
	if header.UserId != userId {
		t.FailNow()
	}
	if header.MessageId != messageId {
		t.FailNow()
	}
	if info.FileType != fileType {
		t.FailNow()
	}
	headerRaw, _ := packets.SerializePacket(header, nil)
	raw := []byte{0x02, 0x03, 0x00, 0x00, 0x04, 0xD2, 0x00, 0x00, 0x10, 0xE1}
	if !reflect.DeepEqual(headerRaw, raw) {
		fmt.Printf("Headers not equal:\nExpected: %s\nGot: %s\n", hex.Dump(raw), hex.Dump(headerRaw))
		t.Fail()
	}
	serializedRaw, _ := json.Marshal(info)
	serialized := string(serializedRaw)
	compareJson := fmt.Sprintf("{\"FileType\":\"IMAGE\",\"FileName\":\"%s\",\"FileLength\":%d,\"Compression\":\"%s\",\"CompressedLength\":%d,\"FileHash\":%d}", fileName, fileLength, compression, compressionLength, fileHash)
	if serialized != compareJson {
		fmt.Printf("Serialized and expected do not match!\n%s\n%s\n", serialized, compareJson)
		t.FailNow()
	}
}

func TestFileHavePacket(t *testing.T) {
	userId := uint32(1234)
	messageId := uint32(4321)

	offset := 123123
	header, fileHave := packets.CreateFileHave(userId, messageId, uint64(offset))
	if header.Category != packets.CAT_DATA {
		t.FailNow()
	}
	if header.Type != packets.D_FILE_HAVE {
		t.FailNow()
	}
	if header.UserId != userId {
		t.FailNow()
	}
	if header.MessageId != messageId {
		t.FailNow()
	}
	if fileHave.FileOffset != uint64(offset) {
		t.FailNow()
	}
	headerRaw, _ := packets.SerializePacket(header, nil)
	raw := []byte{0x02, 0x04, 0x00, 0x00, 0x04, 0xD2, 0x00, 0x00, 0x10, 0xE1}
	if !reflect.DeepEqual(headerRaw, raw) {
		fmt.Printf("Headers not equal:\nExpected: %s\nGot: %s\n", hex.Dump(raw), hex.Dump(headerRaw))
		t.Fail()
	}
	serializedRaw, _ := json.Marshal(fileHave)
	serialized := string(serializedRaw)
	compareJson := fmt.Sprintf("{\"FileOffset\":%d}", offset)
	if serialized != compareJson {
		fmt.Printf("Serialized and expected do not match!\n%s\n%s\n", serialized, compareJson)
		t.FailNow()
	}
}

func TestFilePacket(t *testing.T) {
	userId := uint32(1234)
	messageId := uint32(4321)

	header := packets.CreateFile(userId, messageId)
	if header.Category != packets.CAT_DATA {
		t.FailNow()
	}
	if header.Type != packets.D_FILE {
		t.FailNow()
	}
	if header.UserId != userId {
		t.FailNow()
	}
	if header.MessageId != messageId {
		t.FailNow()
	}
	headerRaw, _ := packets.SerializePacket(header, nil)
	raw := []byte{0x02, 0x05, 0x00, 0x00, 0x04, 0xD2, 0x00, 0x00, 0x10, 0xE1}
	if !reflect.DeepEqual(headerRaw, raw) {
		fmt.Printf("Headers not equal:\nExpected: %s\nGot: %s\n", hex.Dump(raw), hex.Dump(headerRaw))
		t.Fail()
	}
}

func TestFileAck(t *testing.T) {
	userId := uint32(1234)
	messageId := uint32(4321)

	header := packets.CreateFileAck(userId, messageId)
	if header.Category != packets.CAT_DATA {
		t.FailNow()
	}
	if header.Type != packets.D_FILE_ACK {
		t.FailNow()
	}
	if header.UserId != userId {
		t.FailNow()
	}
	if header.MessageId != messageId {
		t.FailNow()
	}
	headerRaw, _ := packets.SerializePacket(header, nil)
	raw := []byte{0x02, 0x06, 0x00, 0x00, 0x04, 0xD2, 0x00, 0x00, 0x10, 0xE1}
	if !reflect.DeepEqual(headerRaw, raw) {
		fmt.Printf("Headers not equal:\nExpected: %s\nGot: %s\n", hex.Dump(raw), hex.Dump(headerRaw))
		t.Fail()
	}
}
