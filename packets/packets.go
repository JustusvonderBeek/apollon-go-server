package packets

import (
	"encoding/json"
	"fmt"
	"log"
)

type Text struct {
	Category      byte
	Type          byte
	UserId        uint32
	MessageId     uint32
	ContactUserId uint32
	// Timestamp time.Time
	Part    uint16
	Message string
}

func CreateText(name string) []byte {
	fmt.Println("Creating text packet")

	text := Text{1, 1, 123456, 1066, 9876, 0, "Das ist eine Nachricht"}

	packet, err := json.Marshal(text)

	if err != nil {
		log.Fatal("Failed to serialize json text packet")
	}

	return packet
}
