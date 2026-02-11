package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CommandStatus represents the status of an MDM command
type CommandStatus string

const (
	CommandStatusPending      CommandStatus = "pending"
	CommandStatusSent         CommandStatus = "sent"
	CommandStatusAcknowledged CommandStatus = "acknowledged"
	CommandStatusError        CommandStatus = "error"
	CommandStatusNotNow       CommandStatus = "notnow"
)

// Command represents an MDM command in the queue
type Command struct {
	ID          string                   `json:"id"`
	TenantID    string                   `json:"tenant_id"`
	DeviceID    string                   `json:"device_id"`
	CommandUUID string                   `json:"command_uuid"`
	RequestType string                   `json:"request_type"`
	Payload     map[string]interface{}   `json:"payload,omitempty"`
	Status      CommandStatus            `json:"status"`
	ErrorChain  []map[string]interface{} `json:"error_chain,omitempty"`
	CreatedAt   time.Time                `json:"created_at"`
	SentAt      *time.Time               `json:"sent_at,omitempty"`
	RespondedAt *time.Time               `json:"responded_at,omitempty"`
}

// CommandStore handles command database operations
type CommandStore struct {
	db *DB
}

// NewCommandStore creates a new command store
func NewCommandStore(db *DB) *CommandStore {
	return &CommandStore{db: db}
}

// Enqueue adds a command to the queue
func (s *CommandStore) Enqueue(tenantID, deviceID, requestType string, payload map[string]interface{}) (string, error) {
	cmdUUID := uuid.New().String()
	id := uuid.New().String()

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO commands (id, tenant_id, device_id, command_uuid, request_type, payload_json, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, id, tenantID, deviceID, cmdUUID, requestType, string(payloadJSON), CommandStatusPending, time.Now())

	if err != nil {
		return "", fmt.Errorf("failed to enqueue command: %w", err)
	}

	return cmdUUID, nil
}

// Next retrieves the next pending command for a device
func (s *CommandStore) Next(deviceID string) (*Command, error) {
	cmd := &Command{}
	var payloadJSON, errorChainJSON sql.NullString
	var sentAt, respondedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, tenant_id, device_id, command_uuid, request_type, payload_json, status,
		       error_chain_json, created_at, sent_at, responded_at
		FROM commands
		WHERE device_id = ? AND status = ?
		ORDER BY created_at ASC
		LIMIT 1
	`, deviceID, CommandStatusPending).Scan(
		&cmd.ID, &cmd.TenantID, &cmd.DeviceID, &cmd.CommandUUID, &cmd.RequestType,
		&payloadJSON, &cmd.Status, &errorChainJSON, &cmd.CreatedAt, &sentAt, &respondedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get next command: %w", err)
	}

	if payloadJSON.Valid {
		json.Unmarshal([]byte(payloadJSON.String), &cmd.Payload)
	}
	if errorChainJSON.Valid {
		json.Unmarshal([]byte(errorChainJSON.String), &cmd.ErrorChain)
	}
	if sentAt.Valid {
		cmd.SentAt = &sentAt.Time
	}
	if respondedAt.Valid {
		cmd.RespondedAt = &respondedAt.Time
	}

	return cmd, nil
}

// NextByUDID retrieves the next pending command for a device by UDID
func (s *CommandStore) NextByUDID(udid string) (*Command, error) {
	// First find the device ID
	var deviceID string
	err := s.db.QueryRow(`SELECT id FROM devices WHERE udid = ? AND is_enrolled = 1 LIMIT 1`, udid).Scan(&deviceID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return s.Next(deviceID)
}

// MarkSent marks a command as sent
func (s *CommandStore) MarkSent(commandUUID string) error {
	_, err := s.db.Exec(`
		UPDATE commands SET status = ?, sent_at = ? WHERE command_uuid = ?
	`, CommandStatusSent, time.Now(), commandUUID)

	return err
}

// MarkAcknowledged marks a command as acknowledged
func (s *CommandStore) MarkAcknowledged(commandUUID string) error {
	_, err := s.db.Exec(`
		UPDATE commands SET status = ?, responded_at = ? WHERE command_uuid = ?
	`, CommandStatusAcknowledged, time.Now(), commandUUID)

	return err
}

// MarkError marks a command as failed with error chain
func (s *CommandStore) MarkError(commandUUID string, errorChain []map[string]interface{}) error {
	errorJSON, _ := json.Marshal(errorChain)

	_, err := s.db.Exec(`
		UPDATE commands SET status = ?, error_chain_json = ?, responded_at = ? WHERE command_uuid = ?
	`, CommandStatusError, string(errorJSON), time.Now(), commandUUID)

	return err
}

// MarkNotNow marks a command as NotNow (device busy)
func (s *CommandStore) MarkNotNow(commandUUID string) error {
	// Reset to pending so it will be retried
	_, err := s.db.Exec(`
		UPDATE commands SET status = ? WHERE command_uuid = ?
	`, CommandStatusPending, commandUUID)

	return err
}

// GetByUUID retrieves a command by UUID
func (s *CommandStore) GetByUUID(commandUUID string) (*Command, error) {
	cmd := &Command{}
	var payloadJSON, errorChainJSON sql.NullString
	var sentAt, respondedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, tenant_id, device_id, command_uuid, request_type, payload_json, status,
		       error_chain_json, created_at, sent_at, responded_at
		FROM commands WHERE command_uuid = ?
	`, commandUUID).Scan(
		&cmd.ID, &cmd.TenantID, &cmd.DeviceID, &cmd.CommandUUID, &cmd.RequestType,
		&payloadJSON, &cmd.Status, &errorChainJSON, &cmd.CreatedAt, &sentAt, &respondedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get command: %w", err)
	}

	if payloadJSON.Valid {
		json.Unmarshal([]byte(payloadJSON.String), &cmd.Payload)
	}
	if sentAt.Valid {
		cmd.SentAt = &sentAt.Time
	}
	if respondedAt.Valid {
		cmd.RespondedAt = &respondedAt.Time
	}

	return cmd, nil
}

// ListByDevice returns all commands for a device
func (s *CommandStore) ListByDevice(deviceID string, limit int) ([]*Command, error) {
	rows, err := s.db.Query(`
		SELECT id, tenant_id, device_id, command_uuid, request_type, status, created_at, sent_at, responded_at
		FROM commands WHERE device_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, deviceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list commands: %w", err)
	}
	defer rows.Close()

	var commands []*Command
	for rows.Next() {
		cmd := &Command{}
		var sentAt, respondedAt sql.NullTime

		if err := rows.Scan(
			&cmd.ID, &cmd.TenantID, &cmd.DeviceID, &cmd.CommandUUID, &cmd.RequestType,
			&cmd.Status, &cmd.CreatedAt, &sentAt, &respondedAt,
		); err != nil {
			return nil, err
		}

		if sentAt.Valid {
			cmd.SentAt = &sentAt.Time
		}
		if respondedAt.Valid {
			cmd.RespondedAt = &respondedAt.Time
		}

		commands = append(commands, cmd)
	}

	return commands, nil
}

// PendingCount returns the number of pending commands for a device
func (s *CommandStore) PendingCount(deviceID string) (int, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM commands WHERE device_id = ? AND status = ?
	`, deviceID, CommandStatusPending).Scan(&count)

	return count, err
}
