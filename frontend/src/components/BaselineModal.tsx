import { useEffect, useState } from 'react'
import { api } from '../api'
import type { CustomerBaseline } from '../types'
import { ShieldCheckIcon } from './Icons'

interface BaselineModalProps {
  userId: string
  isOpen: boolean
  onClose: () => void
}

/** Demo baseline — shown when the live backend is unreachable (e.g. local dev). */
const DEMO_BASELINES: Record<string, CustomerBaseline> = {
  usr_elena_rostova: {
    user_id: 'usr_elena_rostova',
    typical_amount_min: 150,
    typical_amount_max: 12000,
    typical_amount_mean: 3200,
    typical_amount_std_dev: 1850,
    total_transfers: 47,
    typical_start_hour: 9,
    typical_end_hour: 18,
    max_daily_amount: 25000,
    last_password_change: '2025-11-03T14:22:00Z',
    last_pin_change: '2025-08-15T09:10:00Z',
    known_beneficiaries: {
      ben_001: {
        beneficiary_id: 'ben_001',
        name: 'Mikhail Rostov',
        account_number: '****4821',
        bank_name: 'Sberbank',
        transfer_count: 18,
        total_sent: 58400,
      },
      ben_002: {
        beneficiary_id: 'ben_002',
        name: 'Elena Savings Account',
        account_number: '****0093',
        bank_name: 'Barclays UK',
        transfer_count: 22,
        total_sent: 71200,
      },
      ben_003: {
        beneficiary_id: 'ben_003',
        name: 'Rostova Dacha Fund',
        account_number: '****3317',
        bank_name: 'Tinkoff',
        transfer_count: 7,
        total_sent: 14300,
      },
    },
    known_devices: {
      dev_a1: {
        device_id: 'dev_a1',
        device_model: 'iPhone 15 Pro',
        use_count: 89,
        user_agent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_4)',
        first_seen: '2024-03-12T08:00:00Z',
        last_seen: '2025-09-08T19:41:00Z',
      },
      dev_b2: {
        device_id: 'dev_b2',
        device_model: 'MacBook Pro 14"',
        use_count: 34,
        user_agent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5)',
        first_seen: '2024-05-20T11:30:00Z',
        last_seen: '2025-09-07T15:00:00Z',
      },
    },
  },
}

function buildFallback(userId: string): CustomerBaseline {
  return (
    DEMO_BASELINES[userId] ?? {
      user_id: userId,
      typical_amount_min: 100,
      typical_amount_max: 5000,
      typical_amount_mean: 1500,
      typical_amount_std_dev: 800,
      total_transfers: 12,
      typical_start_hour: 8,
      typical_end_hour: 20,
      max_daily_amount: 10000,
      known_beneficiaries: {},
      known_devices: {},
    }
  )
}

export function BaselineModal({ userId, isOpen, onClose }: BaselineModalProps) {
  const [baseline, setBaseline] = useState<CustomerBaseline | null>(null)
  const [loading, setLoading] = useState(false)
  const [isDemo, setIsDemo] = useState(false)

  useEffect(() => {
    if (!isOpen) return

    setLoading(true)
    setIsDemo(false)

    let mounted = true
    api
      .getBaseline(userId)
      .then((data) => {
        if (mounted) {
          setBaseline(data)
          setLoading(false)
        }
      })
      .catch(() => {
        if (mounted) {
          // Backend unreachable — show demo data so the UI is always functional
          setBaseline(buildFallback(userId))
          setIsDemo(true)
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
              <h2 className="modal-title">Customer Behavioral Baseline</h2>
              <p className="modal-subtitle">
                Trajectory Norms &amp; Habitual Payee Ledger · <span className="highlight-id font-mono">{userId}</span>
              </p>
            </div>
          </div>
          <button className="modal-close-btn" onClick={onClose} aria-label="Close modal">
            ✕
          </button>
        </div>

        {isDemo && (
          <div className="demo-banner">
            ⚠ Seeded baseline for {userId} — backend offline or in local simulation mode
          </div>
        )}

        <div className="modal-body">
          {loading ? (
            <div className="modal-loading">Querying customer baseline telemetry...</div>
          ) : baseline ? (
            <div className="baseline-content">
              {!isDemo && (
                <div className="live-badge font-mono">● LIVE RECORD</div>
              )}
              {/* Top Stats Grid */}
              <div className="baseline-stats-grid">
                <div className="b-stat-card">
                  <span className="b-stat-label">Typical Mean</span>
                  <span className="b-stat-val font-mono tnum">
                    ₦{baseline.typical_amount_mean.toLocaleString()}
                  </span>
                  <span className="b-stat-sub font-mono">
                    Std Dev: ±₦{baseline.typical_amount_std_dev.toLocaleString()}
                  </span>
                </div>

                <div className="b-stat-card">
                  <span className="b-stat-label">Expected Range</span>
                  <span className="b-stat-val font-mono tnum">
                    ₦{baseline.typical_amount_min.toLocaleString()} – ₦{baseline.typical_amount_max.toLocaleString()}
                  </span>
                  <span className="b-stat-sub font-mono">Historical Max: ₦{(baseline.max_daily_amount || 250000).toLocaleString()}</span>
                </div>

                <div className="b-stat-card">
                  <span className="b-stat-label">Habitual Transfers</span>
                  <span className="b-stat-val font-mono tnum">{baseline.total_transfers}</span>
                  <span className="b-stat-sub font-mono">Active Window: {String(baseline.typical_start_hour).padStart(2, '0')}:00–{String(baseline.typical_end_hour).padStart(2, '0')}:00</span>
                </div>
              </div>

              {/* Known Payees Section */}
              <div className="baseline-section">
                <h3 className="section-heading">
                  Habitual Beneficiaries ({Object.keys(baseline.known_beneficiaries || {}).length})
                </h3>
                <div className="beneficiaries-list">
                  {Object.values(baseline.known_beneficiaries || {}).map((ben) => (
                    <div key={ben.beneficiary_id} className="beneficiary-item">
                      <div className="ben-top">
                        <span className="ben-name">{ben.name}</span>
                        <span className="ben-count font-mono tnum">{ben.transfer_count} transfers</span>
                      </div>
                      <div className="ben-bottom font-mono">
                        <span>{ben.bank_name} · Acc: {ben.account_number}</span>
                        <span className="ben-total tnum">Total: ₦{ben.total_sent.toLocaleString()}</span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Known Devices Section */}
              <div className="baseline-section">
                <h3 className="section-heading">
                  Recognized Authorized Devices ({Object.keys(baseline.known_devices || {}).length})
                </h3>
                <div className="devices-list">
                  {Object.values(baseline.known_devices || {}).map((dev) => (
                    <div key={dev.device_id} className="device-item">
                      <div className="device-info">
                        <span className="device-name">{dev.device_model}</span>
                        <span className="device-sub font-mono">{dev.user_agent || dev.device_id}</span>
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

        <div className="modal-footer">
          <span className="footer-note">
            Anomalies flagged when live events diverge &gt;3σ from established behavioral baseline.
          </span>
          <button className="btn-modal-done" onClick={onClose}>
            Done
          </button>
        </div>
      </div>
    </div>
  )
}
