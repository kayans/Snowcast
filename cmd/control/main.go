package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/gopher9527/snowcast/pkg/kit"
	"github.com/gopher9527/snowcast/pkg/protocol"
)

var numStations uint16 // number of stations
var station = -1       // current station index

type Send struct {
	commandType uint8 // type of the command that will be sent to the server
	content     any   // content of the command that will be sent to the server
}

func main() {
	if len(os.Args) != 4 { // wrong number of arguments
		// show the usage of the control
		fmt.Println("usage: snowcast_control <server_name> <server_port> <udp_port>")
		return
	}

	closeChan := make(chan int, 1)
	sendChan := make(chan Send, 1)
	// use channels to signal the main loop to act on the data
	socketChan := make(chan any, 1)
	keyboardChan := make(chan string, 1)
	signalChan := make(chan os.Signal, 1)

	// start a goroutine to read from keyboard
	go kit.ReadKeyboardInput(keyboardChan)
	// catch Ctrl + C
	signal.Notify(signalChan, os.Interrupt, syscall.SIGINT)

	connect(os.Args[1], os.Args[2], os.Args[3], closeChan, socketChan, sendChan)

	for {
		// watch all channels, do something when an event happens
		select {
		case <-signalChan:
			return
		case a := <-socketChan: // input from socket
			ok := handleReply(a)
			if !ok {
				return
			}
		case cmd := <-keyboardChan: // input from keyboard
			g := strings.Fields(cmd)
			switch g[0] {
			case "q": // quit
				closeChan <- 1
				return
			case "stations":
				// send a Stations command
				sendChan <- Send{protocol.StationsCommandType, 0}
			default:
				s, err := strconv.ParseUint(cmd, 10, 16)
				if err != nil || uint16(s) >= numStations {
					// input is not a number or the number is outside the range given by the server
					log.Println("invalid input")
					continue
				}
				// send a SetStation command with the user-provided station number
				sendChan <- Send{protocol.SetStationCommandType, uint16(s)}
			}
		}
	}
}

func connect(serverName string, serverPort string, udpPort string, closeChan chan int, socketChan chan any, sendChan chan Send) {
	conn, err := net.Dial("tcp4", fmt.Sprintf("%s:%s", serverName, serverPort))
	if err != nil {
		log.Fatalln(err)
	}
	handshake(conn, udpPort)
	// start a goroutine to wait for a message from the server
	go listen(conn, closeChan, socketChan)
	// start a goroutine to send messages to the server
	go send(conn, closeChan, sendChan)
}

func handshake(conn net.Conn, udpPort string) {
	port, err := strconv.ParseUint(udpPort, 10, 16)
	if err != nil {
		log.Fatalln(err)
	}
	// build a hello message and send it
	_, err = protocol.WriteMessage(conn, protocol.NewHello(uint16(port)))
	if err != nil {
		log.Fatalln(err)
	}
	// wait for a response
	a, err := protocol.ReadMessage(conn, true)
	if err != nil {
		log.Fatalln(err)
	}
	w, ok := a.(*protocol.Welcome) // conversion from any to Welcome
	if !ok {
		log.Fatalln(err)
	}
	fmt.Printf("Welcome to Snowcast! The server has `%d` stations.\n", w.NumStations)
	numStations = w.NumStations // store the number of stations
}

func listen(conn net.Conn, closeChan chan int, socketChan chan any) {
	defer conn.Close() // ensure the socket is closed when this goroutine exits
	for {
		// watch the channel, do something when an event happens
		select {
		case <-closeChan:
			return
		default:
			m, err := protocol.ReadMessage(conn, false)
			if m == nil || err != nil {
				close(socketChan)
				return
			}
			socketChan <- m
		}
	}
}

func handleReply(a any) bool {
	m, ok := a.(protocol.Message) // conversion from any to Messge
	if !ok {
		return false
	}

	switch m.GetType() {
	// case protocol.WelcomeReplyType:
	// 	fmt.Println("Wrong")
	// 	return false
	case protocol.AnnounceReplyType:
		r, ok := m.(*protocol.Announce) // conversion from Message to *Announce
		if !ok {
			return false
		}
		return handleAnnounce(r)
	case protocol.InvalidCommandReplyType:
		r, ok := m.(*protocol.InvalidCommand) // conversion from Message to *InvalidCommand
		if !ok {
			return false
		}
		return handleInvalidCommand(r)
	case protocol.StationsReplyType:
		s, ok := m.(*protocol.StationsReply) // conversion from Message to *StationsReply
		if !ok {
			return false
		}
		return handleStationsReply(s)
	default: // a Welcome or an unknown response was sent
		fmt.Println("unknown reply")
		return false
	}
}

func handleAnnounce(a *protocol.Announce) bool {
	if station == -1 { // the server sends an Announce before the client has sent a SetStation
		return false
	}
	fmt.Printf("New song announced: %s\n", a.Songname) // print to stdout
	return true
}

func handleInvalidCommand(i *protocol.InvalidCommand) bool {
	fmt.Println(string(i.ReplyString)) // print to stdout
	return false
}

func send(conn net.Conn, closeChan chan int, sendChan chan Send) {
	for {
		// watch both channels, do something when an event happens
		select {
		case <-closeChan:
			return
		case send := <-sendChan:
			switch send.commandType {
			case protocol.SetStationCommandType:
				s, ok := send.content.(uint16) // conversion from any to uint16
				if !ok {
					continue
				}
				sendSetStation(conn, s)
			case protocol.StationsCommandType:
				sendStationsCommand(conn)
			}
		}
	}
}

func sendSetStation(conn net.Conn, s uint16) {
	// build a SetStation message and send it
	_, err := protocol.WriteMessage(conn, protocol.NewSetStation(s))
	if err != nil {
		fmt.Println(err)
	}
	station = int(s)
}

// ======================================== Extra Credit     ========================================

// send a command which requests a listing of what each of the stations is currently playing
func sendStationsCommand(conn net.Conn) {
	// build a StationsCommand message and send it
	_, err := protocol.WriteMessage(conn, protocol.NewStationsCommand())
	if err != nil {
		fmt.Println(err)
	}
}

// handle a reply which returns a listing of what each of the stations is currently playing
func handleStationsReply(s *protocol.StationsReply) bool {
	fmt.Println(string(s.ReplyString)) // just print the listing
	return true
}
