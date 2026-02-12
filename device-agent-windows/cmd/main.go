package main

import (
	"device-agent-windows/internal/client"
	"device-agent-windows/internal/config"
	"device-agent-windows/internal/enforcement"
	"device-agent-windows/internal/helper"
	"log"
	"strings"
	"time"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	state, err := config.LoadState(cfg.AgentConfig.StateFilePath)
	if err != nil {
		log.Fatalf("failed to load state: %v", err)
	}

	applyStateToConfig(cfg, state)
	helper.SetDeviceDetails(cfg)
	syncStateFromConfig(cfg, state)
	if err := persistState(cfg, state); err != nil {
		log.Printf("failed to persist startup state: %v", err)
	}

	if !state.Registered {
		registerWithRetry(cfg, state)
	} else {
		log.Printf("Using persisted registration, device_id=%s", cfg.DeviceDetails.DeviceID)
	}

	startHeartbeatLoop(cfg, state)
}

func applyStateToConfig(cfg *config.Config, state *config.PersistedState) {
	if state.DeviceID != "" {
		cfg.DeviceDetails.DeviceID = state.DeviceID
	}
	if state.MacID != "" {
		cfg.DeviceDetails.MacID = state.MacID
	}
	if state.OSDetails != "" {
		cfg.DeviceDetails.OSDetails = state.OSDetails
	}
	if state.Status != "" {
		cfg.DeviceDetails.Status = state.Status
	}
}

func syncStateFromConfig(cfg *config.Config, state *config.PersistedState) {
	state.DeviceID = cfg.DeviceDetails.DeviceID
	state.MacID = cfg.DeviceDetails.MacID
	state.OSDetails = cfg.DeviceDetails.OSDetails
	state.Status = cfg.DeviceDetails.Status
}

func persistState(cfg *config.Config, state *config.PersistedState) error {
	syncStateFromConfig(cfg, state)
	return config.SaveState(cfg.AgentConfig.StateFilePath, state)
}

func registrationRetryDuration(cfg *config.Config) time.Duration {
	if cfg.AgentConfig.RegistrationRetrySeconds <= 0 {
		return 5 * time.Second
	}
	return time.Duration(cfg.AgentConfig.RegistrationRetrySeconds) * time.Second
}

func heartbeatDuration(cfg *config.Config) time.Duration {
	if cfg.AgentConfig.HeartbeatIntervalSeconds <= 0 {
		return 10 * time.Second
	}
	return time.Duration(cfg.AgentConfig.HeartbeatIntervalSeconds) * time.Second
}

func registerWithRetry(cfg *config.Config, state *config.PersistedState) {
	retry := registrationRetryDuration(cfg)
	for {
		if err := client.RegisterDevice(cfg); err != nil {
			log.Printf("registration failed: %v (retry in %v)", err, retry)
			time.Sleep(retry)
			continue
		}

		state.Registered = true
		state.Status = cfg.DeviceDetails.Status
		if err := persistState(cfg, state); err != nil {
			log.Printf("failed to persist registration state: %v", err)
		}
		log.Printf("Device registered successfully, device_id=%s", cfg.DeviceDetails.DeviceID)
		return
	}
}

func shouldReRegister(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "404")
}

func normalizeStatus(v string) string {
	return strings.ToUpper(strings.TrimSpace(v))
}

func restartDeviceNow() {
	if _, err := helper.RunCommand("shutdown", "/r", "/t", "0"); err != nil {
		log.Printf("failed to restart device: %v", err)
	}
}

func notifyRecoveryKeyWithRetry(cfg *config.Config, protectorID, recoveryKey string) bool {
	const attempts = 3
	const backoff = 2 * time.Second

	if protectorID == "" || recoveryKey == "" {
		return false
	}

	for i := 1; i <= attempts; i++ {
		if err := client.SendRecoveryKeyUpdate(cfg, protectorID, recoveryKey); err == nil {
			log.Printf("recovery key update sent to backend for protector_id=%s", protectorID)
			return true
		} else {
			log.Printf("failed to send recovery key update (attempt %d/%d): %v", i, attempts, err)
		}
		time.Sleep(backoff)
	}
	return false
}

func lockAndReboot(cfg *config.Config, state *config.PersistedState) {
	recovery, err := enforcement.EnforceDeviceLockWithRecovery()
	if err != nil {
		log.Printf("failed to enforce device lock: %v", err)
		return
	}

	state.LockApplied = true
	state.Status = "LOCK"
	if recovery != nil {
		state.ManagedRecoveryKeyID = recovery.ID
		state.ManagedRecoveryKey = recovery.Key
		notifyRecoveryKeyWithRetry(cfg, recovery.ID, recovery.Key)
	}
	if err := persistState(cfg, state); err != nil {
		log.Printf("failed to persist lock state before reboot: %v", err)
	}

	if err := enforcement.ForceRecoveryAndReboot(); err != nil {
		log.Printf("failed to force recovery reboot: %v", err)
	}
}

func unlockAndResume(cfg *config.Config, state *config.PersistedState) {
	if err := enforcement.ReleaseDeviceLock(state.ManagedRecoveryKeyID); err != nil {
		log.Printf("failed to release lock state: %v", err)
		return
	}

	state.LockApplied = false
	state.ManagedRecoveryKeyID = ""
	state.ManagedRecoveryKey = ""
	state.Status = "ACTIVE"
	if err := persistState(cfg, state); err != nil {
		log.Printf("failed to persist unlock state: %v", err)
	}
	log.Println("Device unlocked and agent-managed recovery protector removed")
}

func startHeartbeatLoop(cfg *config.Config, state *config.PersistedState) {
	interval := heartbeatDuration(cfg)
	log.Printf("Starting heartbeat poll every %d seconds", int(interval.Seconds()))

	for {
		action, err := client.SendHeartbeat(cfg)
		if err != nil {
			log.Printf("heartbeat failed: %v", err)
			if shouldReRegister(err) {
				log.Println("backend does not recognize this device_id; re-registering")
				state.Registered = false
				_ = persistState(cfg, state)
				registerWithRetry(cfg, state)
			}
			time.Sleep(interval)
			continue
		}

		nAction := normalizeStatus(action)
		nStatus := normalizeStatus(cfg.DeviceDetails.Status)
		state.LastHeartbeatAction = nAction
		if nStatus != "" {
			state.Status = nStatus
		} else if nAction != "" {
			state.Status = nAction
		}
		if err := persistState(cfg, state); err != nil {
			log.Printf("failed to persist heartbeat state: %v", err)
		}

		isLockedSignal := nAction == "LOCK" || nStatus == "LOCK"
		isActiveSignal := nAction == "ACTIVE" || nStatus == "ACTIVE"

		if isLockedSignal {
			log.Println("LOCK signal received; stopping heartbeat and enforcing lock")
			if state.LockApplied {
				log.Println("Device already locked and still not unlocked by backend; restarting device")
				restartDeviceNow()
				return
			}

			lockAndReboot(cfg, state)
			return
		}

		if isActiveSignal && state.LockApplied {
			log.Println("ACTIVE signal received for previously locked device; disabling lock")
			unlockAndResume(cfg, state)
		}

		time.Sleep(interval)
	}
}
