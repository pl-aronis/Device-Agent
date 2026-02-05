import { useState, useEffect } from 'react'
import axios from 'axios'
import DeviceCard from './components/DeviceCard'
import './App.css'

function App() {
  const [devices, setDevices] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)
  const [successMessage, setSuccessMessage] = useState(null)
  const [lockingDeviceId, setLockingDeviceId] = useState(null)

  const API_BASE = 'http://localhost:8080'

  const fetchDevices = async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await axios.get(`${API_BASE}/admin/status`)
      // Handle both array and object with devices property
      const deviceList = Array.isArray(response.data) ? response.data : response.data.devices || []
      setDevices(deviceList)
    } catch (err) {
      const errorMsg = err.response?.data?.message || err.message || 'Unknown error'
      setError(`Failed to fetch devices: ${errorMsg}`)
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const handleLock = async (deviceId) => {
    try {
      setError(null)
      setLockingDeviceId(deviceId) // Start showing progress
      
      const response = await axios.post(`${API_BASE}/admin/set`, null, {
        params: {
          id: deviceId,
          status: 'LOCK',
        },
      })
      
      // Update local state
      setDevices(devices.map(d =>
        d.id === deviceId ? { ...d, status: 'LOCK' } : d
      ))
      
      setSuccessMessage(`🔒 Locking device ${deviceId}... Agent will send recovery key and restart.`)
      
      // Keep the locking indicator visible longer to show the process
      setTimeout(() => {
        setLockingDeviceId(null)
        setSuccessMessage(`✓ Device ${deviceId} locked successfully. Recovery key captured and machine restarting.`)
        setTimeout(() => setSuccessMessage(null), 5000)
      }, 3000)
    } catch (err) {
      setLockingDeviceId(null)
      const errorMsg = err.response?.data?.message || err.message || 'Failed to lock device'
      setError(errorMsg)
      console.error(err)
    }
  }

  const handleUnlock = async (deviceId) => {
    try {
      setError(null)
      const response = await axios.post(`${API_BASE}/admin/set`, null, {
        params: {
          id: deviceId,
          status: 'ACTIVE',
        },
      })
      
      // Update local state with recovery key from response
      setDevices(devices.map(d =>
        d.id === deviceId
          ? {
              ...d,
              status: 'ACTIVE',
              recovery_key: response.data.recovery_key || d.recovery_key,
            }
          : d
      ))
      setSuccessMessage(`Device ${deviceId} unlocked. Recovery key retrieved.`)
      setTimeout(() => setSuccessMessage(null), 3000)
    } catch (err) {
      const errorMsg = err.response?.data?.message || err.message || 'Failed to unlock device'
      setError(errorMsg)
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
        <h1>🔐 Device Agent Control Panel</h1>
        <p className="subtitle">Admin Dashboard - Mobile Device Management</p>
      </header>

      <div className="controls">
        <button className="btn btn-primary" onClick={fetchDevices} disabled={loading}>
          {loading ? '⟳ Refreshing...' : '⟳ Refresh Devices'}
        </button>
      </div>

      {error && (
        <div className="error-banner">
          <span>⚠️ {error}</span>
          <button className="close-banner" onClick={() => setError(null)}>✕</button>
        </div>
      )}

      {successMessage && (
        <div className="success-banner">
          <span>✓ {successMessage}</span>
          <button className="close-banner" onClick={() => setSuccessMessage(null)}>✕</button>
        </div>
      )}

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
                isLocking={lockingDeviceId === device.id}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

export default App
