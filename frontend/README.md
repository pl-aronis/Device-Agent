# Device Agent Frontend

React + Vite admin dashboard for managing Device Agent devices.

## Features

- **Device Management**: View all registered devices in a clean grid layout
- **Lock/Unlock Control**: Lock or unlock devices with a single click
- **Status Display**: Visual status badges showing if devices are ACTIVE or LOCKED
- **Device Information**: Display device location, MAC ID, OS details, and last seen timestamp
- **Recovery Details Display**: Show latest recovery key and recovery protector ID from backend
- **Auto-refresh**: Devices list refreshes every 10 seconds
- **Responsive Design**: Works on desktop, tablet, and mobile devices

## Quick Start

### Prerequisites
- Node.js 16+ and npm

### Installation & Development

```bash
cd frontend
npm install
npm run dev
```

The app will start at `http://localhost:5173` and proxy API calls to `http://localhost:8080`

### Build for Production

```bash
npm run build
npm run preview
```

## Project Structure

```
frontend/
├── src/
│   ├── components/
│   │   ├── DeviceCard.jsx      # Individual device card component
│   │   └── DeviceCard.css      # Device card styling
│   ├── App.jsx                 # Main app component
│   ├── App.css                 # App styling
│   ├── index.css               # Global styling
│   └── main.jsx                # React entry point
├── index.html                  # HTML template
├── vite.config.js             # Vite configuration
└── package.json               # Dependencies
```

## API Integration

The frontend communicates with the backend at `http://localhost:8080`:

### Endpoints Used

- `GET /admin/status` - Fetch all devices
- `GET /admin/set?id=<device_id>&status=<ACTIVE|LOCK>` - Lock or unlock device

## Features Overview

### Device Card
Each device displays:
- **Device ID**: Unique identifier
- **Status Badge**: Shows if device is LOCKED (🔒) or ACTIVE (✓)
- **Location**: Physical/logical device location
- **MAC ID**: Network interface address
- **OS Details**: Operating system information
- **Last Seen**: Last heartbeat timestamp
- **Recovery Protector ID**: BitLocker protector ID reported by the agent
- **Lock/Unlock Button**: Toggle device status
- **Recovery Info Toggle**: Show/hide recovery details

### Recovery Details Display
When recovery data exists on a device:
- A recovery key section appears on the device card
- Shows both the recovery protector ID and the recovery key
- Provides a copy-to-clipboard button for easy sharing

## Development Notes

- The app uses Axios for HTTP requests
- Vite proxy automatically redirects API calls to the backend
- CSS uses modern features like Grid, Flexbox, and Gradients
- Responsive design uses CSS media queries
