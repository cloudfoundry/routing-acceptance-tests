package main

import (
	"flag"
	"fmt"
	"net"
	"os"
)

const (
	DEFAULT_ADDRESS = "localhost:3333"
	CONN_TYPE       = "tcp"
)

var serverAddress = flag.String(
	"address",
	DEFAULT_ADDRESS,
	"The host:port that the server is bound to.",
)

func main() {
	flag.Parse()
	// Listen for incoming connections.
	listener, err := net.Listen(CONN_TYPE, *serverAddress)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer listener.Close()
	fmt.Println("Listening on " + *serverAddress)
	for {
		// Listen for an incoming connection.
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go handleRequest(conn)
	}
}

// Handles incoming requests.
func handleRequest(conn net.Conn) {
	// Close the connection when you're done with it.
	defer conn.Close()
	// Make a buffer to hold incoming data.
	buff := make([]byte, 1024)
	// Continue to receive the data forever...
	for {
		// Read the incoming connection into the buffer.
		bytes, err := conn.Read(buff)
		if err != nil {
			fmt.Println("Closing connection:", err.Error())
			return
		}
		fmt.Print(string(buff[0:bytes]))
		_, err = conn.Write(buff[0:bytes])
		if err != nil {
			fmt.Println("Closing connection:", err.Error())
			return
		}
	}
}
