package config

type Config struct {
	BaseURL                  string
	HeartbeatIntervalSeconds int
	EncryptionTimeoutMinutes int
	ProtectionTimeoutMinutes int
	ForceRecoverySleepSec    int
	StateFile                string
}

var AppConfig = &Config{
	BaseURL:                  "https://your-backend.com",
	HeartbeatIntervalSeconds: 30,
	EncryptionTimeoutMinutes: 60,
	ProtectionTimeoutMinutes: 2,
	ForceRecoverySleepSec:    60,
	StateFile:                "agent_state.json",
}
