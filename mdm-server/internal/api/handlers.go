package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"mdm-server/internal/apns"
	"mdm-server/internal/store"

	"howett.net/plist"
)

// CheckinMessage struct for decoding plist messages
type CheckinMessage struct {
	MessageType string `plist:"MessageType"`
	Topic       string `plist:"Topic"`
	UDID        string `plist:"UDID"`
	Token       []byte `plist:"Token,omitempty"` // For TokenUpdate
	PushMagic   string `plist:"PushMagic,omitempty"`

	// Additional fields from Authenticate
	BuildVersion string `plist:"BuildVersion,omitempty"`
	OSVersion    string `plist:"OSVersion,omitempty"`
	ProductName  string `plist:"ProductName,omitempty"`
	SerialNumber string `plist:"SerialNumber,omitempty"`
	Model        string `plist:"Model,omitempty"`
	ModelName    string `plist:"ModelName,omitempty"`
	DeviceName   string `plist:"DeviceName,omitempty"`
}

// CheckinHandler handles MDM check-in requests
type CheckinHandler struct {
	deviceStore  *store.DeviceStore
	commandStore *store.CommandStore
	tenantStore  *store.TenantStore
}

// NewCheckinHandler creates a new checkin handler
func NewCheckinHandler(ds *store.DeviceStore, cs *store.CommandStore, ts *store.TenantStore) *CheckinHandler {
	return &CheckinHandler{
		deviceStore:  ds,
		commandStore: cs,
		tenantStore:  ts,
	}
}

// ServeHTTP handles the /mdm/checkin endpoint
func (h *CheckinHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Received MDM Check-in")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", 500)
		return
	}

	var msg CheckinMessage
	if _, err := plist.Unmarshal(body, &msg); err != nil {
		log.Printf("Failed to unmarshal plist: %v", err)
		http.Error(w, "Invalid plist", 400)
		return
	}

	log.Printf("Check-in Type: %s, Device UDID: %s", msg.MessageType, msg.UDID)

	// Determine tenant from topic or certificate
	tenantID := h.resolveTenant(r, msg.Topic)
	if tenantID == "" {
		log.Printf("Could not resolve tenant for device %s (topic: %s)", msg.UDID, msg.Topic)
		http.Error(w, "Could not determine tenant", http.StatusBadRequest)
		return
	}

	switch msg.MessageType {
	case "Authenticate":
		h.handleAuthenticate(w, msg, tenantID)

	case "TokenUpdate":
		h.handleTokenUpdate(w, msg, tenantID)

	case "CheckOut":
		h.handleCheckOut(w, msg)

	default:
		log.Printf("Unknown MessageType: %s", msg.MessageType)
		w.WriteHeader(400)
	}
}

// handleAuthenticate handles the first step of enrollment
func (h *CheckinHandler) handleAuthenticate(w http.ResponseWriter, msg CheckinMessage, tenantID string) {
	log.Printf("Processing Authenticate for device %s (serial: %s)", msg.UDID, msg.SerialNumber)

	// In production, you might validate a challenge password here
	// For now, accept all authentication requests

	w.WriteHeader(200)
	w.Write([]byte(""))
}

// handleTokenUpdate handles device token registration
func (h *CheckinHandler) handleTokenUpdate(w http.ResponseWriter, msg CheckinMessage, tenantID string) {
	log.Printf("Processing TokenUpdate for device %s", msg.UDID)

	device, err := h.deviceStore.SaveDevice(tenantID, msg.UDID, msg.Token, msg.PushMagic)
	if err != nil {
		log.Printf("Failed to save device: %v", err)
		http.Error(w, "Failed to save device", 500)
		return
	}

	// Update device info if available
	if msg.SerialNumber != "" || msg.Model != "" {
		info := map[string]interface{}{
			"SerialNumber": msg.SerialNumber,
			"Model":        msg.Model,
			"ModelName":    msg.ModelName,
			"DeviceName":   msg.DeviceName,
			"OSVersion":    msg.OSVersion,
			"BuildVersion": msg.BuildVersion,
			"ProductName":  msg.ProductName,
		}
		h.deviceStore.UpdateDeviceInfo(device.ID, info)
	}

	log.Printf("Device %s enrolled successfully (ID: %s)", msg.UDID, device.ID)
	w.WriteHeader(200)
	w.Write([]byte(""))
}

