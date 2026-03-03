package app

import (
	"device-agent-windows/internal/config"
	"device-agent-windows/internal/device"
	"device-agent-windows/internal/logger"
	"time"
)

func (r *Runner) startHeartbeatLoop(mac string) {
	ticker := time.NewTicker(time.Duration(config.AppConfig.HeartbeatIntervalSeconds) * time.Second)

	for {
		<-ticker.C

		loc := device.GetLocation()
		resp, err := r.client.Heartbeat(mac, loc)
		if err != nil {
			logger.Error(err)
			continue
		}

		if resp.ShouldLock {
			ticker.Stop()
			r.executeLock(mac)
			return
		}
	}
}
