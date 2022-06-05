package database

import (
	"Loxias/packets"
	"apollontypes"
	"errors"
	"log"
	"net"
	"strings"
)

var database = make(map[uint32]apollontypes.User)

func StoreInDatabase(user packets.Create, connection net.Conn) {
	log.Println("Storing user in database")

	newUser := apollontypes.User{
		Username:   user.Username,
		UserId:     user.UserId,
		Connection: connection,
	}

	_, exists := database[user.UserId]

	if exists {
		log.Printf("User with ID %d already exists", user.UserId)
		return
	}

	database[user.UserId] = newUser

	log.Printf("Stored user \"%s\" with id \"%d\"", user.Username, user.UserId)
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

func PrintDatabase() {
	log.Println("Database:\n--------------------------")
	log.Printf("%s", database)
	log.Println("---------------------")
}
