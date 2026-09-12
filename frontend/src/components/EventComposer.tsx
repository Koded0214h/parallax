import { useEffect, useState, type FormEvent } from 'react'
import type { EventType, ParallaxEvent } from '../types'
import { ActivityIcon, ChevronRightIcon } from './Icons'

interface EventComposerProps {
  isOpen: boolean
  onClose: () => void
  sessionId: string
  userId: string
  backendHealthy: boolean
  busy: boolean
  onSend: (event: ParallaxEvent) => void
}

// The event types a hand-built scenario actually needs (prd.md §10), each
// with the one or two fields that matter for the decision.
const TYPE_OPTIONS: { value: EventType; label: string }[] = [
  { value: 'LOGIN', label: 'Login' },
  { value: 'DEVICE_SEEN', label: 'Device seen' },
  { value: 'DEVICE_CHANGED', label: 'Device changed' },
  { value: 'PASSWORD_CHANGED', label: 'Password changed' },
  { value: 'PIN_CHANGED', label: 'PIN changed' },
  { value: 'BENEFICIARY_CREATED', label: 'Beneficiary added' },
  { value: 'AMOUNT_ENTERED', label: 'Amount entered' },
  { value: 'OTP_REQUESTED', label: 'OTP requested' },
  { value: 'OTP_VERIFIED', label: 'OTP verified' },
  { value: 'TRANSFER_COMPLETED', label: 'Transfer completed' },
  { value: 'TRANSFER_FAILED', label: 'Transfer failed' },
]

function randomID(prefix: string) {
  return `${prefix}_${Math.random().toString(36).slice(2, 8)}`
}

export function EventComposer({
  isOpen,
  onClose,
  sessionId,
  userId,
  backendHealthy,
  busy,
  onSend,
}: EventComposerProps) {
  const [session, setSession] = useState(sessionId)
  const [user, setUser] = useState(userId)
  const [type, setType] = useState<EventType>('AMOUNT_ENTERED')
  const [amount, setAmount] = useState('850000')
  const [deviceKnown, setDeviceKnown] = useState(true)
  const [beneficiaryId, setBeneficiaryId] = useState('ben_new_recipient')

  // The drawer stays mounted (just hidden) so its state survives being
  // toggled — but that means session/user need an explicit refresh each time
  // it opens, otherwise they'd be stuck on whatever session was active the
  // very first time it was opened.
  useEffect(() => {
    if (isOpen) {
      setSession(sessionId)
      setUser(userId)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen])

  if (!isOpen) return null

  function metadataFor(t: EventType): Record<string, unknown> | undefined {
    switch (t) {
      case 'AMOUNT_ENTERED':
        return { amount: Number(amount) || 0, currency: 'NGN' }
      case 'BENEFICIARY_CREATED':
        return { beneficiary_id: beneficiaryId }
      case 'DEVICE_SEEN':
      case 'DEVICE_CHANGED':
        return { device_known: deviceKnown, device_id: deviceKnown ? 'dev_trusted_01' : randomID('dev_new') }
      default:
        return undefined
    }
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    onSend({
      session_id: session.trim() || sessionId,
      user_id: user.trim() || userId,
      type,
      metadata: metadataFor(type),
    })
  }

  function startNewSession() {
    const fresh = randomID('sess_custom')
    setSession(fresh)
    setUser(randomID('usr_custom'))
  }

  return (
    <aside className="composer-drawer" role="dialog" aria-label="Compose a custom event">
      <div className="composer-header">
        <div className="composer-title-group">
          <ActivityIcon size={14} />
          <span className="composer-title">Compose event</span>
        </div>
        <button className="composer-close" onClick={onClose} aria-label="Close">
          <ChevronRightIcon size={14} />
        </button>
      </div>

      <p className="composer-hint">
        Build one event by hand and fire it at the pipeline — no scripted
        scenario, just whatever you set below. Everything here is synthetic;
        nothing touches a real account.
      </p>

      <form className="composer-form" onSubmit={handleSubmit}>
        <label className="composer-field">
          <span>Session ID</span>
          <input
            value={session}
            onChange={(e) => setSession(e.target.value)}
            className="composer-input font-mono"
            spellCheck={false}
          />
        </label>

        <label className="composer-field">
          <span>User ID</span>
          <input
            value={user}
            onChange={(e) => setUser(e.target.value)}
            className="composer-input font-mono"
            spellCheck={false}
          />
        </label>

        <button type="button" className="composer-link-btn" onClick={startNewSession}>
          + start a fresh session/user pair
        </button>

        <label className="composer-field">
          <span>Event type</span>
          <select
            value={type}
            onChange={(e) => setType(e.target.value as EventType)}
            className="composer-input"
          >
            {TYPE_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        </label>

        {type === 'AMOUNT_ENTERED' && (
          <label className="composer-field">
            <span>Amount (₦)</span>
            <input
              type="number"
              min={0}
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              className="composer-input font-mono"
            />
          </label>
        )}

        {type === 'BENEFICIARY_CREATED' && (
          <label className="composer-field">
            <span>Beneficiary ID</span>
            <input
              value={beneficiaryId}
              onChange={(e) => setBeneficiaryId(e.target.value)}
              className="composer-input font-mono"
            />
          </label>
        )}

        {(type === 'DEVICE_SEEN' || type === 'DEVICE_CHANGED') && (
          <label className="composer-field composer-toggle">
            <span>Device is known</span>
            <input
              type="checkbox"
              checked={deviceKnown}
              onChange={(e) => setDeviceKnown(e.target.checked)}
            />
          </label>
        )}

        <button type="submit" className="composer-submit" disabled={busy}>
          {busy ? 'Sending…' : 'Send event'}
        </button>

        <p className="composer-status">
          {backendHealthy
            ? 'Goes to the live gateway — the intent panel updates for real.'
            : 'Gateway offline: this event is only added to the local timeline.'}
        </p>
      </form>
    </aside>
  )
}
