# Testing ABM/DEP Flow Without Real ABM

> [!TIP]
> **Mock DEP Server Now Implemented!** See the [Quick Start](#quick-start-using-the-mock-dep-server) section below.

---

## Quick Start: Using the Mock DEP Server

We've implemented a complete mock DEP server that simulates Apple's DEP API. Here's how to use it:

### Option 1: Run Unit Tests with Built-in Mock

```bash
cd mdm-server

# Run all DEP tests
go test ./internal/dep/... -v
```

### Option 2: Use Mock Mode in Your Code

```go
import "mdm-server/internal/dep"

// Create client with built-in mock responses
client := dep.NewClientWithConfig(dep.ClientConfig{
    UseMock: true,
})

// Works exactly like the real API
devices, cursor, err := client.FetchDevices("")
profile, err := client.DefineProfile(dep.Profile{...})
result, err := client.AssignProfile(profileUUID, serials)
```

### Option 3: Run Standalone Mock Server

```bash
# Start the mock DEP server
go run ./cmd/mock-dep -addr :8080

# Then configure your MDM server to use it:
export DEP_MOCK_URL=http://localhost:8080
```

### Option 4: Environment-Based Configuration

```bash
# For development (built-in mock)
export DEP_MOCK=true

# For integration testing (mock server)
export DEP_MOCK_URL=http://localhost:8080

# For production
export DEP_CONSUMER_KEY=your-key
export DEP_CONSUMER_SECRET=your-secret
export DEP_ACCESS_TOKEN=your-token
export DEP_ACCESS_SECRET=your-secret
```

### Files Implemented

| File | Purpose |
|------|---------|
| `internal/dep/client.go` | Configurable DEP client with mock support |
| `internal/dep/mock_server.go` | Standalone mock DEP server |
| `internal/dep/client_test.go` | Comprehensive unit tests |
| `cmd/mock-dep/main.go` | CLI to run mock server |

---

## The Problem

EC2 Mac instances can't be enrolled in Apple Business Manager (ABM) because AWS owns the hardware. This means you can't test the **zero-touch DEP enrollment flow** where a device automatically enrolls during Setup Assistant.

However, you can still thoroughly test your MDM code using these strategies:

---

## Testing Strategy Overview

| What You're Testing | Method | Real Hardware Needed? |
|---------------------|--------|----------------------|
| DEP API client code (fetch devices, assign profiles) | Mock DEP Server | ❌ No |
| MDM enrollment profile installation | Manual enrollment on EC2 Mac | ❌ No |
| MDM command handling | Manual enrollment on EC2 Mac | ❌ No |
| APNS push notifications | Manual enrollment on EC2 Mac | ❌ No |
| **Full DEP zero-touch flow** | Physical Mac + ABM | ✅ Yes |
| **Setup Assistant skip items** | Physical Mac + ABM | ✅ Yes |

---

## Strategy 1: Mock DEP Server (For Development)

Create a mock server that simulates Apple's DEP API responses. This lets you test your DEP client code without connecting to Apple.

### Architecture

```
┌─────────────────┐      ┌──────────────────┐      ┌─────────────────┐
│  Your MDM       │ ---> │  Mock DEP Server │      │  Apple DEP API  │
│  Server         │      │  (localhost)     │      │  (Production)   │
└─────────────────┘      └──────────────────┘      └─────────────────┘
        │                        │                         │
        │    Development         │      Production         │
        └────────────────────────┴─────────────────────────┘
```

### Mock DEP Server Implementation

Create `internal/dep/mock_server.go`:

```go
package dep

import (
    "encoding/json"
    "net/http"
    "sync"
    "time"
)

// MockDEPServer simulates Apple's DEP API for testing
type MockDEPServer struct {
    devices    map[string]Device // serial -> device
    profiles   map[string]Profile
    assignments map[string]string // serial -> profile_uuid
    mu         sync.RWMutex
}

func NewMockDEPServer() *MockDEPServer {
    return &MockDEPServer{
        devices:     make(map[string]Device),
        profiles:    make(map[string]Profile),
        assignments: make(map[string]string),
    }
}

// AddTestDevice adds a device to the mock ABM
func (m *MockDEPServer) AddTestDevice(serial, model, description string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.devices[serial] = Device{
        SerialNumber:       serial,
        Model:              model,
        Description:        description,
        ProfileStatus:      "empty",
        DeviceAssignedDate: time.Now().Format(time.RFC3339),
    }
}

// Handler returns an http.Handler for the mock DEP server
func (m *MockDEPServer) Handler() http.Handler {
    mux := http.NewServeMux()
    
    // GET /session - Create session token
    mux.HandleFunc("/session", func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(map[string]string{
            "auth_session_token": "mock-session-token-12345",
        })
    })
    
    // POST /server/devices - Fetch devices
    mux.HandleFunc("/server/devices", func(w http.ResponseWriter, r *http.Request) {
        m.mu.RLock()
        devices := make([]Device, 0, len(m.devices))
        for _, d := range m.devices {
            devices = append(devices, d)
        }
        m.mu.RUnlock()
        
        json.NewEncoder(w).Encode(SyncResponse{
            Devices:      devices,
            Cursor:       "",
            MoreToFollow: false,
        })
    })
    
    // POST /profile - Define enrollment profile
    mux.HandleFunc("/profile", func(w http.ResponseWriter, r *http.Request) {
        var profile Profile
        json.NewDecoder(r.Body).Decode(&profile)
        
        uuid := generateMockUUID()
        m.mu.Lock()
        m.profiles[uuid] = profile
        m.mu.Unlock()
        
        json.NewEncoder(w).Encode(map[string]string{
            "profile_uuid": uuid,
        })
    })
    
    // PUT /profile/devices - Assign profile to devices
    mux.HandleFunc("/profile/devices", func(w http.ResponseWriter, r *http.Request) {
        var req struct {
            ProfileUUID string   `json:"profile_uuid"`
            Devices     []string `json:"devices"` // serial numbers
        }
        json.NewDecoder(r.Body).Decode(&req)
        
        m.mu.Lock()
        result := make(map[string]string)
        for _, serial := range req.Devices {
            if _, exists := m.devices[serial]; exists {
                m.assignments[serial] = req.ProfileUUID
                device := m.devices[serial]
                device.ProfileStatus = "assigned"
                m.devices[serial] = device
                result[serial] = "SUCCESS"
            } else {
                result[serial] = "NOT_ACCESSIBLE"
            }
        }
        m.mu.Unlock()
        
        json.NewEncoder(w).Encode(map[string]interface{}{
            "devices": result,
        })
    })
    
    // GET /profile - Get profile details
    mux.HandleFunc("/profile/", func(w http.ResponseWriter, r *http.Request) {
        // Implementation for profile retrieval
    })
    
    return mux
}

func generateMockUUID() string {
    return "mock-profile-" + time.Now().Format("20060102150405")
}
```

### Using the Mock Server in Tests

```go
// dep_test.go
package dep

import (
    "net/http/httptest"
    "testing"
)

func TestDEPClient_FetchDevices(t *testing.T) {
    // Create mock server
    mockServer := NewMockDEPServer()
    mockServer.AddTestDevice("C02TEST001", "MacBook Pro", "Test Device 1")
    mockServer.AddTestDevice("C02TEST002", "Mac mini", "Test Device 2")
    
    // Start test server
    ts := httptest.NewServer(mockServer.Handler())
    defer ts.Close()
    
    // Create client pointing to mock server
    client := NewClientWithBaseURL(ts.URL, "key", "secret", "token", "secret")
    
    // Test fetching devices
    devices, cursor, err := client.FetchDevices("")
    if err != nil {
        t.Fatalf("FetchDevices failed: %v", err)
    }
    
    if len(devices) != 2 {
        t.Errorf("Expected 2 devices, got %d", len(devices))
    }
    
    t.Logf("Cursor: %s", cursor)
}

func TestDEPClient_AssignProfile(t *testing.T) {
    mockServer := NewMockDEPServer()
    mockServer.AddTestDevice("C02TEST001", "MacBook Pro", "Test Device")
    
    ts := httptest.NewServer(mockServer.Handler())
    defer ts.Close()
    
    client := NewClientWithBaseURL(ts.URL, "key", "secret", "token", "secret")
    
    // Define profile
    profile := Profile{
        ProfileName:       "Test MDM Profile",
        URL:               "https://mdm.example.com/enroll",
        AwaitDeviceConfig: true,
        IsMDMRemovable:    false,
    }
    
    profileUUID, err := client.DefineProfile(profile)
    if err != nil {
        t.Fatalf("DefineProfile failed: %v", err)
    }
    
    t.Logf("Created profile: %s", profileUUID)
}
```

---

## Strategy 2: Simulated DEP Enrollment (On EC2 Mac)

While you can't get automatic DEP enrollment, you can **simulate the enrollment flow** programmatically.

### The Enrollment Flow Comparison

| Step | Real DEP Flow | Simulated Flow |
|------|---------------|----------------|
| 1 | Device boots, contacts Apple activation servers | N/A (skip) |
| 2 | Apple returns MDM enrollment URL | You provide the URL directly |
| 3 | Device downloads enrollment profile | Download profile via script/Safari |
| 4 | User confirms installation | User/script confirms |
| 5 | Device sends TokenUpdate to MDM | Same ✅ |
| 6 | MDM sends commands | Same ✅ |

### Automated Enrollment Script (for EC2 Mac)

```bash
#!/bin/bash
# simulate_dep_enrollment.sh
# Run this on the EC2 Mac instance

MDM_SERVER="https://your-mdm-server.example.com"
ENROLLMENT_PATH="/mdm/enroll"

echo "=== Simulating DEP Enrollment ==="

# Step 1: Download enrollment profile
echo "[1/4] Downloading enrollment profile..."
curl -s "$MDM_SERVER$ENROLLMENT_PATH" -o /tmp/enrollment.mobileconfig

# Step 2: Verify profile
echo "[2/4] Verifying profile..."
security cms -D -i /tmp/enrollment.mobileconfig | head -20

# Step 3: Install profile (requires user interaction or pre-approval)
echo "[3/4] Installing enrollment profile..."
sudo profiles install -path /tmp/enrollment.mobileconfig

# Step 4: Verify enrollment
echo "[4/4] Verifying MDM enrollment..."
sudo profiles show -type enrollment

echo "=== Enrollment Complete ==="
```

---

## Strategy 3: Physical Mac with Apple Business Manager

For **complete end-to-end testing** including the Setup Assistant flow, you need physical hardware.

### Minimum Requirements

1. **Mac mini M1 or later** (~$599 new, ~$400 refurbished)
2. **Apple Business Manager account** (free with D-U-N-S number)
3. **Your organization must be registered** with Apple

### Setting Up a Test Mac with ABM

#### Step 1: Register for Apple Business Manager

1. Go to [business.apple.com](https://business.apple.com)
2. Click "Sign up now"
3. You'll need:
   - D-U-N-S number (get one free at [dnb.com](https://www.dnb.com/duns.html))
   - Organization verification (can take 24-48 hours)

#### Step 2: Add Your MDM Server to ABM

1. In ABM, go to **Settings** → **Device Management Settings**
2. Click **Add MDM Server**
3. Upload your MDM server's public key
4. Download the DEP token (server token file)
5. Import the token into your MDM server

#### Step 3: Add Test Device to ABM

**Option A: Purchase from Apple/Authorized Reseller**
- Devices purchased directly are automatically added to ABM

**Option B: Add Existing Device via Apple Configurator 2**
1. Download Apple Configurator 2 (free on Mac App Store)
2. Connect your test Mac via USB-C/Thunderbolt
3. Use "Prepare" to add the device to ABM
4. The device will then appear in ABM

> [!WARNING]
> Adding a device to ABM via Apple Configurator **wipes the device**.

#### Step 4: Assign Device to MDM Server

1. In ABM, go to **Devices**
2. Find your device by serial number
3. Assign it to your MDM server

#### Step 5: Test the Flow

1. **Erase the Mac** (System Settings → General → Transfer or Reset)
2. Boot the Mac
3. Connect to WiFi
4. Setup Assistant should show your organization name
5. MDM enrollment happens automatically

---

## Strategy 4: Use Apple's DEP Simulator (Deprecated)

Apple used to provide a DEP simulator, but it's been **discontinued**. The mock server approach (Strategy 1) is the modern replacement.

---

## Recommended Testing Pipeline

```
┌─────────────────────────────────────────────────────────────────────┐
│                        DEVELOPMENT PHASE                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  1. Unit Tests with Mock DEP Server                                │
│     - Test DEP client methods                                       │
│     - Test profile assignment logic                                 │
│     - Test device sync logic                                        │
│                                                                     │
│  2. Integration Tests on EC2 Mac                                   │
│     - Manual enrollment                                             │
│     - MDM command testing                                           │
│     - Profile installation/removal                                  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        STAGING/QA PHASE                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  3. Physical Mac + Test ABM Account                                │
│     - Full DEP enrollment flow                                      │
│     - Setup Assistant customization                                 │
│     - Pre-stage user assignment                                     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        PRODUCTION                                   │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  4. Production ABM + Real Devices                                  │
│     - Monitor enrollment success rate                               │
│     - Track command delivery                                        │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## What Your Code Already Covers

Looking at your `client.go`, you already have a mock implementation:

```go
// Lines 63-73 in client.go
if true {
    return []Device{
        {
            SerialNumber:  "C02XXXXX",
            Model:         "MacBook Pro",
            Description:   "Device Agent Test Device",
            ProfileStatus: "assigned",
        },
    }, "next_cursor_123", nil
}
```

This is a good start! But it should be **configurable** so you can switch between mock and real DEP.

---

## Recommended Code Changes

### 1. Make DEP Client Configurable

```go
type ClientConfig struct {
    BaseURL      string // Override for mock server
    UseMock      bool   // Use built-in mock responses
    ConsumerKey  string
    ConsumerSecret string
    AccessToken  string
    AccessSecret string
}

func NewClientWithConfig(cfg ClientConfig) *Client {
    baseURL := depBaseURL
    if cfg.BaseURL != "" {
        baseURL = cfg.BaseURL
    }
    return &Client{
        baseURL:  baseURL,
        useMock:  cfg.UseMock,
        httpClient: &http.Client{Timeout: 30 * time.Second},
    }
}
```

### 2. Environment-Based Configuration

```go
// In your server startup
func initDEPClient() *dep.Client {
    if os.Getenv("DEP_MOCK") == "true" {
        return dep.NewClientWithConfig(dep.ClientConfig{
            UseMock: true,
        })
    }
    
    if mockURL := os.Getenv("DEP_MOCK_URL"); mockURL != "" {
        return dep.NewClientWithConfig(dep.ClientConfig{
            BaseURL: mockURL,
        })
    }
    
    // Production
    return dep.NewClientWithConfig(dep.ClientConfig{
        ConsumerKey:    os.Getenv("DEP_CONSUMER_KEY"),
        ConsumerSecret: os.Getenv("DEP_CONSUMER_SECRET"),
        AccessToken:    os.Getenv("DEP_ACCESS_TOKEN"),
        AccessSecret:   os.Getenv("DEP_ACCESS_SECRET"),
    })
}
```

---

## Summary

| Testing Level | What to Use | Cost | What You Can Test |
|---------------|-------------|------|-------------------|
| **Unit Tests** | Mock DEP Server | Free | DEP API integration code |
| **Integration** | EC2 Mac + Manual Enrollment | ~$26/day | MDM commands, profiles, APNS |
| **E2E/Staging** | Physical Mac + ABM | ~$400+ once | Full DEP flow, Setup Assistant |

### My Recommendation

1. **Now**: Implement a proper mock DEP server and write unit tests
2. **Use EC2 Mac** for MDM command testing (everything after enrollment works the same)
3. **Later**: Get a cheap Mac mini for full DEP testing when ready for production

The enrollment **source** (DEP vs manual) doesn't change how MDM commands work after enrollment - so 90% of your code can be tested on EC2 Mac!
