import { useEffect, useState } from 'react'
import { api } from '../api'
import type { CustomerBaseline } from '../types'
import { ShieldCheckIcon } from './Icons'

interface BaselineModalProps {
  userId: string
  isOpen: boolean
  onClose: () => void
}

export function BaselineModal({ userId, isOpen, onClose }: BaselineModalProps) {
  const [baseline, setBaseline] = useState<CustomerBaseline | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!isOpen) return

    let mounted = true
    api
      .getBaseline(userId)
      .then((data) => {
        if (mounted) {
          setBaseline(data)
          setLoading(false)
        }
      })
      .catch((err: unknown) => {
        if (mounted) {
          setError(err instanceof Error ? err.message : 'Failed to retrieve baseline')
          setLoading(false)
        }
      })

    return () => {
      mounted = false
    }
  }, [isOpen, userId])

  // Close on Escape key
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape' && isOpen) onClose()
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, onClose])

  if (!isOpen) return null

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal-container baseline-modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div className="modal-title-group">
            <ShieldCheckIcon size={18} className="modal-title-icon" />
            <div>
              <h2 className="modal-title font-mono">CUSTOMER BEHAVIOURAL BASELINE</h2>
              <p className="modal-subtitle font-mono">
                PRD §14 · Trajectory Norms & Habitual Payee Ledger for <span className="highlight-id">{userId}</span>
              </p>
            </div>
          </div>
          <button className="modal-close-btn font-mono" onClick={onClose}>
            ✕
          </button>
        </div>

        <div className="modal-body">
          {loading ? (
            <div className="modal-loading font-mono">Querying customer baseline telemetry...</div>
          ) : error ? (
            <div className="modal-error font-mono">{error}</div>
          ) : baseline ? (
            <div className="baseline-content">
              {/* Top Stats Grid */}
              <div className="baseline-stats-grid">
                <div className="b-stat-card">
                  <span className="b-stat-label font-mono">TYPICAL MEAN</span>
                  <span className="b-stat-val font-mono tnum">
                    £{baseline.typical_amount_mean.toLocaleString()}
                  </span>
                  <span className="b-stat-sub font-mono">
                    Std Dev: ±£{baseline.typical_amount_std_dev.toLocaleString()}
                  </span>
                </div>

                <div className="b-stat-card">
                  <span className="b-stat-label font-mono">EXPECTED RANGE</span>
                  <span className="b-stat-val font-mono tnum">
                    £{baseline.typical_amount_min.toLocaleString()} – £{baseline.typical_amount_max.toLocaleString()}
                  </span>
                  <span className="b-stat-sub font-mono">Historical Max: £{(baseline.max_daily_amount || 250000).toLocaleString()}</span>
                </div>

                <div className="b-stat-card">
                  <span className="b-stat-label font-mono">HABITUAL TRANSFERS</span>
                  <span className="b-stat-val font-mono tnum">{baseline.total_transfers}</span>
                  <span className="b-stat-sub font-mono">Active Window: {String(baseline.typical_start_hour).padStart(2, '0')}:00–{String(baseline.typical_end_hour).padStart(2, '0')}:00</span>
                </div>
              </div>

              {/* Known Payees Section */}
              <div className="baseline-section">
                <h3 className="section-heading font-mono">
                  HABITUAL BENEFICIARIES ({Object.keys(baseline.known_beneficiaries || {}).length})
                </h3>
                <div className="beneficiaries-list">
                  {Object.values(baseline.known_beneficiaries || {}).map((ben) => (
                    <div key={ben.beneficiary_id} className="beneficiary-item">
                      <div className="ben-top">
                        <span className="ben-name font-mono">{ben.name}</span>
                        <span className="ben-count font-mono tnum">{ben.transfer_count} transfers</span>
                      </div>
                      <div className="ben-bottom font-mono">
                        <span>{ben.bank_name} · Acc: {ben.account_number}</span>
                        <span className="ben-total tnum">Total: £{ben.total_sent.toLocaleString()}</span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Known Devices Section */}
              <div className="baseline-section">
                <h3 className="section-heading font-mono">
                  AUTHORIZED RECOGNISED DEVICES ({Object.keys(baseline.known_devices || {}).length})
                </h3>
                <div className="devices-list">
                  {Object.values(baseline.known_devices || {}).map((dev) => (
                    <div key={dev.device_id} className="device-item font-mono">
                      <div className="device-info">
                        <span className="device-name">{dev.device_model}</span>
                        <span className="device-sub">{dev.user_agent || dev.device_id}</span>
                      </div>
                      <div className="device-badge font-mono tnum">
                        {dev.use_count} sessions verified
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          ) : null}
        </div>

        <div className="modal-footer font-mono">
          <span className="footer-note">
            Anomalies flagged when live events diverge &gt;3σ from established behavioral baseline.
          </span>
          <button className="btn-modal-done font-mono" onClick={onClose}>
            CLOSE
          </button>
        </div>
      </div>
    </div>
  )
}
