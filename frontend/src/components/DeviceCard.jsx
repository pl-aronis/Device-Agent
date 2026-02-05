import { useState } from 'react'
import './DeviceCard.css'

function DeviceCard({ device, onLock, onUnlock, isLocking }) {
  const [showRecoveryKey, setShowRecoveryKey] = useState(false)
  const isLocked = device.status === 'LOCK'
  const hasRecoveryKey = device.recovery_key && device.recovery_key.trim() !== ''

  const handleToggleLock = () => {
    if (isLocked) {
      onUnlock(device.id)
      // Show recovery key modal/display if available
      setTimeout(() => {
        if (hasRecoveryKey) {
          setShowRecoveryKey(true)
        }
      }, 500)
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

        {hasRecoveryKey && !isLocked && (
          <div className="detail-item recovery-status">
            <span className="label">🔑 Recovery Key:</span>
            <span className="value">Stored on device</span>
          </div>
        )}
      </div>

      <div className="device-actions">
        <button
          className={`btn ${isLocked ? 'btn-unlock' : 'btn-lock'} ${isLocking ? 'btn-locking' : ''}`}
          onClick={handleToggleLock}
          disabled={isLocking}
        >
          {isLocking ? (
            <>
              <span className="spinner"></span>
              🔒 Locking & Securing...
            </>
          ) : isLocked ? (
            '🔓 Unlock Device'
          ) : (
            '🔒 Lock Device'
          )}
        </button>

        {hasRecoveryKey && !isLocked && (
          <button
            className="btn btn-secondary"
            onClick={() => setShowRecoveryKey(!showRecoveryKey)}
          >
            {showRecoveryKey ? '🔑 Hide Key' : '🔑 Show Key'}
          </button>
        )}
      </div>

      {showRecoveryKey && hasRecoveryKey && (
        <div className="recovery-key-section">
          <div className="recovery-header">
            <h3>🔑 BitLocker Recovery Key</h3>
            <button
              className="close-btn"
              onClick={() => setShowRecoveryKey(false)}
            >
              ✕
            </button>
          </div>
          <div className="recovery-content">
            <p className="recovery-label">Share this key with the user for device recovery:</p>
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
            <p className="recovery-note">
              ⓘ This key was automatically captured when the device was locked and enforced BitLocker encryption.
            </p>
          </div>
        </div>
      )}
    </div>
  )
}

export default DeviceCard
