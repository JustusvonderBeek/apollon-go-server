package main

import (
	"Loxias/server"
	"os"
)

func main() {
	secure := false
	if len(os.Args) > 1 {
		args := os.Args[1:]
		if args[0] == "-tls" {
			secure = true
		}
	}

	server.Start(secure)
}
