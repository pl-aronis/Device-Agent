package api

import (
	"io"
	"log"
	"net/http"

	"mdm-server/internal/commands"
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
}

func CheckinHandler(store *store.DeviceStore, queue *commands.Queue) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		switch msg.MessageType {
		case "Authenticate":
			// First step of enrollment
			log.Println("Processing Authenticate...")
			// In a real server, we might validate a challenge password here.
			// For now, accept blindly.
			w.Write([]byte(""))

		case "TokenUpdate":
			// Register device token for APNs
			log.Println("Processing TokenUpdate...")
			store.SaveDevice(msg.UDID, msg.Token, msg.PushMagic)
			w.Write([]byte(""))

		case "CheckOut":
			// Device unenrollment
			log.Printf("Device %s is checking out (unenrolling)", msg.UDID)
			store.RemoveDevice(msg.UDID)
			w.Write([]byte(""))

		default:
			log.Printf("Unknown MessageType: %s", msg.MessageType)
			w.WriteHeader(400)
		}
	}
}

// ConnectHandler handles the /mdm/connect endpoint where devices fetch commands
func ConnectHandler(queue *commands.Queue) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received MDM Connect Request")
		// 1. Read device response (status of previous command)
		//    (Skipping deep parsing for brevity, but would handle "Status" == "Acknowledged")

		// 2. Parse UDID from request (often in headers or body)
		//    Note: In real MDM, device authenticates with mTLS cert, UDID is extracted from cert subject.
		//    For this mock, we assume we might get it from a header we inject during testing or
		//    parse from the plist body (Idle/Status report).

		// Mocking UDID extraction for progress
		bodyBytes, _ := io.ReadAll(r.Body)
		var report struct {
			UDID   string `plist:"UDID"`
			Status string `plist:"Status"`
		}
		plist.Unmarshal(bodyBytes, &report)

		udid := report.UDID
		if udid == "" {
			// If idle, body is empty, we must rely on cert.
			// Fallback mock:
			udid = "mock-device-udid"
		}

		log.Printf("Device %s connected. Status: %s", udid, report.Status)

		// 3. Fetch next command
		cmd := queue.Next(udid)
		if cmd == nil {
			// No commands pending, return empty 200 OK
			w.WriteHeader(200)
			return
		}

		log.Printf("Sending command %s (%s) to device", cmd.RequestType, cmd.CommandUUID)

		// 4. Construct Command Plist
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
}