// handleCheckOut handles device unenrollment
func (h *CheckinHandler) handleCheckOut(w http.ResponseWriter, msg CheckinMessage) {
	log.Printf("Device %s is checking out (unenrolling)", msg.UDID)

	if err := h.deviceStore.RemoveDevice(msg.UDID); err != nil {
		log.Printf("Failed to remove device: %v", err)
	}

	w.WriteHeader(200)
	w.Write([]byte(""))
}

// resolveTenant determines the tenant from the request
func (h *CheckinHandler) resolveTenant(r *http.Request, topic string) string {
	// Try to get tenant from URL path (e.g., /mdm/checkin/{tenantID})
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) >= 4 && parts[3] != "" {
		return parts[3]
	}

	// Try to resolve from APNs topic
	if topic != "" {
		tenants, err := h.tenantStore.List()
		if err == nil {
			for _, t := range tenants {
				if t.APNsTopic == topic {
					log.Printf("Resolved tenant %s from APNs topic %s", t.ID, topic)
					return t.ID
				}
			}
		}
	}

	// Fallback: use the first available tenant
	tenants, err := h.tenantStore.List()
	if err == nil && len(tenants) > 0 {
		log.Printf("Using first available tenant: %s (%s)", tenants[0].ID, tenants[0].Name)
		return tenants[0].ID
	}

	return ""
}

// ConnectHandler handles the /mdm/connect endpoint where devices fetch commands
type ConnectHandler struct {
	commandStore *store.CommandStore
	deviceStore  *store.DeviceStore
}

// NewConnectHandler creates a new connect handler
func NewConnectHandler(cs *store.CommandStore, ds *store.DeviceStore) *ConnectHandler {
	return &ConnectHandler{
		commandStore: cs,
		deviceStore:  ds,
	}
}

// ServeHTTP handles MDM connect requests
func (h *ConnectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Received MDM Connect Request")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", 500)
		return
	}

	var report struct {
		UDID        string                   `plist:"UDID"`
		Status      string                   `plist:"Status"`
		CommandUUID string                   `plist:"CommandUUID,omitempty"`
		ErrorChain  []map[string]interface{} `plist:"ErrorChain,omitempty"`
	}
	plist.Unmarshal(bodyBytes, &report)

	udid := report.UDID
	if udid == "" {
		// Fallback for testing
		udid = "unknown"
	}

	log.Printf("Device %s connected. Status: %s", udid, report.Status)

	// Update last seen
	if device, ok := h.deviceStore.GetDevice(udid); ok {
		h.deviceStore.UpdateLastSeen(device.ID)
	}

	// Handle command response
	if report.CommandUUID != "" {
		switch report.Status {
		case "Acknowledged":
			h.commandStore.MarkAcknowledged(report.CommandUUID)
			log.Printf("Command %s acknowledged", report.CommandUUID)
		case "Error":
			h.commandStore.MarkError(report.CommandUUID, report.ErrorChain)
			log.Printf("Command %s failed: %v", report.CommandUUID, report.ErrorChain)
		case "NotNow":
			h.commandStore.MarkNotNow(report.CommandUUID)
			log.Printf("Command %s: device busy, will retry", report.CommandUUID)
		}
	}

	// Fetch next command
	cmd, err := h.commandStore.NextByUDID(udid)
	if err != nil {
		log.Printf("Error fetching command: %v", err)
		w.WriteHeader(200)
		return
	}

	if cmd == nil {
		// No commands pending
		w.WriteHeader(200)
		return
	}

	log.Printf("Sending command %s (%s) to device", cmd.RequestType, cmd.CommandUUID)

	// Mark as sent
	h.commandStore.MarkSent(cmd.CommandUUID)

	// Construct Command Plist
	cmdPlist := map[string]interface{}{
		"CommandUUID": cmd.CommandUUID,
		"Command": map[string]interface{}{
			"RequestType": cmd.RequestType,
		},
	}

	// Merge payload
	cmdDict := cmdPlist["Command"].(map[string]interface{})
	for k, v := range cmd.Payload {
		cmdDict[k] = v
	}

	w.Header().Set("Content-Type", "application/xml")
	encoder := plist.NewEncoder(w)
	encoder.Encode(cmdPlist)
}

// AdminHandler handles admin API requests
type AdminHandler struct {
	deviceStore  *store.DeviceStore
	commandStore *store.CommandStore
	tenantStore  *store.TenantStore
	apnsPool     *apns.ClientPool
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(ds *store.DeviceStore, cs *store.CommandStore, ts *store.TenantStore, pool *apns.ClientPool) *AdminHandler {
	return &AdminHandler{
		deviceStore:  ds,
		commandStore: cs,
		tenantStore:  ts,
		apnsPool:     pool,
	}
}

// ListTenants returns all tenants
func (h *AdminHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := h.tenantStore.List()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(tenants)
}

// CreateTenant creates a new tenant
func (h *AdminHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string `json:"name"`
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}

	tenant, err := h.tenantStore.Create(req.Name, req.Domain)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(201)
	json.NewEncoder(w).Encode(tenant)
}

