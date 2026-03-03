package backend

type RegisterResponse struct {
	AgentID string `json:"agent_id"`
}

type ReAuthResponse struct {
	AgentID    string `json:"agent_id"`
	RecoveryID string `json:"recovery_id"`
}

type HeartbeatResponse struct {
	ShouldLock bool `json:"should_lock"`
}
