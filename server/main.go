package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error listening:", err)
		os.Exit(1)
	}
	defer l.Close()
	fmt.Println("Listening on port 8080")
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err)
			os.Exit(1)
		}
		go handleRequest(conn)
	}
}

// Handles incoming requests.
func handleRequest(conn net.Conn) {
	//sending random data continuously
	for {
		_, err := conn.Write([]byte("random data"))
		if err != nil {
			fmt.Println("Error writing to stream.")
			break
		}
		//time.Sleep(10 * time.Millisecond)
	}
}
