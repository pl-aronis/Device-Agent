package service

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"device-agent-linux/enforcement"
	"device-agent-linux/heartbeat"
	"device-agent-linux/tamper"
)

const (
	MaxOfflineDuration = 1 * time.Minute // update after testing
	LockCacheFile      = "/var/lib/device-agent-linux/lock.cache"

	// systemd writes this file when a shutdown/reboot is scheduled.
	// Its presence tells us the process is ending because the OS is going
	// down, not because an operator manually killed the agent.
	systemdShutdownSentinel = "/run/systemd/shutdown/scheduled"

	// NoLockFile is a sentinel dropped by an operator (e.g. uninstall.sh)
	// to signal that the upcoming process stop is intentional and should
	// NOT trigger a device lock. The file is removed after it is consumed.
	NoLockFile = "/var/lib/device-agent-linux/no-lock.flag"
)

func cacheLockCommand() {
	os.WriteFile(LockCacheFile, []byte("LOCK"), 0644)
}

func isLockCached() bool {
	if _, err := os.Stat(LockCacheFile); err == nil {
		return true
	}
	return false
}

// isSystemShuttingDown returns true when systemd has scheduled a
// shutdown or reboot. In that case we should NOT lock the device from
// defer — the machine is going down intentionally.
func isSystemShuttingDown() bool {
	_, err := os.Stat(systemdShutdownSentinel)
	return err == nil
}

// isIntentionalStop returns true when an operator has explicitly
// signalled that this process stop should not lock the device.
// The sentinel file is consumed (removed) so it cannot be reused.
func isIntentionalStop() bool {
	if _, err := os.Stat(NoLockFile); err != nil {
		return false
	}
	os.Remove(NoLockFile)
	return true
}

// Run starts the heartbeat loop. It accepts a context so the caller
// can cancel it externally. A deferred LockDevice call fires when the
// process is killed, unless a system shutdown/reboot is in progress.
func Run(ctx context.Context, ip string, port string) {
	// Capture SIGINT and SIGTERM so the defer below can execute.
	// Without this, the runtime would exit immediately on signal and
	// defer would NOT run.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Cancel the context when a signal arrives so the loop exits.
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		select {
		case sig := <-sigCh:
			log.Printf("[SERVICE] Received signal %s — shutting down", sig)
			cancel()
		case <-ctx.Done():
		}
	}()

	// Lock the device when the process exits, unless:
	//  a) the OS is shutting down / rebooting (machine powers off anyway), or
	//  b) an operator pre-created the no-lock sentinel for a planned stop.
	defer func() {
		signal.Stop(sigCh)
		if isSystemShuttingDown() {
			log.Println("[SERVICE] System shutdown/reboot detected — skipping defer lock")
			return
		}
		if isIntentionalStop() {
			log.Println("[SERVICE] Intentional stop flag detected — skipping defer lock")
			return
		}
		log.Println("[SERVICE] Process exiting — locking device via defer")
		enforcement.LockDevice(ip)
	}()

	lastSuccessfulHeartbeat := time.Now()

	for {
		// Check for cancellation before each iteration.
		select {
		case <-ctx.Done():
			log.Println("[SERVICE] Context cancelled — exiting service loop")
			return
		default:
		}

		action := heartbeat.SendHeartbeat(ip, port)
		log.Println("Heartbeat response: ", action)

		switch action {
		case "LOCK":
			log.Println("Policy violation → locking device")
			cacheLockCommand()
			enforcement.LockDevice(ip)
		case "WARNING":
			log.Println("Policy warning → displaying alert")
			// TODO
			// enforcement.ShowWarning()
		case "NONE":
			lastSuccessfulHeartbeat = time.Now()
		}

		tamper.CheckIntegrity()

		if isLockCached() {
			log.Println("[OFFLINE LOCK] Cached lock command found → locking device")
			enforcement.LockDevice(ip)
		}

		if time.Since(lastSuccessfulHeartbeat) > MaxOfflineDuration {
			log.Println("[FAIL-CLOSED] Backend unreachable for too long → locking device")
			cacheLockCommand()
			enforcement.LockDevice(ip)
		}

		// Sleep with context awareness so we wake up immediately on cancel.
		select {
		case <-ctx.Done():
			log.Println("[SERVICE] Context cancelled during sleep — exiting service loop")
			return
		case <-time.After(20 * time.Second):
		}
	}
}
