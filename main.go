package main

import (
	"Loxias/server"
	"flag"
)

func main() {
	// Parsing the cmdline
	securePtr := flag.Bool("t", false, "Enable TLS")
	flag.Parse()

	server.Start(*securePtr)
}
