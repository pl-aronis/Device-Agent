package app

import (
	"device-agent-windows/internal/backend"
	"device-agent-windows/internal/config"
	"device-agent-windows/internal/device"
	"device-agent-windows/internal/enforcement"
	"device-agent-windows/internal/logger"
	"device-agent-windows/internal/state"
	"os/exec"
)

type Runner struct {
	client *backend.Client
	store  *state.Store
}

func NewRunner() *Runner {
	return &Runner{
		client: backend.NewClient(config.AppConfig.BaseURL),
		store:  state.NewStore(config.AppConfig.StateFile),
	}
}

func (r *Runner) Start() {
	mac := device.GetMacID()

	if !r.store.IsRegistered() {
		r.register(mac)
	} else {
		r.reAuth(mac)
	}

	r.startHeartbeatLoop(mac)
}

func (r *Runner) register(mac string) {
	info := device.CollectDeviceInfo()
	resp, err := r.client.Register(info)
	if err != nil {
		logger.Fatal(err)
	}
	r.store.SaveAgentID(resp.AgentID)
}

func (r *Runner) reAuth(mac string) {
	resp, err := r.client.ReAuthenticate(mac)
	if err != nil {
		logger.Fatal(err)
	}
	r.store.SaveAgentID(resp.AgentID)

	if resp.RecoveryID != "" {
		enforcement.DeleteProtector(resp.RecoveryID)
	}
}

func (r *Runner) executeLock(mac string) {
	execImpl := &enforcement.CommandExecutor{}
	bl := enforcement.NewBitlocker(execImpl)

	key, id, err := bl.Enforce()
	if err != nil {
		r.client.SendLockFailure(mac, err.Error())
		return
	}

	r.client.SendLockSuccess(mac, key, id)
	exec.Command("shutdown", "/r", "/t", "0").Start()
}
