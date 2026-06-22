package acproto

import (
	"fmt"
	"math"
	"sync"
	"time"

	"plugin-dynamic-conditions/internal/config"
)

type Plugin interface {
	OnInit(*Manager)
	OnConnected()
	OnClientLoaded(MsgClientLoaded)
	OnNewSession(MsgSessionInfo)
	OnCarUpdate(MsgCarUpdate)
	Dispose()
}
type basePlugin struct{}

type Manager struct {
	settings  config.PluginSettings
	udp       DuplexUDPClient
	plugins   []Plugin
	drivers   map[byte]float64
	mu        sync.Mutex
	connected bool
}

func NewManager(s config.PluginSettings) *Manager {
	return &Manager{settings: s, drivers: map[byte]float64{}}
}
func (m *Manager) AddPlugin(p Plugin) { m.plugins = append(m.plugins, p); p.OnInit(m) }
func (m *Manager) Connect() error {
	err := m.udp.Open(m.settings.ListeningPort, m.settings.RemoteHostName, m.settings.RemotePort, m.messageReceived, func(e error) { fmt.Println("UDP warning:", e) })
	if err != nil {
		return err
	}
	m.connected = true
	for _, p := range m.plugins {
		p.OnConnected()
	}
	go func() {
		time.Sleep(3 * time.Second)
		if m.connected {
			_ = m.udp.SendBytes(SerializeSessionInfoRequest(-1))
		}
	}()
	return nil
}
func (m *Manager) Close() {
	m.connected = false
	_ = m.udp.Close()
	for _, p := range m.plugins {
		p.Dispose()
	}
}
func (m *Manager) IsConnected() bool { return m.connected }
func (m *Manager) DriverSpeeds() []float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]float64, 0, len(m.drivers))
	for _, v := range m.drivers {
		out = append(out, v)
	}
	return out
}
func (m *Manager) BroadcastCSPCommand(cmd string) { _ = m.udp.SendBytes(SerializeBroadcastChat(cmd)) }
func (m *Manager) SendCSPCommand(carID byte, cmd string) {
	_ = m.udp.SendBytes(SerializeSendChat(carID, cmd))
}
func (m *Manager) messageReceived(ts TimestampedBytes) {
	msg, err := Parse(ts)
	if err != nil {
		return
	}
	switch v := msg.(type) {
	case MsgClientLoaded:
		for _, p := range m.plugins {
			p.OnClientLoaded(v)
		}
	case MsgSessionInfo:
		for _, p := range m.plugins {
			if v.IsNew {
				p.OnNewSession(v)
			}
		}
	case MsgCarUpdate:
		m.mu.Lock()
		m.drivers[v.CarID] = float64(length(v.Velocity)) * 3.6
		m.mu.Unlock()
		for _, p := range m.plugins {
			p.OnCarUpdate(v)
		}
	case MsgCarInfo:
		m.mu.Lock()
		if !v.IsConnected {
			delete(m.drivers, v.CarID)
		} else if _, ok := m.drivers[v.CarID]; !ok {
			m.drivers[v.CarID] = 0
		}
		m.mu.Unlock()
	}
}
func length(v Vector3F) float32 { return float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y + v.Z*v.Z))) }
