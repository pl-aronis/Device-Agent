package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"windows-backend/internal/models"
)

type JSONStore struct {
	file string
	mu   sync.Mutex
	data map[string]models.Device
}

func NewJSONStore(file string) *JSONStore {
	s := &JSONStore{
		file: file,
		data: make(map[string]models.Device),
	}
	s.load()
	return s
}

func (s *JSONStore) load() {
	b, err := os.ReadFile(s.file)
	if err != nil {
		return
	}
	json.Unmarshal(b, &s.data)
}

func (s *JSONStore) save() {
	b, _ := json.MarshalIndent(s.data, "", "  ")
	_ = os.MkdirAll(filepath.Dir(s.file), 0755)
	os.WriteFile(s.file, b, 0644)
}

func (s *JSONStore) Save(device models.Device) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[device.MacID] = device
	s.save()
}

func (s *JSONStore) Get(mac string) (models.Device, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.data[mac]
	return d, ok
}

func (s *JSONStore) GetByAgentID(agentID string) (models.Device, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, d := range s.data {
		if d.AgentID == agentID {
			return d, true
		}
	}

	return models.Device{}, false
}

func (s *JSONStore) List() []models.Device {
	s.mu.Lock()
	defer s.mu.Unlock()

	devices := make([]models.Device, 0, len(s.data))
	for _, d := range s.data {
		devices = append(devices, d)
	}

	slices.SortFunc(devices, func(a, b models.Device) int {
		if a.LastSeen.Before(b.LastSeen) {
			return 1
		}
		if a.LastSeen.After(b.LastSeen) {
			return -1
		}
		if a.AgentID < b.AgentID {
			return -1
		}
		if a.AgentID > b.AgentID {
			return 1
		}
		return 0
	})

	return devices
}
