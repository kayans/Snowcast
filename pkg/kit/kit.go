package kit

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// a struct to represent client connections
type Client struct {
	Station   *Station    // current station
	TcpConn   net.Conn    // for future use
	UdpConn   net.Conn    // use for sending song data
	CloseChan chan int    // use for closing all client connections
	SongChan  chan string // use for sending Announce messages
}

// a struct to represent stations
type Station struct {
	Songname  string    // name of the song currently playing
	Listeners []*Client // all clients listening to this station
}

func NewStation(name string) *Station {
	return &Station{Songname: name}
}

// a struct to represent the state of the server
type State struct {
	clients      []*Client      // all connected clients
	Stations     []*Station     // all stations
	waitGroup    sync.WaitGroup // use for waiting for all clients to be done
	clientsMutex sync.RWMutex   // ensure only one goroutine can modify the client list at a time
}

func NewState(files []string) *State {
	n := len(files)
	stations := make([]*Station, n)
	for i, file := range files {
		stations[i] = NewStation(file)
	}
	return &State{Stations: stations}
}

func (s *State) StartStations() {
	// start sending data from radio stations to client listener programs
	for _, station := range s.Stations {
		go start(station, s) // start a new goroutine to send out song data
	}
}

const (
	count     = 16                // send 16 chunks per second
	chunkSize = 16 * 1024 / count // the size of a chunk of song data
	interval  = 1000000 / count   // send out a chunk of song data to every connected listener at every time interval
)

func start(s *Station, state *State) {
	file, err := os.Open(s.Songname)
	if err != nil {
		log.Println(err)
		return
	}
	// send song data at a rate of 16KiB/s
	for {
		data := make([]byte, chunkSize) // read a chunk from the file
		startTime := time.Now()         // store the start time
		n, err := file.Read(data)
		if err != nil && err != io.EOF {
			fmt.Println(err)
			return
		}
		if n < chunkSize || err == io.EOF { // send an Announce when a song repeats
			file.Seek(0, io.SeekStart) // set the offset for the next Read
			notify(s, state)           // notify
		}
		send(s, state, data, n) // send out this chunk of song data to every connected listener
		// measure the time it takes to send out the data, and subtract this from the sleep time
		time.Sleep(interval*time.Microsecond - time.Since(startTime))
	}
}

func send(s *Station, state *State, data []byte, n int) {
	for _, client := range s.Listeners {
		client.UdpConn.Write(data[:n]) // send out the data to listener
	}
}

func notify(s *Station, state *State) {
	for _, client := range s.Listeners {
		client.SongChan <- s.Songname // send songname to channel
	}
}

func (s *State) AddClient(tcpConn net.Conn, udpConn net.Conn) *Client {
	client := &Client{
		Station:   nil,
		TcpConn:   tcpConn,
		UdpConn:   udpConn,
		CloseChan: make(chan int, 1),
		SongChan:  make(chan string, 1),
	}
	s.waitGroup.Add(1)
	s.clientsMutex.Lock()
	s.clients = append(s.clients, client)
	s.clientsMutex.Unlock()
	return client
}

func (s *State) RemoveClient(client *Client) {
	s.clientsMutex.Lock()
	index := -1
	for i, c := range s.clients {
		if c == client {
			index = i
			break
		}
	}
	if index == -1 {
		s.clientsMutex.Unlock()
		return
	} else {
		s.clients = append(s.clients[:index], s.clients[index+1:]...)
		s.clientsMutex.Unlock()
	}
	if client.Station != nil {
		index = -1
		for i, c := range client.Station.Listeners {
			if c == client {
				index = i
				break
			}
		}
		if index != -1 {
			// remove client from listener list of subscribed station
			client.Station.Listeners = append(client.Station.Listeners[:index], client.Station.Listeners[index+1:]...)
		}
	}
	s.waitGroup.Done()
}

func (s *State) SetStation(x int, client *Client) {
	if client.Station != nil {
		index := -1
		for i, c := range client.Station.Listeners {
			if c == client {
				index = i
				break
			}
		}
		if index != -1 {
			// remove client from listener list of old station
			client.Station.Listeners = append(client.Station.Listeners[:index], client.Station.Listeners[index+1:]...)
		}
	}
	// change station
	client.Station = s.Stations[x]
	// add client to listener list of new station
	client.Station.Listeners = append(client.Station.Listeners, client)
}

func (s *State) Close() {
	s.clientsMutex.Lock()
	for _, client := range s.clients {
		client.CloseChan <- 1 // send something to channel
	}
	s.clientsMutex.Unlock()
	s.waitGroup.Wait() // Wait for all clients to be done
}

func ReadKeyboardInput(inputChan chan string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		cmd, err := reader.ReadString('\n') // wait for a line of input
		if err != nil {
			log.Println(err)
			continue
		}
		cmd = strings.Replace(cmd, "\n", "", -1)
		if len(cmd) > 0 { // only non-empty line will be sent
			inputChan <- cmd // send to main loop
		}
		reader.Reset(os.Stdin)
	}
}
