package main

import (
	"log"
	"net"
	"os"
	"time"
)

const (
	defaultPort        = ":2575"
	maxConcurrentConns = 100
	readTimeout        = 30 * time.Second
)

func main() {
	port := os.Getenv("HL7_PORT")
	if port == "" {
		port = defaultPort
	}

	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	log.Printf("HL7 Listener started on %s", port)

	// Semaphore to limit concurrent connections
	sem := make(chan struct{}, maxConcurrentConns)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}

		// Acquire semaphore
		sem <- struct{}{}

		go func(c net.Conn) {
			defer func() {
				// Release semaphore
				<-sem
			}()
			handleConnection(c)
		}(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Set read deadline to prevent slow-client attacks
	if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		log.Printf("Failed to set read deadline: %v", err)
		return
	}

	// HL7 processing logic would go here
	// For skeleton, we just log and maybe read a bit to demonstrate

	// Example read (not implemented in full parser yet)
	// buf := make([]byte, 4096)
	// _, err := conn.Read(buf)

	log.Println("Connection accepted and handled securely")
}
