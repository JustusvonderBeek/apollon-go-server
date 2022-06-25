package database

import (
	"Loxias/apollontypes"
	"Loxias/packets"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
)

var database = make(map[uint32]apollontypes.User)

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
		return errors.New("Empty username")
	}

	if user.UserId == 0 {
		log.Println("Cannot store user with user id 0!")
		return errors.New("User ID 0")
	}

	return nil
}

func StoreInDatabase(user packets.Create, connection net.Conn) error {
	// log.Println("Storing user in database")

	newUser := apollontypes.User{
		Username:   user.Username,
		UserId:     user.UserId,
		Connection: connection,
	}

	err := CheckUser(newUser)
	if err != nil {
		return err
	}

	_, exists := database[user.UserId]

	if exists {
		log.Printf("User with ID %d already exists", user.UserId)
		return errors.New("User already exists")
	}

	database[user.UserId] = newUser

	log.Printf("Stored user \"%s\" with id \"%d\"", user.Username, user.UserId)
	return nil
}

func StoreUserInDatabase(user apollontypes.User) error {
	// log.Println("Storing user in database")
	err := CheckUser(user)
	if err != nil {
		return err
	}
	ref, exists := database[user.UserId]
	if exists && ref.Connection != nil {
		log.Printf("User with ID %d already exists", user.UserId)
		return errors.New("user already exists")
	}
	database[user.UserId] = user
	log.Printf("Stored user \"%s\" with id \"%d\"", user.Username, user.UserId)
	return nil
}

func SearchUsers(search string) []packets.Contact {
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

func GetUser(userId uint32) (apollontypes.User, error) {
	user, err := database[userId]

	if !err {
		log.Printf("Failed to retrieve user with id \"%d\"", userId)
		return user, errors.New("User not found")
	}

	return user, nil
}

func IdExists(id uint32) bool {
	log.Printf("Checking if ID %d exists", id)

	_, exists := database[id]

	return exists
}

func Clear() {
	database = make(map[uint32]apollontypes.User)
}

func ConvertToByte() ([]byte, error) {
	var content []byte
	var err error
	err = nil
	seperator := ","
	counter := 0
	for _, v := range database {
		v.Connection = nil
		raw, err := json.Marshal(v)
		if err != nil {
			log.Printf("Failed to convert user %d to byte", v.UserId)
			err = errors.New("Not all users converted")
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

func ReadFromFile(file string) error {
	Clear()
	log.Printf("Reading from \"%s\"", file)
	content, err := ioutil.ReadFile(file)
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
