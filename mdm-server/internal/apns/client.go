package apns

import (
	"log"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/certificate"
)

// Client wraps the APNs push client
type Client struct {
	apnsClient *apns2.Client
	topic      string // Bundle ID or Push Topic (from cert subject)
}

// NewClient creates a new APNs client from a .p12 cert file
func NewClient(p12File string, p12Password string, topic string) (*Client, error) {
	cert, err := certificate.FromP12File(p12File, p12Password)
	if err != nil {
		return nil, err
	}

	client := apns2.NewClient(cert).Production()
	return &Client{
		apnsClient: client,
		topic:      topic,
	}, nil
}

// SendPush sends a "Wake Up" notification to the device
func (c *Client) SendPush(deviceToken string, pushMagic string) error {
	notification := &apns2.Notification{
		DeviceToken: deviceToken,
		Topic:       c.topic,
		Payload:     []byte(`{"mdm":"` + pushMagic + `"}`),
	}

	res, err := c.apnsClient.Push(notification)
	if err != nil {
		log.Printf("Error sending push: %v", err)
		return err
	}

	if res.Sent() {
		log.Printf("Push sent: %v %v", res.StatusCode, res.ApnsID)
	} else {
		log.Printf("Push failed: %v %v %v", res.StatusCode, res.ApnsID, res.Reason)
	}

	return nil
}