// ListDevices returns all enrolled devices for a tenant
func (h *AdminHandler) ListDevices(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")

	var devices []*store.Device
	var err error

	if tenantID != "" {
		devices, err = h.deviceStore.ListByTenant(tenantID)
	} else {
		devices = h.deviceStore.ListDevices()
	}

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(devices)
}

// DeviceAction handles POST requests to perform actions on devices
func (h *AdminHandler) DeviceAction(w http.ResponseWriter, r *http.Request) {
	// Path: /api/devices/{udid}/{action} or /api/tenants/{tenantID}/devices/{udid}/{action}
	parts := strings.Split(r.URL.Path, "/")

	var udid, action, tenantID string

	// Parse path
	if len(parts) >= 5 && parts[2] == "devices" {
		udid = parts[3]
		action = parts[4]
	} else if len(parts) >= 7 && parts[2] == "tenants" && parts[4] == "devices" {
		tenantID = parts[3]
		udid = parts[5]
		action = parts[6]
	} else {
		http.Error(w, "Invalid path", 400)
		return
	}

	// Handle GET /api/devices/{udid}/commands
	if action == "commands" && r.Method == http.MethodGet {
		device, ok := h.deviceStore.GetDevice(udid)
		if !ok || device == nil {
			http.Error(w, `{"status":"error","message":"Device not found"}`, 404)
			return
		}

		commands, err := h.commandStore.ListByDevice(device.ID, 50)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"status":"error","message":"%s"}`, err), 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(commands)
		return
	}

	// Find device
	var device *store.Device
	var ok bool

	if tenantID != "" {
		device, _ = h.deviceStore.GetByUDID(tenantID, udid)
		ok = device != nil
	} else {
		device, ok = h.deviceStore.GetDevice(udid)
	}

	if !ok || device == nil {
		http.Error(w, "Device not found", 404)
		return
	}

	var cmdUUID string
	var err error

	switch action {
	case "lock":
		payload := map[string]interface{}{
			"PIN":     "123456",
			"Message": "This device has been locked by IT.",
		}
		cmdUUID, err = h.commandStore.Enqueue(device.TenantID, device.ID, "DeviceLock", payload)

	case "locate":
		cmdUUID, err = h.commandStore.Enqueue(device.TenantID, device.ID, "DeviceLocation", nil)

	case "lostmode":
		payload := map[string]interface{}{
			"Message":     "Lost Device. Please return.",
			"PhoneNumber": "555-123-4567",
			"Footnote":    "Property of Organization",
		}
		cmdUUID, err = h.commandStore.Enqueue(device.TenantID, device.ID, "EnableLostMode", payload)

	case "disablelostmode":
		cmdUUID, err = h.commandStore.Enqueue(device.TenantID, device.ID, "DisableLostMode", nil)

	case "wipe":
		payload := map[string]interface{}{
			"PIN": "123456",
		}
		cmdUUID, err = h.commandStore.Enqueue(device.TenantID, device.ID, "EraseDevice", payload)

	case "deviceinfo":
		queries := []string{
			"DeviceName", "OSVersion", "BuildVersion", "ModelName", "Model",
			"ProductName", "SerialNumber", "UDID", "WiFiMAC", "BluetoothMAC",
		}
		payload := map[string]interface{}{"Queries": queries}
		cmdUUID, err = h.commandStore.Enqueue(device.TenantID, device.ID, "DeviceInformation", payload)

	default:
		http.Error(w, "Unknown action", 400)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to enqueue command: %v", err), 500)
		return
	}

	// Trigger APNs Push
	if h.apnsPool != nil {
		go func() {
			if err := h.apnsPool.SendPush(device.TenantID, device.PushToken, device.PushMagic); err != nil {
				log.Printf("Failed to send push to %s: %v", udid, err)
			}
		}()
	}

	w.WriteHeader(200)
	fmt.Fprintf(w, `{"status":"ok", "command_uuid":"%s"}`, cmdUUID)
}

// GetCommandHistory returns command history for a device
func (h *AdminHandler) GetCommandHistory(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid path", 400)
		return
	}

	deviceID := parts[3]

	commands, err := h.commandStore.ListByDevice(deviceID, 50)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(commands)
}
