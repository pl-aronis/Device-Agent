package state

import (
	"encoding/json"
	"os"
)

type Store struct {
	file string
	data struct {
		AgentID string `json:"agent_id"`
	}
}

func NewStore(file string) *Store {
	s := &Store{file: file}
	s.load()
	return s
}

func (s *Store) load() {
	b, err := os.ReadFile(s.file)
	if err != nil {
		return
	}
	json.Unmarshal(b, &s.data)
}

func (s *Store) save() {
	b, _ := json.MarshalIndent(s.data, "", "  ")
	os.WriteFile(s.file, b, 0644)
}

func (s *Store) IsRegistered() bool {
	return s.data.AgentID != ""
}

func (s *Store) SaveAgentID(id string) {
	s.data.AgentID = id
	s.save()
}
