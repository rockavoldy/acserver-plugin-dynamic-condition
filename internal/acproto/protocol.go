package acproto

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

type MessageType byte

const (
	ErrorByte           MessageType = 0
	NewSession          MessageType = 50
	NewConnection       MessageType = 51
	ConnectionClosed    MessageType = 52
	CarUpdate           MessageType = 53
	CarInfo             MessageType = 54
	EndSession          MessageType = 55
	Version             MessageType = 56
	Chat                MessageType = 57
	ClientLoaded        MessageType = 58
	SessionInfo         MessageType = 59
	Error               MessageType = 60
	LapCompleted        MessageType = 73
	ClientEvent         MessageType = 130
	RealtimePosInterval MessageType = 200
	GetCarInfo          MessageType = 201
	SendChat            MessageType = 202
	BroadcastChat       MessageType = 203
	GetSessionInfo      MessageType = 204
	KickUser            MessageType = 206
	NextSession         MessageType = 207
	RestartSession      MessageType = 208
	AdminCommand        MessageType = 209
)

type TimestampedBytes struct {
	RawData      []byte
	IncomingDate time.Time
}

type Vector3F struct{ X, Y, Z float32 }

type Message interface{ Type() MessageType }

type MsgVersionInfo struct{ Value byte }

func (MsgVersionInfo) Type() MessageType { return Version }

type MsgClientLoaded struct{ CarID byte }

func (MsgClientLoaded) Type() MessageType { return ClientLoaded }

type MsgCarUpdate struct {
	CarID                    byte
	WorldPosition, Velocity  Vector3F
	Gear                     byte
	EngineRPM                uint16
	NormalizedSplinePosition float32
}

func (MsgCarUpdate) Type() MessageType { return CarUpdate }

type MsgChat struct {
	CarID   byte
	Message string
}

func (MsgChat) Type() MessageType { return Chat }

type MsgSessionInfo struct {
	Version, SessionIndex, CurrentSessionIndex, SessionCount byte
	ServerName, Track, TrackConfig, Name                     string
	SessionType                                              byte
	SessionDuration, Laps, WaitTime                          uint16
	AmbientTemp, RoadTemp                                    byte
	Weather                                                  string
	ElapsedMS                                                int32
	IsNew                                                    bool
}

func (m MsgSessionInfo) Type() MessageType {
	if m.IsNew {
		return NewSession
	}
	return SessionInfo
}

type MsgCarInfo struct {
	CarID                                                 byte
	IsConnected                                           bool
	CarModel, CarSkin, DriverName, DriverTeam, DriverGUID string
}

func (MsgCarInfo) Type() MessageType { return CarInfo }

type MsgNewConnection struct {
	DriverName, DriverGUID, CarModel, CarSkin string
	CarID                                     byte
}

func (MsgNewConnection) Type() MessageType { return NewConnection }

type MsgConnectionClosed struct {
	DriverName, DriverGUID string
	CarID                  byte
}

func (MsgConnectionClosed) Type() MessageType { return ConnectionClosed }

type MsgError struct{ ErrorMessage string }

func (MsgError) Type() MessageType { return Error }

type DuplexUDPClient struct {
	conn   *net.UDPConn
	remote *net.UDPAddr
	done   chan struct{}
	mu     sync.Mutex
}

func (c *DuplexUDPClient) Open(listeningPort int, remoteHost string, remotePort int, onMessage func(TimestampedBytes), onError func(error)) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return errors.New("UDP client already opened")
	}
	remote, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", remoteHost, remotePort))
	if err != nil {
		return err
	}
	local, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", listeningPort))
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", local)
	if err != nil {
		return err
	}
	c.conn, c.remote, c.done = conn, remote, make(chan struct{})
	go func() {
		buf := make([]byte, 65535)
		for {
			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				select {
				case <-c.done:
					return
				default:
					if onError != nil {
						onError(err)
					}
					continue
				}
			}
			raw := append([]byte(nil), buf[:n]...)
			onMessage(TimestampedBytes{RawData: raw, IncomingDate: time.Now()})
		}
	}()
	return nil
}
func (c *DuplexUDPClient) Opened() bool { c.mu.Lock(); defer c.mu.Unlock(); return c.conn != nil }
func (c *DuplexUDPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	close(c.done)
	err := c.conn.Close()
	c.conn = nil
	return err
}
func (c *DuplexUDPClient) SendBytes(raw []byte) error {
	c.mu.Lock()
	conn, remote := c.conn, c.remote
	c.mu.Unlock()
	if conn == nil {
		return errors.New("UDP client not opened")
	}
	_, err := conn.WriteToUDP(raw, remote)
	return err
}

