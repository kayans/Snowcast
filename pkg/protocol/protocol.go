package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"time"
)

const (
	HelloCommandType        uint8 = 0
	SetStationCommandType   uint8 = 1
	WelcomeReplyType        uint8 = 2
	AnnounceReplyType       uint8 = 3
	InvalidCommandReplyType uint8 = 4
	MessageTypeBound        uint8 = 4 // the upper boundary of types of standard messages
	// addition to the protocol fot extra credit
	StationsCommandType uint8 = 254 // request a listing of what each of the stations is currently playing
	StationsReplyType   uint8 = 255 // return a listing of what each of the stations is currently playing
)

// a interface to represent commands or replies
type Message interface {
	GetType() uint8           // return type of command or reply
	Marshal() ([]byte, error) // marshal message into a byte array
	Unmarshal([]byte)         // unmarshal message from a byte array
}

// ======================================== Hello Command ========================================

type Hello struct {
	commandType uint8
	UdpPort     uint16 // offset is 1
}

func NewHello(udpPort uint16) *Hello {
	return &Hello{HelloCommandType, udpPort}
}

func (h *Hello) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	// err := buf.WriteByte(h.commandType)
	err := binary.Write(buf, binary.BigEndian, h.commandType)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.BigEndian, h.UdpPort)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// func Unmarshal(data []byte, v any) error {
//     reflect
// }
// var h Hello
// err := Unmarshal(data, h)
// var w Welcome
// err := Unmarshal(data, w)

// interface Message { Unmarshal([]byte) }
// func (h *Hello) Unmarshal(data []byte) {}
// var h Hello
// h.Unmarshal(buf)
// func (w *Welcome) Unmarshal(data []byte) {}
// var w Welcome
// w.Unmarshal(buf)

func (h *Hello) Unmarshal(data []byte) {
	// h.commandType = data[0]
	h.commandType = HelloCommandType
	h.UdpPort = binary.BigEndian.Uint16(data[1:])
}

func (h *Hello) GetType() uint8 {
	return h.commandType
}

// ======================================== Hello Command      ========================================

// ======================================== SetStation Command ========================================

type SetStation struct {
	commandType   uint8
	StationNumber uint16 // offset is 1
}

func NewSetStation(stationNumber uint16) *SetStation {
	return &SetStation{SetStationCommandType, stationNumber}
}

func (s *SetStation) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, s.commandType)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.BigEndian, s.StationNumber)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *SetStation) Unmarshal(data []byte) {
	s.commandType = SetStationCommandType
	s.StationNumber = binary.BigEndian.Uint16(data[1:])
}

func (s *SetStation) GetType() uint8 {
	return s.commandType
}

// ======================================== SetStation Command ========================================

// ======================================== Welcome Reply      ========================================

type Welcome struct {
	replyType   uint8
	NumStations uint16 // offset is 1
}

func NewWelcome(numStations uint16) *Welcome {
	return &Welcome{WelcomeReplyType, numStations}
}

func (w *Welcome) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, w.replyType)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.BigEndian, w.NumStations)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (w *Welcome) Unmarshal(data []byte) {
	w.replyType = WelcomeReplyType
	w.NumStations = binary.BigEndian.Uint16(data[1:])
}

func (w *Welcome) GetType() uint8 {
	return w.replyType
}

// ======================================== Welcome Reply  ========================================

// ======================================== Announce Reply ========================================

type Announce struct {
	replyType    uint8
	songnameSize uint8
	Songname     []byte // offset is 2
	// fmt.Println(string(a.Songname))
	// buf.Write(a.Songname)

	// Songname string
	// fmt.Println(a.Songname)
	// buf.Write([]byte(a.Songname))
}

func NewAnnounce(songname string) *Announce {
	return &Announce{AnnounceReplyType, uint8(len(songname)), []byte(songname)}
}

func (a *Announce) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, a.replyType)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.BigEndian, a.songnameSize)
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(a.Songname)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (a *Announce) Unmarshal(data []byte) {
	a.replyType = AnnounceReplyType
	a.songnameSize = data[1]
	a.Songname = data[2:]
}

func (a *Announce) GetType() uint8 {
	return a.replyType
}

// ======================================== Announce Reply       ========================================

// ======================================== InvalidCommand Reply ========================================

type InvalidCommand struct {
	replyType       uint8
	replyStringSize uint8
	ReplyString     []byte // offset is 2
}

func NewInvalidCommand(replyString string) *InvalidCommand {
	return &InvalidCommand{InvalidCommandReplyType, uint8(len(replyString)), []byte(replyString)}
}

func (i *InvalidCommand) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, i.replyType)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.BigEndian, i.replyStringSize)
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(i.ReplyString)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (i *InvalidCommand) Unmarshal(data []byte) {
	i.replyType = InvalidCommandReplyType
	i.replyStringSize = data[1]
	i.ReplyString = data[2:]
}

func (i *InvalidCommand) GetType() uint8 {
	return i.replyType
}

