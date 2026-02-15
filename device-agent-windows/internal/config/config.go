package config

import (
	"device-agent-windows/internal/model"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	BackendConfig struct {
		BackendIp   string `json:"backend_ip" default:"localhost" env:"BACKEND_IP"`
		BackendPort string `json:"backend_port" default:"8080" env:"BACKEND_PORT"`
	}
	AgentConfig struct {
		HeartbeatIntervalSeconds int    `json:"heartbeat_interval_seconds" default:"10" env:"HEARTBEAT_INTERVAL_SECONDS"`
		RegistrationRetrySeconds int    `json:"registration_retry_seconds" default:"5" env:"REGISTRATION_RETRY_SECONDS"`
		StateFilePath            string `json:"state_file_path" default:"C:\\ProgramData\\DeviceAgent\\state.json" env:"STATE_FILE_PATH"`
	}

	DeviceDetails struct {
		DeviceID  string
		MacID     string
		Location  model.GeoResponse
		OSDetails string
		Status    string
	}
}

type PersistedState struct {
	Registered            bool      `json:"registered"`
	DeviceID              string    `json:"device_id"`
	MacID                 string    `json:"mac_id"`
	OSDetails             string    `json:"os_details"`
	Status                string    `json:"status"`
	LockApplied           bool      `json:"lock_applied"`
	ManagedRecoveryKeyID  string    `json:"managed_recovery_key_id"`
	ManagedRecoveryKey    string    `json:"managed_recovery_key"`
	LastHeartbeatAction   string    `json:"last_heartbeat_action"`
	LastUpdatedUnixSecond int64     `json:"last_updated_unix_second"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func New() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func LoadState(path string) (*PersistedState, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &PersistedState{}, nil
		}
		return nil, err
	}

	var s PersistedState
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func SaveState(path string, state *PersistedState) error {
	state.UpdatedAt = time.Now().UTC()
	state.LastUpdatedUnixSecond = state.UpdatedAt.Unix()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
