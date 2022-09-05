package main

import (
	"Loxias/server"
	"flag"
	"io"
	"log"
	"os"
)

func main() {
	// Parsing the cmdline
	securePtr := flag.Bool("t", false, "Enable TLS")
	addr := flag.String("a", "0.0.0.0", "Listen address")
	port := flag.String("p", "50000", "Listen port")
	logfile := flag.String("l", "server.log", "The logfile")
	flag.Parse()

	setupLogger(*logfile)
	server.Start(*addr, *port, *securePtr)
}

func setupLogger(logfile string) {
	logFile, err := os.OpenFile(logfile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}