// ======================================== InvalidCommand Reply ========================================

func WriteMessage(conn net.Conn, m Message) (int, error) {
	buf, err := m.Marshal() // marshal message into a byte array
	if err != nil {
		return 0, err
	}
	n, err := conn.Write(buf) // send it
	return n, err
}

func readMessageType(conn net.Conn, timeout bool) (uint8, error) {
	buf := make([]byte, 1)
	var t time.Time
	if timeout {
		t = time.Now().Add(100 * time.Millisecond) // the deadline for io.ReadFull call
	} else {
		t = time.Time{} //  a zero value for t means Read will not time out.
	}
	conn.SetReadDeadline(t)
	_, err := io.ReadFull(conn, buf)
	return buf[0], err
}

func ReadMessage(conn net.Conn, timeout bool) (any, error) {
	t, err := readMessageType(conn, timeout) // read message type
	if err != nil {
		return nil, err
	}
	// check whether t is a valid message type
	if t > MessageTypeBound && t != StationsCommandType && t != StationsReplyType {
		return nil, errors.New("unknown message type")
	}
	var offset uint8 = 1 // starting position of remaining part of Hello/SetStation/Welcome/StationsCommand in the buffer
	var remain uint8 = 2 // size of remaining part of Hello/SetStation/Welcome/StationsCommand
	var buf []byte
	if t == AnnounceReplyType || t == InvalidCommandReplyType || t == StationsReplyType {
		sizeBuf := make([]byte, 1)                                   // the buffer for the size of the remaining part of the message
		conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond)) // the deadline for io.ReadFull call
		_, err = io.ReadFull(conn, sizeBuf)                          // read size of remaining part
		if err != nil {
			return nil, err
		}
		offset = 2                        // starting position of remaining part of AnnounceReplyType/InvalidCommandReplyType/StationsReplyType in the buffer
		remain = sizeBuf[0]               // size of remaining part of AnnounceReplyType/InvalidCommandReplyType/StationsReplyType
		buf = make([]byte, offset+remain) // the buffer for the message
		buf[0] = t                        // message type
		buf[1] = byte(remain)             // string size
	} else {
		buf = make([]byte, offset+remain) // the buffer for the message
		buf[0] = t                        // message type
	}

	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond)) // receive all of the remaining bytes of the message within the next 100 milliseconds
	n, err := io.ReadFull(conn, buf[offset:])                    // read the remaining bytes and store them in the buffer beginning at offset

	if err == io.EOF {
		return nil, err
	} else if err != nil {
		return nil, err
	} else if n != int(remain) { // invalid message length
		return nil, err
	}

	switch t {
	case HelloCommandType:
		var h Hello
		h.Unmarshal(buf)
		return &h, nil
	case SetStationCommandType:
		var s SetStation
		s.Unmarshal(buf)
		return &s, nil
	case WelcomeReplyType:
		var w Welcome
		w.Unmarshal(buf)
		return &w, nil
	case AnnounceReplyType:
		var a Announce
		a.Unmarshal(buf)
		return &a, nil
	case InvalidCommandReplyType:
		var i InvalidCommand
		i.Unmarshal(buf)
		return &i, nil
	case StationsCommandType:
		var s StationsCommand
		s.Unmarshal(buf)
		return &s, nil
	case StationsReplyType:
		var s StationsReply
		s.Unmarshal(buf)
		return &s, nil
	}
	return nil, nil // MissingReturn
}

// ======================================== Extra Credit     ========================================

// ======================================== Stations Command ========================================

// command which requests a listing of what each of the stations is currently playing
type StationsCommand struct {
	commandType uint8
	none        uint16 // it is required by the ReadMessage function
}

func NewStationsCommand() *SetStation {
	return &SetStation{StationsCommandType, 0}
}

func (s *StationsCommand) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, s.commandType)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.BigEndian, s.none)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *StationsCommand) Unmarshal(data []byte) {
	s.commandType = StationsCommandType
	s.none = binary.BigEndian.Uint16(data[1:])
}

func (s *StationsCommand) GetType() uint8 {
	return s.commandType
}

// ======================================== Stations Command ========================================

// ======================================== Stations Reply   ========================================

// reply which returns a listing of what each of the stations is currently playing
type StationsReply struct {
	replyType       uint8
	replyStringSize uint8
	ReplyString     []byte // offset is 2
}

func NewStationsReply(replyString string) *StationsReply {
	return &StationsReply{StationsReplyType, uint8(len(replyString)), []byte(replyString)}
}

func (i *StationsReply) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, i.replyType)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.BigEndian, i.replyStringSize)
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(i.ReplyString)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (i *StationsReply) Unmarshal(data []byte) {
	i.replyType = StationsReplyType
	i.replyStringSize = data[1]
	i.ReplyString = data[2:]
}

func (i *StationsReply) GetType() uint8 {
	return i.replyType
}

// ======================================== Stations Reply ========================================
