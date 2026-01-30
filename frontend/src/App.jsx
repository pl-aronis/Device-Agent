import { useState, useEffect } from 'react'
import axios from 'axios'
import DeviceCard from './components/DeviceCard'
import './App.css'

function App() {
  const [devices, setDevices] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  const API_BASE = 'http://localhost:8080'

  const fetchDevices = async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await axios.get(`${API_BASE}/admin/status`)
      setDevices(response.data || [])
    } catch (err) {
      setError(`Failed to fetch devices: ${err.message}`)
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const handleLock = async (deviceId) => {
    try {
      const response = await axios.get(`${API_BASE}/admin/set`, {
        params: {
          id: deviceId,
          status: 'LOCK',
        },
      })
      // Update local state
      setDevices(devices.map(d =>
        d.id === deviceId ? { ...d, status: 'LOCK' } : d
      ))
    } catch (err) {
      setError(`Failed to lock device: ${err.message}`)
      console.error(err)
    }
  }

  const handleUnlock = async (deviceId) => {
    try {
      const response = await axios.get(`${API_BASE}/admin/set`, {
        params: {
          id: deviceId,
          status: 'ACTIVE',
        },
      })
      // Update local state with recovery key
      setDevices(devices.map(d =>
        d.id === deviceId
          ? {
              ...d,
              status: 'ACTIVE',
              recovery_key: response.data.recovery_key,
            }
          : d
      ))
    } catch (err) {
      setError(`Failed to unlock device: ${err.message}`)
      console.error(err)
    }
  }

  useEffect(() => {
    fetchDevices()
    // Refresh devices every 10 seconds
    const interval = setInterval(fetchDevices, 10000)
    return () => clearInterval(interval)
  }, [])

  return (
    <div className="app-container">
      <header className="app-header">
        <h1>Device Agent Control Panel</h1>
        <p className="subtitle">Admin Dashboard</p>
      </header>

      <div className="controls">
        <button className="btn btn-primary" onClick={fetchDevices}>
          {loading ? 'Refreshing...' : 'Refresh'}
        </button>
      </div>

      {error && <div className="error-banner">{error}</div>}

      <div className="devices-section">
        {loading && devices.length === 0 ? (
          <p className="loading">Loading devices...</p>
        ) : devices.length === 0 ? (
          <p className="no-devices">No devices registered yet</p>
        ) : (
          <div className="devices-grid">
            {devices.map(device => (
              <DeviceCard
                key={device.id}
                device={device}
                onLock={handleLock}
                onUnlock={handleUnlock}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

export default App
