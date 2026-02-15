import { useState } from 'react'
import './DeviceCard.css'

function DeviceCard({ device, onLock, onUnlock }) {
  const [showRecoveryKey, setShowRecoveryKey] = useState(false)
  const isLocked = device.status === 'LOCK'
  const hasRecoveryInfo = Boolean(device.recovery_key || device.recovery_protector_id)

  const handleToggleLock = () => {
    if (isLocked) {
      onUnlock(device.id)
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
          {isLocked ? 'LOCKED' : 'ACTIVE'}
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

        <div className="detail-item">
          <span className="label">Recovery Protector ID:</span>
          <span className="value">{device.recovery_protector_id || 'N/A'}</span>
        </div>
      </div>

      <div className="device-actions">
        <button
          className={`btn ${isLocked ? 'btn-unlock' : 'btn-lock'}`}
          onClick={handleToggleLock}
        >
          {isLocked ? 'Unlock Device' : 'Lock Device'}
        </button>
        {hasRecoveryInfo && (
          <button
            className="btn btn-secondary"
            onClick={() => setShowRecoveryKey(prev => !prev)}
          >
            {showRecoveryKey ? 'Hide Recovery Info' : 'Show Recovery Info'}
          </button>
        )}
      </div>

      {showRecoveryKey && hasRecoveryInfo && (
        <div className="recovery-key-section">
          <div className="recovery-header">
            <h3>Recovery Details</h3>
            <button
              className="close-btn"
              onClick={() => setShowRecoveryKey(false)}
            >
              x
            </button>
          </div>
          <div className="recovery-content">
            {device.recovery_protector_id && (
              <div className="recovery-row">
                <p className="recovery-label">Protector ID</p>
                <div className="recovery-key-box">
                  <code>{device.recovery_protector_id}</code>
                </div>
              </div>
            )}
            {device.recovery_key && (
              <div className="recovery-row">
                <p className="recovery-label">Recovery Key</p>
                <div className="recovery-key-box">
                  <code>{device.recovery_key}</code>
                  <button
                    className="copy-btn"
                    onClick={() => {
                      navigator.clipboard.writeText(device.recovery_key)
                      alert('Recovery key copied to clipboard')
                    }}
                  >
                    Copy
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

export default DeviceCard
