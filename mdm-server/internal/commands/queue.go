package commands

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// MDM Command Payload Constants
const (
	RequestTypeDeviceLock      = "DeviceLock"
	RequestTypeEraseDevice     = "EraseDevice"
	RequestTypeDeviceLocation  = "DeviceLocation"
	RequestTypeEnableLostMode  = "EnableLostMode"
	RequestTypeDisableLostMode = "DisableLostMode"
	RequestTypeRestartDevice   = "RestartDevice"
	RequestTypeShutDownDevice  = "ShutDownDevice"
)

// Command represents a queued MDM command
type Command struct {
	CommandUUID string
	RequestType string
	Payload     map[string]interface{}
	CreatedAt   time.Time
}

// Queue manages pending commands for devices
type Queue struct {
	mu       sync.Mutex
	commands map[string][]Command // Map DeviceUDID -> []Command
}

func NewQueue() *Queue {
	return &Queue{
		commands: make(map[string][]Command),
	}
}

// Enqueue adds a command to the device's queue
func (q *Queue) Enqueue(udid string, requestType string, payload map[string]interface{}) string {
	q.mu.Lock()
	defer q.mu.Unlock()

	cmdUUID := uuid.New().String()
	cmd := Command{
		CommandUUID: cmdUUID,
		RequestType: requestType,
		Payload:     payload,
		CreatedAt:   time.Now(),
	}

	q.commands[udid] = append(q.commands[udid], cmd)
	return cmdUUID
}

// Next returns the next command for a device and removes it from the queue
func (q *Queue) Next(udid string) *Command {
	q.mu.Lock()
	defer q.mu.Unlock()

	cmds := q.commands[udid]
	if len(cmds) == 0 {
		return nil
	}

	cmd := cmds[0]
	q.commands[udid] = cmds[1:]
	return &cmd
}
