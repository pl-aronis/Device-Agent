package service

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
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

// isSystemShuttingDown returns true when systemd is performing a
// shutdown or reboot. In that case we should NOT lock the device from
// defer — the machine is going down intentionally.
func isSystemShuttingDown() bool {
	// Primary: shutdown.target is active during any shutdown/reboot
	// (works for both immediate and delayed shutdowns).
	if err := exec.Command("systemctl", "is-active", "--quiet", "shutdown.target").Run(); err == nil {
		log.Println("System is shutting down")
		return true
	}
	if err := exec.Command("systemctl", "is-active", "--quiet", "poweroff.target").Run(); err == nil {
		log.Println("System is powering off")
		return true
	}
	if err := exec.Command("systemctl", "is-active", "--quiet", "reboot.target").Run(); err == nil {
		log.Println("System is rebooting")
		return true
	}
	// Fallback: sentinel file exists only for delayed shutdowns.
	_, err := os.Stat(systemdShutdownSentinel)
	return err == nil
}

// notifyBackendOffline pings the backend so it knows the agent process
// is about to go down unexpectedly (i.e. not a planned stop).
// A short timeout is used so this never blocks the actual lock.
func notifyBackendOffline(ip, port string) {
	url := fmt.Sprintf("http://%s:%s/ping", ip, port)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("[SERVICE] Failed to ping backend on exit: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("[SERVICE] Backend notified of agent exit (status %d)", resp.StatusCode)
}

// isIntentionalStop returns true when an operator has explicitly
// signalled that this process stop should not lock the device.
// The sentinel file is consumed (removed) so it cannot be reused.
func isIntentionalStop() bool {
	if _, err := os.Stat(NoLockFile); err != nil {
		return false
	}
	if err := os.Remove(NoLockFile); err != nil {
		log.Printf("[SERVICE] Failed to remove no-lock file: %v", err)
	}
	return true
}

// Run starts the heartbeat loop. It accepts a context so the caller
// can cancel it externally. A deferred LockDevice call fires when the
// process is killed, unless a system shutdown/reboot is in progress.
func Run(ctx context.Context, deviceId, ip, port string) {
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
		notifyBackendOffline(ip, port)
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

		action := heartbeat.SendHeartbeat(deviceId, ip, port)
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
		case "ACTIVE":
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
