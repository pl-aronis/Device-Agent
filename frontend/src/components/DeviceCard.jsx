import { useState } from 'react'
import './DeviceCard.css'

function DeviceCard({ device, onLock, onUnlock }) {
  const [showRecoveryKey, setShowRecoveryKey] = useState(false)
  const isLocked = device.status === 'LOCK'

  const handleToggleLock = () => {
    if (isLocked) {
      onUnlock(device.id)
      // Show recovery key modal/display
      setTimeout(() => setShowRecoveryKey(true), 300)
    } else {
      onLock(device.id)
      setShowRecoveryKey(false)
    }
  }

  return (
    <div className="device-card">
      <div className="device-header">
        <h2 className="device-id">{device.id}</h2>
        <span className={`status-badge ${isLocked ? 'locked' : 'active'}`}>
          {isLocked ? '🔒 LOCKED' : '✓ ACTIVE'}
        </span>
      </div>

      <div className="device-details">
        <div className="detail-item">
          <span className="label">Location:</span>
          <span className="value">{device.location || 'N/A'}</span>
        </div>

        <div className="detail-item">
          <span className="label">MAC ID:</span>
          <span className="value">{device.mac_id || 'N/A'}</span>
        </div>

        <div className="detail-item">
          <span className="label">OS Details:</span>
          <span className="value">{device.os_details || 'N/A'}</span>
        </div>

        <div className="detail-item">
          <span className="label">Last Seen:</span>
          <span className="value">
            {device.last_seen
              ? new Date(device.last_seen).toLocaleString()
              : 'Never'}
          </span>
        </div>
      </div>

      <div className="device-actions">
        <button
          className={`btn ${isLocked ? 'btn-unlock' : 'btn-lock'}`}
          onClick={handleToggleLock}
        >
          {isLocked ? '🔓 Unlock Device' : '🔒 Lock Device'}
        </button>
      </div>

      {showRecoveryKey && device.recovery_key && (
        <div className="recovery-key-section">
          <div className="recovery-header">
            <h3>🔑 Recovery Key</h3>
            <button
              className="close-btn"
              onClick={() => setShowRecoveryKey(false)}
            >
              ✕
            </button>
          </div>
          <div className="recovery-content">
            <p className="recovery-label">Share this key with the user:</p>
            <div className="recovery-key-box">
              <code>{device.recovery_key}</code>
              <button
                className="copy-btn"
                onClick={() => {
                  navigator.clipboard.writeText(device.recovery_key)
                  alert('Recovery key copied to clipboard!')
                }}
              >
                📋 Copy
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export default DeviceCard
