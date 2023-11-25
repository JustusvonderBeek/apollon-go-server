package database

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"database/sql"

	"anzu.cloudsheeptech.com/apollontypes"
	"anzu.cloudsheeptech.com/packets"
)

var database = make(map[uint32]apollontypes.User)
var databaseFile = "database.json"
var directory = "./"
var noWrite = false

func UseSQL() {
	db, err := sql.Open("mysql", "anzuchat@localhost:1234@/anzuchat")
	if err != nil {
		panic(err)
	}
	// See "Important settings" section.
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
}

func PrintUser(user apollontypes.User) {
	log.Printf("{ Username: %s, UserId: %d, Connection: nil }", user.Username, user.UserId)
}

func PrintDatabase() {
	log.Println("--------------------------")
	for _, v := range database {
		PrintUser(v)
	}
	log.Println("--------------------------")
}

func CheckUser(user apollontypes.User) error {
	if user.Username == "" {
		log.Println("Cannot store user with empty username!")
		return errors.New("empty username")
	}

	if user.UserId == 0 {
		log.Println("Cannot store user with user id 0!")
		return errors.New("user ID 0")
	}

	return nil
}

func UpdateDatabase(channel chan apollontypes.User) {
	log.Println("Starting database thread")
	for {
		user := <-channel
		StoreUserInDatabase(user)
	}
}

func StoreInDatabase(userId uint32, username string) error {
	// log.Println("Storing user in database")

	newUser := apollontypes.User{
		Username: username,
		UserId:   userId,
	}

	return StoreUserInDatabase(newUser)
}

func StoreUserInDatabase(user apollontypes.User) error {
	ReadFromFile(databaseFile)
	// log.Println("Storing user in database")
	err := CheckUser(user)
	if err != nil {
		return err
	}
	_, exists := database[user.UserId]
	if exists {
		log.Printf("User with ID %d already exists", user.UserId)
		return errors.New("user already exists")
	}
	database[user.UserId] = user
	log.Printf("Stored user \"%s\" with id \"%d\"", user.Username, user.UserId)
	SaveToFile(databaseFile)
	return nil
}

func SearchUsers(search string) []packets.Contact {
	ReadFromFile(databaseFile)
	log.Printf("Searching for \"%s\"", search)

	var users []packets.Contact

	for _, v := range database {
		if strings.Contains(v.Username, search) {
			newUser := packets.Contact{
				UserId:   v.UserId,
				Username: v.Username,
			}
			users = append(users, newUser)
		}
	}

	return users
}

func SearchUserId(userId uint32) (packets.Contact, error) {
	err := ReadFromFile(databaseFile)
	if err != nil {
		log.Printf("Failed to read database from '%s'", databaseFile)
		return packets.Contact{}, errors.New("database not found")
	}
	for _, v := range database {
		if v.UserId == userId {
			foundUser := packets.Contact{
				UserId:   v.UserId,
				Username: v.Username,
			}
			return foundUser, nil
		}
	}
	return packets.Contact{}, errors.New("userID not found")
}

func GetUser(userId uint32) (apollontypes.User, error) {
	user, err := database[userId]

	if !err {
		log.Printf("Failed to retrieve user with id \"%d\"", userId)
		return user, errors.New("user not found")
	}

	return user, nil
}

func IdExists(id uint32) bool {
	ReadFromFile(databaseFile)
	log.Printf("Checking if ID %d exists", id)

	_, exists := database[id]

	return exists
}

func Clear() {
	database = make(map[uint32]apollontypes.User)
}

func Delete() {
	database = make(map[uint32]apollontypes.User)
	os.Create(databaseFile)
	// Maybe also delete all outstanding message files?
	dir, err := os.Open(directory)
	if err != nil {
		log.Fatal("Failed to find dir!")
	}
	fileInfo, err := dir.Readdir(-1)
	dir.Close()
	if err != nil {
		log.Fatal("Failed to get directory information")
	}
	for _, file := range fileInfo {
		if strings.HasSuffix(file.Name(), ".json") {
			os.Remove("./" + file.Name())
		}
	}
}

