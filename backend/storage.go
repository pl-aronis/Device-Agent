package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Device struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"`
	RecoveryKey string    `json:"recovery_key"`
	MacID       string    `json:"mac_id"`
	Location    string    `json:"location"`
	OSDetails   string    `json:"os_details"`
	LastSeen    time.Time `json:"last_seen"`
}

type Storage struct {
	path    string
	mu      sync.Mutex
	devices map[string]Device
}

func NewStorage(path string) *Storage {
	s := &Storage{path: path, devices: make(map[string]Device)}
	_ = s.load()
	return s
}

func (s *Storage) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// ensure parent dir exists
	_ = os.MkdirAll(filepath.Dir(s.path), 0755)

	b, err := ioutil.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return s.saveLocked()
		}
		return err
	}
	if len(b) == 0 {
		return nil
	}
	var arr []Device
	if err := json.Unmarshal(b, &arr); err != nil {
		return err
	}
	for _, d := range arr {
		s.devices[d.ID] = d
	}
	return nil
}

func (s *Storage) saveLocked() error {
	arr := make([]Device, 0, len(s.devices))
	for _, d := range s.devices {
		arr = append(arr, d)
	}
	data, err := json.MarshalIndent(arr, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(s.path, data, 0644)
}

func (s *Storage) save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

func genID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func genRecoveryKey() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Storage) Register(prefID, macID, location, osDetails string) (Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := prefID
	if id == "" {
		id = genID()
	}
	d := Device{
		ID:          id,
		Status:      "ACTIVE",
		RecoveryKey: genRecoveryKey(),
		MacID:       macID,
		Location:    location,
		OSDetails:   osDetails,
		LastSeen:    time.Now(),
	}
	s.devices[id] = d
	if err := s.saveLocked(); err != nil {
		return Device{}, err
	}
	return d, nil
}

func (s *Storage) Heartbeat(id string) (Device, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.devices[id]
	if !ok {
		return Device{}, false
	}
	d.LastSeen = time.Now()
	s.devices[id] = d
	_ = s.saveLocked()
	return d, true
}

func (s *Storage) UpdateStatus(id, status string) (Device, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.devices[id]
	if !ok {
		return Device{}, "", false
	}
	oldStatus := d.Status
	d.Status = status
	s.devices[id] = d
	_ = s.saveLocked()
	return d, oldStatus, true
}

func (s *Storage) GetDevice(id string) (Device, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.devices[id]
	return d, ok
}

func (s *Storage) AllDevices() []Device {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Device, 0, len(s.devices))
	for _, d := range s.devices {
		out = append(out, d)
	}
	return out
}
