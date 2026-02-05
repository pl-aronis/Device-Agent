package storage

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"backend/internal/models"
)

// Store interface defines storage operations
type Store interface {
	Register(prefID, macID, location, osDetails string) (models.Device, error)
	Heartbeat(id string) (models.Device, bool)
	UpdateStatus(id, status string) (models.Device, string, bool)
	UpdateRecoveryKey(id, recoveryKey string) (models.Device, bool)
	GetDevice(id string) (models.Device, bool)
	AllDevices() []models.Device
	Close() error
}

// FileStore implements Store interface using file-based storage
type FileStore struct {
	path    string
	mu      sync.RWMutex
	devices map[string]models.Device
}

// NewFileStore creates a new file-based storage
func NewFileStore(path string) (*FileStore, error) {
	fs := &FileStore{
		path:    path,
		devices: make(map[string]models.Device),
	}

	if err := fs.load(); err != nil {
		return nil, fmt.Errorf("failed to load storage: %w", err)
	}

	log.Printf("[STORAGE] Initialized file store at %s", path)
	return fs, nil
}

// load loads devices from the file
func (fs *FileStore) load() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(fs.path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// If file doesn't exist, return without error
	if _, err := os.Stat(fs.path); os.IsNotExist(err) {
		return fs.saveLocked()
	}

	// Read and parse the file
	data, err := ioutil.ReadFile(fs.path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Empty file is ok
	if len(data) == 0 {
		return nil
	}

	var devices []models.Device
	if err := json.Unmarshal(data, &devices); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	for _, d := range devices {
		fs.devices[d.ID] = d
	}

	return nil
}

// saveLocked saves devices to file (must be called with lock held)
func (fs *FileStore) saveLocked() error {
	devices := make([]models.Device, 0, len(fs.devices))
	for _, d := range fs.devices {
		devices = append(devices, d)
	}

	data, err := json.MarshalIndent(devices, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := ioutil.WriteFile(fs.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// save saves devices to file (acquires lock)
func (fs *FileStore) save() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.saveLocked()
}

// Register registers a new device
func (fs *FileStore) Register(prefID, macID, location, osDetails string) (models.Device, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	id := prefID
	if id == "" {
		id = genID()
	}

	// Check if device already exists
	if _, exists := fs.devices[id]; exists {
		return models.Device{}, fmt.Errorf("device with ID %s already registered", id)
	}

	device := models.Device{
		ID:          id,
		Status:      "ACTIVE",
		RecoveryKey: "", // Will be updated when device sends it
		MacID:       macID,
		Location:    location,
		OSDetails:   osDetails,
		LastSeen:    time.Now(),
	}

	fs.devices[id] = device

	if err := fs.saveLocked(); err != nil {
		return models.Device{}, err
	}

	log.Printf("[STORAGE] Registered device: %s (MAC: %s)", id, macID)
	return device, nil
}

// Heartbeat updates the device's last seen time
func (fs *FileStore) Heartbeat(id string) (models.Device, bool) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	device, ok := fs.devices[id]
	if !ok {
		return models.Device{}, false
	}

	device.LastSeen = time.Now()
	fs.devices[id] = device

	if err := fs.saveLocked(); err != nil {
		log.Printf("[STORAGE] Failed to save after heartbeat: %v", err)
	}

	return device, true
}

// UpdateStatus updates the device status
func (fs *FileStore) UpdateStatus(id, status string) (models.Device, string, bool) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	device, ok := fs.devices[id]
	if !ok {
		return models.Device{}, "", false
	}

	oldStatus := device.Status
	device.Status = status
	device.LastSeen = time.Now()
	fs.devices[id] = device

	if err := fs.saveLocked(); err != nil {
		log.Printf("[STORAGE] Failed to save after status update: %v", err)
	}

	log.Printf("[STORAGE] Updated device %s status: %s -> %s", id, oldStatus, status)
	return device, oldStatus, true
}

// UpdateRecoveryKey updates the device's recovery key
func (fs *FileStore) UpdateRecoveryKey(id, recoveryKey string) (models.Device, bool) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	device, ok := fs.devices[id]
	if !ok {
		return models.Device{}, false
	}

	device.RecoveryKey = recoveryKey
	device.LastSeen = time.Now()
	fs.devices[id] = device

	if err := fs.saveLocked(); err != nil {
		log.Printf("[STORAGE] Failed to save recovery key: %v", err)
		return models.Device{}, false
	}

	log.Printf("[STORAGE] Updated recovery key for device: %s", id)
	return device, true
}

// GetDevice retrieves a device by ID
func (fs *FileStore) GetDevice(id string) (models.Device, bool) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	device, ok := fs.devices[id]
	return device, ok
}

// AllDevices returns all registered devices
func (fs *FileStore) AllDevices() []models.Device {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	devices := make([]models.Device, 0, len(fs.devices))
	for _, d := range fs.devices {
		devices = append(devices, d)
	}

	return devices
}

// Close closes the store (no-op for file-based storage)
func (fs *FileStore) Close() error {
	return nil
}

// genID generates a random device ID
func genID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		log.Printf("[STORAGE] Failed to generate random ID: %v", err)
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(b)
}
