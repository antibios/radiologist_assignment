package main

import (
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":2575")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	log.Println("HL7 Listener started on :2575")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	// HL7 processing logic would go here
	log.Println("Connection accepted")
}
