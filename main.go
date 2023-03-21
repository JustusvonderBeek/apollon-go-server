package main

import (
	"Loxias/configuration"
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
	tlsPort := flag.String("tp", "50001", "TLS listen port")
	logfile := flag.String("l", "server.log", "The logfile")
	clearDb := flag.Bool("e", true, "Clear existing database")
	tlsCertificate := flag.String("c", "resources/apollon.crt", "The location of the TLS certificate")
	tlsKeyfile := flag.String("k", "resources/apollon.key", "The location of the TLS key")
	flag.Parse()

	configuration := configuration.Config{
		Secure:             *securePtr,
		ListenAddr:         *addr,
		ListenPort:         *port,
		SecureListenPort:   *tlsPort,
		Logfile:            *logfile,
		ClearDatabase:      *clearDb,
		CertificateFile:    *tlsCertificate,
		CertificateKeyfile: *tlsKeyfile,
	}

	setupLogger(*logfile)
	server.Start(configuration)
}

func setupLogger(logfile string) {
	logFile, err := os.OpenFile(logfile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}
