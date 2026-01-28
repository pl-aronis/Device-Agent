package store

import (
	"encoding/hex"
	"sync"
)

// Device represents a managed Apple device
type Device struct {
	UDID      string `json:"udid"`
	PushToken string `json:"push_token"` // Hex encoded
	PushMagic string `json:"push_magic"`
	// Can add more fields like SerialNumber, OSVersion later
}

type DeviceStore struct {
	mu      sync.RWMutex
	devices map[string]*Device
}

func NewDeviceStore() *DeviceStore {
	return &DeviceStore{
		devices: make(map[string]*Device),
	}
}

func (s *DeviceStore) SaveDevice(udid string, tokenBytes []byte, pushMagic string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	token := hex.EncodeToString(tokenBytes)
	s.devices[udid] = &Device{
		UDID:      udid,
		PushToken: token,
		PushMagic: pushMagic,
	}
}

func (s *DeviceStore) GetDevice(udid string) (*Device, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.devices[udid]
	return d, ok
}

func (s *DeviceStore) ListDevices() []*Device {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]*Device, 0, len(s.devices))
	for _, d := range s.devices {
		list = append(list, d)
	}
	return list
}

func (s *DeviceStore) RemoveDevice(udid string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.devices, udid)
}
