import { useEffect, useRef, useState } from 'react'
import type { EventType, ParallaxEvent } from '../types'
import { ChevronRightIcon, TerminalIcon } from './Icons'

interface EventTimelineProps {
  events: ParallaxEvent[]
  onClear?: () => void
}

function getEventPillClass(type: EventType): string {
  switch (type) {
    case 'LOGIN':
    case 'LOGOUT':
    case 'OTP_VERIFIED':
      return 'pill-auth'
    case 'DEVICE_CHANGED':
    case 'PASSWORD_CHANGED':
    case 'PIN_CHANGED':
      return 'pill-security'
    case 'BENEFICIARY_CREATED':
    case 'BENEFICIARY_MODIFIED':
      return 'pill-payee'
    case 'AMOUNT_ENTERED':
    case 'TRANSFER_STARTED':
      return 'pill-amount'
    case 'TRANSFER_COMPLETED':
      return 'pill-success'
    case 'TRANSFER_FAILED':
      return 'pill-danger'
    case 'INTENT_PROBE_STARTED':
    case 'INTENT_PROBE_RESPONSE':
      return 'pill-probe'
    default:
      return 'pill-default'
  }
}

function formatDelta(currentTs?: number, prevTs?: number): string {
  if (!currentTs || !prevTs) return '+0.0s'
  const deltaSec = (currentTs - prevTs) / 1000
  if (deltaSec < 0.1) return '+0.1s'
  return `+${deltaSec.toFixed(1)}s`
}

export function EventTimeline({ events, onClear }: EventTimelineProps) {
  const [selectedEvent, setSelectedEvent] = useState<ParallaxEvent | null>(null)
  const [autoScroll, setAutoScroll] = useState(true)
  const listEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (autoScroll && listEndRef.current) {
      listEndRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [events, autoScroll])

  return (
    <div className="panel event-timeline-panel">
      <div className="panel-header">
        <div className="panel-title-group">
          <span className="panel-code font-mono">01</span>
          <h2 className="panel-title">LIVE SESSION STREAM</h2>
        </div>
        <div className="panel-actions">
          <span className="event-badge font-mono">
            {events.length} {events.length === 1 ? 'EVENT' : 'EVENTS'}
          </span>
          {onClear && events.length > 0 && (
            <button
              onClick={onClear}
              className="panel-btn font-mono"
              title="Clear timeline events"
            >
              CLEAR
            </button>
          )}
          <button
            onClick={() => setAutoScroll((v) => !v)}
            className={`panel-btn font-mono ${autoScroll ? 'active' : ''}`}
            title="Auto-scroll to latest incoming event"
          >
            AUTO-SCROLL {autoScroll ? 'ON' : 'OFF'}
          </button>
        </div>
      </div>

      <div className="panel-body timeline-body">
        {events.length === 0 ? (
          <div className="empty-state font-mono">
            <TerminalIcon size={20} className="empty-icon" />
            <p>AWAITING INGESTED EVENTS</p>
            <span className="empty-hint">Execute scenario or stream via POST /v1/events</span>
          </div>
        ) : (
          <div className="timeline-list">
            {events.map((evt, idx) => {
              const isSelected = selectedEvent?.event_id === evt.event_id && !!evt.event_id
              const prevEvent = idx > 0 ? events[idx - 1] : undefined
              const delta = formatDelta(evt.timestamp, prevEvent?.timestamp)
              const seqNum = evt.seq ? String(evt.seq).padStart(2, '0') : String(idx + 1).padStart(2, '0')

              return (
                <div
                  key={evt.event_id || `${evt.type}-${idx}`}
                  className={`timeline-item ${isSelected ? 'selected' : ''}`}
                  onClick={() => setSelectedEvent(isSelected ? null : evt)}
                >
                  <div className="item-prefix font-mono">
                    <span className="item-seq">#{seqNum}</span>
                    <span className="item-delta">{delta}</span>
                  </div>

                  <div className="item-content">
                    <div className="item-header">
                      <span className={`event-pill font-mono ${getEventPillClass(evt.type)}`}>
                        {evt.type}
                      </span>
                      {evt.metadata && Object.keys(evt.metadata).length > 0 && (
                        <span className="meta-preview font-mono">
                          {getSummaryMetadata(evt.type, evt.metadata)}
                        </span>
                      )}
                    </div>

                    {isSelected && evt.metadata && (
                      <div className="item-drawer">
                        <div className="drawer-header font-mono">
                          <span>PAYLOAD METADATA</span>
                          <span className="drawer-id">ID: {evt.event_id ?? 'synthetic'}</span>
                        </div>
                        <pre className="json-dump font-mono">
                          {JSON.stringify(evt.metadata, null, 2)}
                        </pre>
                      </div>
                    )}
                  </div>

                  <div className="item-action">
                    <ChevronRightIcon
                      size={12}
                      className={`expand-chevron ${isSelected ? 'rotated' : ''}`}
                    />
                  </div>
                </div>
              )
            })}
            <div ref={listEndRef} />
          </div>
        )}
      </div>

      <div className="panel-footer font-mono">
        <span className="footer-label">STREAM PROTOCOL:</span>
        <span className="footer-val">SSE / v1 / stream / DropOldest</span>
      </div>
    </div>
  )
}

function getSummaryMetadata(_type: EventType, meta: Record<string, unknown>): string {
  if (meta.amount_gbp !== undefined) {
    return `£${Number(meta.amount_gbp).toLocaleString('en-GB', { minimumFractionDigits: 2 })}`
  }
  if (meta.new_device) {
    return String(meta.new_device)
  }
  if (meta.beneficiary_name) {
    return String(meta.beneficiary_name)
  }
  if (meta.device_id) {
    return String(meta.device_id)
  }
  if (meta.auth_method) {
    return String(meta.auth_method)
  }
  if (meta.probe_type) {
    return String(meta.probe_type)
  }
  const firstKey = Object.keys(meta)[0]
  return firstKey ? `${firstKey}: ${String(meta[firstKey])}` : ''
}
