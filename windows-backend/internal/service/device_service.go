package service

import (
	"errors"
	"strings"
	"time"
	"windows-backend/internal/models"
	"windows-backend/internal/store"

	"github.com/google/uuid"
)

type DeviceService struct {
	store *store.JSONStore
}

func NewDeviceService(store *store.JSONStore) *DeviceService {
	return &DeviceService{store: store}
}

func (s *DeviceService) Register(d models.Device) models.Device {

	existing, found := s.store.Get(d.MacID)
	if found {
		existing.LastSeen = time.Now().UTC()
		s.store.Save(existing)
		return existing
	}

	d.AgentID = uuid.New().String()
	d.ShouldLock = false
	d.IsLocked = false
	d.LastSeen = time.Now().UTC()

	s.store.Save(d)
	return d
}

func (s *DeviceService) ReAuthenticate(mac string) (models.Device, bool) {
	return s.store.Get(mac)
}

func (s *DeviceService) Heartbeat(mac string, lat, lon float64) models.Device {

	d, found := s.store.Get(mac)
	if !found {
		return models.Device{}
	}

	d.Latitude = lat
	d.Longitude = lon
	d.LastSeen = time.Now().UTC()

	s.store.Save(d)
	return d
}

func (s *DeviceService) MarkLocked(mac, key, id string) {
	d, found := s.store.Get(mac)
	if !found {
		return
	}
	d.IsLocked = true
	d.ShouldLock = false
	d.RecoveryKey = key
	d.RecoveryID = id
	d.LastSeen = time.Now().UTC()
	s.store.Save(d)
}

func (s *DeviceService) MarkLockFailed(mac string) {
	d, found := s.store.Get(mac)
	if !found {
		return
	}
	d.ShouldLock = true
	d.LastSeen = time.Now().UTC()
	s.store.Save(d)
}

func (s *DeviceService) ListDevices() []models.Device {
	return s.store.List()
}

func (s *DeviceService) SetAdminStatus(agentID, status string) (models.Device, error) {
	d, found := s.store.GetByAgentID(agentID)
	if !found {
		return models.Device{}, errors.New("device not found")
	}

	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "LOCK":
		d.ShouldLock = true
		d.IsLocked = false
	case "ACTIVE":
		d.ShouldLock = false
		d.IsLocked = false
	default:
		return models.Device{}, errors.New("invalid status")
	}

	d.LastSeen = time.Now().UTC()
	s.store.Save(d)
	return d, nil
}
