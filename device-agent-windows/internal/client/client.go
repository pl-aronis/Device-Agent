package client

import (
	"bytes"
	"device-agent-windows/internal/config"
	"device-agent-windows/internal/helper"
	"device-agent-windows/internal/model"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func newRegisterBackendClient(cfg *config.Config) *model.RegisterClientRequest {
	location, err := helper.GetLocation()
	if err != nil {
		location = model.GeoResponse{}
	}
	return &model.RegisterClientRequest{
		DeviceID:  cfg.DeviceDetails.DeviceID,
		AgentId:   cfg.DeviceDetails.MacID,
		Location:  location,
		OSDetails: cfg.DeviceDetails.OSDetails,
	}
}

func newBackendClient(cfg *config.Config) *model.HeartbeatRequest {
	location, err := helper.GetLocation()
	if err != nil {
		location = model.GeoResponse{}
	}
	return &model.HeartbeatRequest{
		DeviceID: cfg.DeviceDetails.DeviceID,
		AgentId:  cfg.DeviceDetails.MacID,
		Location: location,
	}
}

func RegisterDevice(cfg *config.Config) error {
	req := newRegisterBackendClient(cfg)
	backendURL := helper.ConstructBackendURL(cfg) + model.RegisterEndpoint

	reqBody, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest("POST", backendURL, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("register returned %d: %s", resp.StatusCode, string(b))
	}

	var registerResp model.RegisterClientResponse
	if err := json.NewDecoder(resp.Body).Decode(&registerResp); err != nil {
		return err
	}

	if registerResp.DeviceID != "" {
		cfg.DeviceDetails.DeviceID = registerResp.DeviceID
	}
	cfg.DeviceDetails.Status = registerResp.Status

	return nil
}

func SendHeartbeat(cfg *config.Config) (string, error) {
	req := newBackendClient(cfg)
	backendURL := helper.ConstructBackendURL(cfg) + model.HeartbeatEndpoint

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "NONE", err
	}

	httpReq, err := http.NewRequest("POST", backendURL, bytes.NewReader(reqBody))
	if err != nil {
		return "NONE", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.ContentLength = int64(len(reqBody))

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		fmt.Printf("[HEARTBEAT] Request failed: %v", err)
		return "NONE", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "NONE", fmt.Errorf("heartbeat returned %d: %s", resp.StatusCode, string(b))
	}

	var hbResp model.HeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&hbResp); err != nil {
		return "NONE", err
	}

	if hbResp.Status != "" {
		cfg.DeviceDetails.Status = hbResp.Status
	} else {
		cfg.DeviceDetails.Status = hbResp.Action
	}
	return hbResp.Action, nil
}

func SendRecoveryKeyUpdate(cfg *config.Config, protectorID, recoveryKey string) error {
	req := model.RecoveryKeyUpdateRequest{
		DeviceID:    cfg.DeviceDetails.DeviceID,
		ProtectorID: protectorID,
		RecoveryKey: recoveryKey,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return err
	}

	backendURL := helper.ConstructBackendURL(cfg) + model.RecoveryKeyEndpoint
	httpReq, err := http.NewRequest("POST", backendURL, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("recovery-key update returned %d: %s", resp.StatusCode, string(b))
	}

	return nil
}