func SetDatabaseLocation(location string) {
	databaseFile = location
}

func SetDatabaseNoWrite(overwriteDatabase bool) {
	noWrite = overwriteDatabase
}

func ConvertToByte() ([]byte, error) {
	var content []byte
	var err error
	seperator := ","
	counter := 0
	for _, v := range database {
		// v.Connection = nil
		// v.Incoming = nil
		raw, err := json.Marshal(v)
		if err != nil {
			log.Printf("Failed to convert user %d to byte", v.UserId)
			// err = errors.New("not all users converted")
			continue
		}
		// The '...' signal that all elements of raw should be appended
		content = append(content, raw...)
		if counter+1 < len(database) {
			content = append(content, []byte(seperator)...)
		}
		counter++
	}
	return content, err
}

func SaveToFile(file string) error {
	log.Printf("Saving to \"%s\"", file)
	if noWrite {
		return nil
	}
	f, err := os.Create(file)
	if err != nil {
		log.Printf("Failed to create file \"%s\"", file)
		return err
	}
	// Don't forget to close the file
	defer f.Close()
	users, err := ConvertToByte()
	if err != nil {
		log.Println("Failed to save users to file!")
		return err
	}
	f.Write([]byte("["))
	f.Write(users)
	f.Write([]byte("]"))
	return nil
}

// TODO: Find a method to store and retrieve all types of JSON from a single or multiple files!
func SaveAnyToFile[T packets.Packet](any T, file string) error {
	log.Printf("Saving to \"%s\"", file)
	if noWrite {
		return nil
	}
	// Append if file exists
	content, err := os.ReadFile(file)
	var messages []T
	if err == nil {
		// File exists, append content
		err := json.Unmarshal(content, &messages)
		if err != nil {
			log.Printf("Failed to convert existing data to JSON. Messages will be overwritte")
		}
		messages = append(messages, any)
	} else {
		messages = append(messages, any)
	}
	f, err := os.Create(file)
	if err != nil {
		log.Printf("Failed to created file \"%s\"", file)
		return err
	}
	defer f.Close()
	encoded, err := json.Marshal(messages)
	if err != nil {
		log.Println("Failed to encoded messages")
		return err
	}
	_, err = f.Write(encoded)
	if err != nil {
		log.Println("Failed to write to file")
		return err
	}
	return nil
}

func SaveMessagesToFile(message packets.Text, sender uint32, file string) error {
	// Store the sender address instead of the contact ID
	// because that is what we don't know afterwards
	message.ContactUserId = sender
	return SaveAnyToFile(message, file)
}

func SaveTextAckToFile(ack packets.TextAck, file string) error {
	return SaveAnyToFile(ack, file)
}

func SaveContactInfoToFile(info packets.ContactInfo, file string) error {
	return SaveAnyToFile(info, file)
}

func SaveSearchAnswerToFile(search packets.Search, file string) error {
	return SaveAnyToFile(search, file)
}

func SaveContactOption(option packets.ContactOption, file string) error {
	return SaveAnyToFile(option, file)
}

func ReadFromFile(file string) error {
	Clear()
	log.Printf("Reading from \"%s\"", file)
	content, err := os.ReadFile(file)
	if err != nil {
		log.Println(err)
		return err
	}
	var data []apollontypes.User
	err = json.Unmarshal(content, &data)
	if err != nil {
		log.Println(err)
		return err
	}
	for _, v := range data {
		database[v.UserId] = v
	}
	return nil
}

func ReadMessagesFromFile(file string) ([]packets.Text, error) {
	log.Printf("Reading messages from \"%s\"", file)
	content, err := os.ReadFile(file)
	if err != nil {
		log.Printf("Failed to read messages from \"%s\"", file)
		return nil, err
	}
	var messages []packets.Text
	err = json.Unmarshal(content, &messages)
	if err != nil {
		log.Printf("Failed to convert \"%s\" content to JSON", file)
		return nil, err
	}
	return messages, nil
}