func Parse(raw TimestampedBytes) (Message, error) {
	if len(raw.RawData) == 0 {
		return nil, errors.New("empty message")
	}
	r := bytes.NewReader(raw.RawData[1:])
	mt := MessageType(raw.RawData[0])
	switch mt {
	case Version:
		v, _ := r.ReadByte()
		return MsgVersionInfo{Value: v}, nil
	case ClientLoaded:
		b, _ := r.ReadByte()
		return MsgClientLoaded{CarID: b}, nil
	case Chat:
		id, _ := r.ReadByte()
		s, _ := readStringW(r)
		return MsgChat{CarID: id, Message: s}, nil
	case SessionInfo, NewSession:
		m := MsgSessionInfo{IsNew: mt == NewSession}
		m.Version, _ = r.ReadByte()
		m.SessionIndex, _ = r.ReadByte()
		m.CurrentSessionIndex, _ = r.ReadByte()
		m.SessionCount, _ = r.ReadByte()
		m.ServerName, _ = readStringW(r)
		m.Track, _ = readString(r)
		m.TrackConfig, _ = readString(r)
		m.Name, _ = readString(r)
		m.SessionType, _ = r.ReadByte()
		binary.Read(r, binary.LittleEndian, &m.SessionDuration)
		binary.Read(r, binary.LittleEndian, &m.Laps)
		binary.Read(r, binary.LittleEndian, &m.WaitTime)
		m.AmbientTemp, _ = r.ReadByte()
		m.RoadTemp, _ = r.ReadByte()
		m.Weather, _ = readString(r)
		binary.Read(r, binary.LittleEndian, &m.ElapsedMS)
		return m, nil
	case CarUpdate:
		m := MsgCarUpdate{}
		m.CarID, _ = r.ReadByte()
		readVector(r, &m.WorldPosition)
		readVector(r, &m.Velocity)
		m.Gear, _ = r.ReadByte()
		binary.Read(r, binary.LittleEndian, &m.EngineRPM)
		binary.Read(r, binary.LittleEndian, &m.NormalizedSplinePosition)
		return m, nil
	case CarInfo:
		m := MsgCarInfo{}
		m.CarID, _ = r.ReadByte()
		b, _ := r.ReadByte()
		m.IsConnected = b != 0
		m.CarModel, _ = readStringW(r)
		m.CarSkin, _ = readStringW(r)
		m.DriverName, _ = readStringW(r)
		m.DriverTeam, _ = readStringW(r)
		m.DriverGUID, _ = readStringW(r)
		return m, nil
	case NewConnection:
		m := MsgNewConnection{}
		m.DriverName, _ = readStringW(r)
		m.DriverGUID, _ = readStringW(r)
		m.CarID, _ = r.ReadByte()
		m.CarModel, _ = readStringW(r)
		m.CarSkin, _ = readStringW(r)
		return m, nil
	case ConnectionClosed:
		m := MsgConnectionClosed{}
		m.DriverName, _ = readStringW(r)
		m.DriverGUID, _ = readStringW(r)
		m.CarID, _ = r.ReadByte()
		return m, nil
	case Error:
		s, _ := readStringW(r)
		return MsgError{ErrorMessage: s}, nil
	default:
		return nil, fmt.Errorf("unsupported message type %d", mt)
	}
}

func SerializeBroadcastChat(message string) []byte {
	var b bytes.Buffer
	b.WriteByte(byte(BroadcastChat))
	writeStringW(&b, message)
	return b.Bytes()
}
func SerializeSendChat(carID byte, message string) []byte {
	var b bytes.Buffer
	b.WriteByte(byte(SendChat))
	b.WriteByte(carID)
	writeStringW(&b, message)
	return b.Bytes()
}
func SerializeRequestCarInfo(carID byte) []byte { return []byte{byte(GetCarInfo), carID} }
func SerializeSessionInfoRequest(index int16) []byte {
	var b bytes.Buffer
	b.WriteByte(byte(GetSessionInfo))
	binary.Write(&b, binary.LittleEndian, index)
	return b.Bytes()
}
func SerializeRealtimeInterval(ms uint16) []byte {
	var b bytes.Buffer
	b.WriteByte(byte(RealtimePosInterval))
	binary.Write(&b, binary.LittleEndian, ms)
	return b.Bytes()
}

func readVector(r *bytes.Reader, v *Vector3F) {
	binary.Read(r, binary.LittleEndian, &v.X)
	binary.Read(r, binary.LittleEndian, &v.Y)
	binary.Read(r, binary.LittleEndian, &v.Z)
}
func readString(r *bytes.Reader) (string, error) {
	l, err := r.ReadByte()
	if err != nil {
		return "", err
	}
	data := make([]byte, int(l))
	_, err = r.Read(data)
	return string(data), err
}
func writeStringW(b *bytes.Buffer, s string) {
	runes := []rune(s)
	b.WriteByte(byte(len(runes)))
	for _, r := range runes {
		binary.Write(b, binary.LittleEndian, uint32(r))
	}
}
func readStringW(r *bytes.Reader) (string, error) {
	l, err := r.ReadByte()
	if err != nil {
		return "", err
	}
	out := make([]rune, int(l))
	for i := range out {
		var x uint32
		if err := binary.Read(r, binary.LittleEndian, &x); err != nil {
			return "", err
		}
		out[i] = rune(x)
	}
	return string(out), nil
}
