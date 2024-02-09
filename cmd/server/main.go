package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gopher9527/snowcast/pkg/kit"
	"github.com/gopher9527/snowcast/pkg/protocol"
)

var state *kit.State

func main() {
	if len(os.Args) < 3 { // wrong number of arguments
		// show the usage of the server
		fmt.Println("usage: snowcast_server <tcpport> <file0> [file 1] [file 2] ...")
		return
	}

	state = kit.NewState(os.Args[2:])
	// stations start even though no one is listening now
	state.StartStations()

	listen(os.Args[1])

	keyboardChan := make(chan string, 1)
	// start a goroutine to read from keyboard
	go kit.ReadKeyboardInput(keyboardChan)

	signalChan := make(chan os.Signal, 1)
	// catch Ctrl + C
	signal.Notify(signalChan, os.Interrupt, syscall.SIGINT)

	for {
		// watch both channels, do something when an event happens
		select {
		case <-signalChan:
			state.Close()
			return
		case cmd := <-keyboardChan: // input from keyboard
			g := strings.Fields(cmd)
			switch g[0] {
			case "p":
				if len(g) == 1 {
					// print to stdout a list of stations along with the listeners that are connected to each one
					go print(os.Stdout)
				} else {
					// write the list of stations to the specified file
					file, err := os.Create(g[1])
					if err != nil {
						continue
					}
					go print(file)
				}
			case "q": //  close all connections and exit
				state.Close()
				return
			}
		}
	}
}
func listen(tcpPort string) {
	// get a TCPAddr and listen on the port number we specified on the command line
	addr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf(":%s", tcpPort))
	if err != nil {
		log.Fatalln(err)
	}
	// create listen socket
	listener, err := net.ListenTCP("tcp4", addr)
	if err != nil {
		log.Fatalln(err)
	}
	go accept(listener)
}

func accept(listener *net.TCPListener) {
	for {
		conn, err := listener.AcceptTCP() // wait for new connections
		if err != nil {
			continue
		}
		go handle(conn) // start a goroutine for the connection
	}
}

func handle(tcpConn net.Conn) {
	udpConn, ok := handshake(tcpConn)
	if !ok {
		tcpConn.Close()
		return
	}

	client := state.AddClient(tcpConn, udpConn)

	closeChan := make(chan int, 1)
	socketChan := make(chan any, 1)
	// start a goroutine to wait for a message from the client
	go message(tcpConn, closeChan, socketChan)

	for {
		// watch all channels, do something when an event happens
		select {
		case a := <-socketChan:
			if a == nil {
				closeChan <- 1
				state.RemoveClient(client)
				return
			} else {
				ok := handleCommand(tcpConn, a, client)
				if !ok {
					// close the connection right away in order to pass the test
					// this may be because handleCommand function and channel are somewhat time consuming
					tcpConn.Close()
					closeChan <- 1
					state.RemoveClient(client)
					return
				}
			}
		case songname := <-client.SongChan: // receive on the channel
			_, err := protocol.WriteMessage(tcpConn, protocol.NewAnnounce(songname))
			if err != nil {
				closeChan <- 1
				state.RemoveClient(client)
				return
			}
		case <-client.CloseChan:
			closeChan <- 1
			state.RemoveClient(client)
			return
		}
	}
}

func handshake(tcpConn net.Conn) (net.Conn, bool) {
	// try to read a message from the socket
	a, err := protocol.ReadMessage(tcpConn, true)
	if err != nil {
		return nil, false
	}
	h, ok := a.(*protocol.Hello) // conversion from any to *Hello
	if !ok {
		return nil, false
	}
	// build a welcome message and send it
	protocol.WriteMessage(tcpConn, protocol.NewWelcome(uint16(len(state.Stations))))
	if err != nil {
		return nil, false
	}
	remoteAddr := tcpConn.RemoteAddr()
	remoteIP := strings.Split(remoteAddr.String(), ":")[0]
	// create a connection to use for sending song data
	udpConn, err := net.Dial("udp4", fmt.Sprintf("%s:%d", remoteIP, h.UdpPort))
	if err != nil {
		return nil, false
	}
	return udpConn, true
}

func message(conn net.Conn, closeChan chan int, socketChan chan any) {
	defer conn.Close() // ensure the socket is closed when this goroutine exits
	for {
		// watch the channel, do something when an event happens
		select {
		case <-closeChan:
			return
		default:
			m, err := protocol.ReadMessage(conn, false)
			if err != nil {
				close(socketChan)
				return
			}
			socketChan <- m
		}
	}
}

func handleCommand(conn net.Conn, a any, client *kit.Client) bool {
	m, ok := a.(protocol.Message) // conversion from any to *Message
	if !ok {
		return false
	}
	switch m.GetType() {
	// case protocol.HelloCommandType:
	// 	protocol.WriteMessage(conn, protocol.NewInvalidCommand("Wrong"))
	// 	return false
	case protocol.SetStationCommandType:
		s, ok := m.(*protocol.SetStation) // conversion from Message to *SetStation
		if !ok {
			return false
		}
		return handleSetStation(conn, *s, client)
	case protocol.StationsCommandType:
		s, ok := m.(*protocol.StationsCommand) // conversion from Message to *StationsCommand
		if !ok {
			return false
		}
		return handleStationsCommand(conn, *s, client)
	default: // a Hello or An unknown command was sent
		protocol.WriteMessage(conn, protocol.NewInvalidCommand("invalid command"))
		return false
	}
}

// func handleHello(conn net.Conn, h protocol.Hello, client *kit.Client) bool {
// 	// just for fun
// 	return false
// }

func handleSetStation(conn net.Conn, s protocol.SetStation, client *kit.Client) bool {
	if uint16(len(state.Stations)) <= s.StationNumber {
		// build a InvalidCommand message and send it
		protocol.WriteMessage(conn, protocol.NewInvalidCommand("invalid station number"))
		return false
	}
	state.SetStation(int(s.StationNumber), client)
	// build a Announce message and send it
	_, err := protocol.WriteMessage(conn, protocol.NewAnnounce(client.Station.Songname))
	return err == nil
}

func print(w io.Writer) {
	// write the list of stations to the specified Writer
	for i, station := range state.Stations {
		fmt.Fprintf(w, "%d,%s", i, station.Songname)
		for _, listener := range station.Listeners {
			fmt.Fprintf(w, ",%s", listener.UdpConn.RemoteAddr())
		}
		fmt.Fprintln(w)
	}
}

// ======================================== Extra Credit     ========================================

func handleStationsCommand(conn net.Conn, s protocol.StationsCommand, client *kit.Client) bool {
	// returns a listing of what each of the stations is currently playing
	var result string
	for i, station := range state.Stations {
		result = fmt.Sprintf("%s%d %s\n", result, i, station.Songname)
	}
	// build a StationsReply message and send it
	_, err := protocol.WriteMessage(conn, protocol.NewStationsReply(result))
	return err == nil
}
