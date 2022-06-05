package database_test

import (
	"Loxias/apollontypes"
	"Loxias/database"
	"log"
	"os"
	"testing"
)

func TestInsertUser(t *testing.T) {
	log.Println("Testing inserting user")

	user := apollontypes.User{
		Username:   "test",
		UserId:     123456789,
		Connection: nil,
	}

	err := database.StoreUserInDatabase(user)

	if err != nil {
		log.Printf("Failed: %s", err)
		t.Fail()
	}

	user = apollontypes.User{
		Username:   "",
		UserId:     12345,
		Connection: nil,
	}

	err = database.StoreUserInDatabase(user)
	if err == nil {
		log.Println("Wrong user not rejected!")
		t.Fail()
	}

	user.Username = "test"
	user.UserId = 0

	err = database.StoreUserInDatabase(user)
	if err == nil {
		log.Println("Wrong user not rejected!")
		t.Fail()
	}

	user.UserId = 123456789
	err = database.StoreUserInDatabase(user)
	if err == nil {
		log.Println("Duplicate user stored!")
		t.Fail()
	}
	user.UserId = 10
	err = database.StoreUserInDatabase(user)
	if err != nil {
		log.Println("Failed to insert correct user")
		t.Fail()
	}
	database.PrintDatabase()
}

func TestStoringUser(t *testing.T) {
	log.Println("Testing storing users")
	user := apollontypes.User{
		Username:   "test",
		UserId:     1,
		Connection: nil,
	}
	err := database.StoreUserInDatabase(user)
	if err != nil {
		log.Println("Failed to store test user in database")
		t.Fail()
	}
	user.UserId = 2
	user.Username = "test2"
	err = database.StoreUserInDatabase(user)
	if err != nil {
		log.Println("Failed to store test user 2 in database")
		t.Fail()
	}
	err = database.SaveToFile("./database.json")
	if err != nil {
		log.Println("Failed to write to file!")
		t.Fail()
	}
	f, err := os.OpenFile("./database.json", os.O_RDONLY, os.ModeAppend)
	if err != nil {
		log.Println("Created file not existing!")
		t.Fail()
	}
	defer f.Close()
}

func TestLoadingDatabase(t *testing.T) {
	database.Clear()
	log.Println("Testing loading database")
	err := database.ReadFromFile("./database.json")
	if err != nil {
		log.Printf("Failed to load database: %s", err)
		t.Fail()
	}
	user, err := database.GetUser(2)
	if err != nil {
		log.Println("Failed to retrieve existing user")
		t.Fail()
	}
	if user.Username != "test2" {
		log.Println("Existing user was stored incorrectly!")
		t.Fail()
	}
}
