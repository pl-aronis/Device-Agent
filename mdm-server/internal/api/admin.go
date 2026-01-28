package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"mdm-server/internal/apns"
	"mdm-server/internal/commands"
	"mdm-server/internal/store"
)

type AdminHandlerType struct {
	Store *store.DeviceStore
	Queue *commands.Queue
	APNs  *apns.Client
}

func AdminHandler(s *store.DeviceStore, q *commands.Queue, a *apns.Client) *AdminHandlerType {
	return &AdminHandlerType{Store: s, Queue: q, APNs: a}
}

// ListDevices returns all enrolled devices
func (h *AdminHandlerType) ListDevices(w http.ResponseWriter, r *http.Request) {
	devices := h.Store.ListDevices()
	json.NewEncoder(w).Encode(devices)
}

// DeviceAction handles POST requests to perform actions on devices
func (h *AdminHandlerType) DeviceAction(w http.ResponseWriter, r *http.Request) {
	// Path: /api/devices/{udid}/{action}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "Invalid path", 400)
		return
	}

	udid := parts[3]
	action := parts[4]

	// Verify device exists
	device, ok := h.Store.GetDevice(udid)
	if !ok {
		http.Error(w, "Device not found", 404)
		return
	}

	var cmdUUID string

	switch action {
	case "lock":
		payload := map[string]interface{}{
			"PIN":     "123456",
			"Message": "This device has been locked by IT.",
		}
		cmdUUID = h.Queue.Enqueue(udid, commands.RequestTypeDeviceLock, payload)

	case "locate":
		cmdUUID = h.Queue.Enqueue(udid, commands.RequestTypeDeviceLocation, nil)

	case "lostmode":
		payload := map[string]interface{}{
			"Message":     "Lost Device. Please return.",
			"PhoneNumber": "555-123-4567",
			"Footnote":    "Property of ACME Corp",
		}
		cmdUUID = h.Queue.Enqueue(udid, commands.RequestTypeEnableLostMode, payload)

	case "disablelostmode":
		cmdUUID = h.Queue.Enqueue(udid, commands.RequestTypeDisableLostMode, nil)

	case "wipe":
		payload := map[string]interface{}{
			"PIN": "123456",
		}
		cmdUUID = h.Queue.Enqueue(udid, commands.RequestTypeEraseDevice, payload)

	default:
		http.Error(w, "Unknown action", 400)
		return
	}

	// Trigger APNs Push
	if h.APNs != nil {
		go func() {
			err := h.APNs.SendPush(device.PushToken, device.PushMagic)
			if err != nil {
				fmt.Printf("Failed to send push to %s: %v\n", udid, err)
			}
		}()
	}

	w.WriteHeader(200)
	fmt.Fprintf(w, `{"status":"ok", "command_uuid":"%s"}`, cmdUUID)
}
