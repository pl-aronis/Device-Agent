package apns

import (
	"fmt"
	"log"
	"sync"

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

// NewClientFromBytes creates a new APNs client from certificate bytes
// certData should be PEM-encoded certificate, keyData is the PEM key (or combined PEM)
func NewClientFromBytes(certData, keyData []byte, topic string) (*Client, error) {
	// Combine cert and key into single PEM block if they're separate
	var pemData []byte
	if len(keyData) > 0 {
		pemData = append(certData, '\n')
		pemData = append(pemData, keyData...)
	} else {
		pemData = certData
	}

	cert, err := certificate.FromPemBytes(pemData, "")
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
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

// ClientPool manages APNs clients for multiple tenants
type ClientPool struct {
	mu      sync.RWMutex
	clients map[string]*Client // tenantID -> Client
	loader  ClientLoader
}

// ClientLoader is a function that loads APNs credentials for a tenant
type ClientLoader func(tenantID string) (certData, keyData []byte, topic string, err error)

// NewClientPool creates a new APNs client pool
func NewClientPool(loader ClientLoader) *ClientPool {
	return &ClientPool{
		clients: make(map[string]*Client),
		loader:  loader,
	}
}

// GetClient returns the APNs client for a tenant, creating it if necessary
func (p *ClientPool) GetClient(tenantID string) (*Client, error) {
	p.mu.RLock()
	if client, ok := p.clients[tenantID]; ok {
		p.mu.RUnlock()
		return client, nil
	}
	p.mu.RUnlock()

	// Load client
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if client, ok := p.clients[tenantID]; ok {
		return client, nil
	}

	certData, keyData, topic, err := p.loader(tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to load APNs credentials for tenant %s: %w", tenantID, err)
	}

	if len(certData) == 0 {
		return nil, fmt.Errorf("no APNs certificate configured for tenant %s", tenantID)
	}

	client, err := NewClientFromBytes(certData, keyData, topic)
	if err != nil {
		return nil, err
	}

	p.clients[tenantID] = client
	log.Printf("APNs client initialized for tenant %s", tenantID)

	return client, nil
}

// SendPush sends a push notification for a tenant
func (p *ClientPool) SendPush(tenantID, deviceToken, pushMagic string) error {
	client, err := p.GetClient(tenantID)
	if err != nil {
		return err
	}

	return client.SendPush(deviceToken, pushMagic)
}

// InvalidateClient removes a tenant's client from the pool
func (p *ClientPool) InvalidateClient(tenantID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.clients, tenantID)
}

// InvalidateAll clears all cached clients
func (p *ClientPool) InvalidateAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clients = make(map[string]*Client)
}
