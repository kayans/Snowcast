package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

func main() {
	if len(os.Args) != 2 { // wrong number of arguments
		// show the usage of the listener
		fmt.Println("usage: snowcast_listener <udp_port>")
		return
	}
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf(":%s", os.Args[1]))
	if err != nil {
		log.Fatalln(err)
	}
	// create a socket and bind it to the port on which we want to listen
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		log.Fatalln(err)
	}
	// receives song data from the server and just writes it to stdout
	io.Copy(os.Stdout, conn)
}
