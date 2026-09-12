import { useEffect, useRef, useState } from 'react'
import { api } from '../api'
import type { ParallaxEvent } from '../types'

interface DataStreamOverlayProps {
  isOpen: boolean
  onClose: () => void
}

const MAX_LINES = 200

// A raw look at what's actually moving through Parallax right now — every
// event, as JSON, as it's ingested. This is infrastructure monitoring, not a
// banking UI: the point is to make the "it's really streaming" claim visible.
export function DataStreamOverlay({ isOpen, onClose }: DataStreamOverlayProps) {
  const [lines, setLines] = useState<{ id: string; raw: ParallaxEvent }[]>([])
  const [paused, setPaused] = useState(false)
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!isOpen) return
    const unsubscribe = api.streamEvents((e) => {
      setLines((prev) => {
        if (paused) return prev
        const next = [...prev, { id: e.event_id || `${e.session_id}_${e.seq}`, raw: e }]
        return next.length > MAX_LINES ? next.slice(next.length - MAX_LINES) : next
      })
    })
    return () => unsubscribe()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen])

  useEffect(() => {
    if (!paused) bottomRef.current?.scrollIntoView({ block: 'end' })
  }, [lines, paused])

  if (!isOpen) return null

  return (
    <div className="data-overlay" role="dialog" aria-modal="true" aria-label="Live data stream">
      <div className="data-overlay-header">
        <div className="data-overlay-title-group">
          <span className="data-overlay-dot" />
          <span className="data-overlay-title">Live data stream</span>
          <span className="data-overlay-sub">
            Raw JSON, as ingested — {lines.length} of last {MAX_LINES} events
          </span>
        </div>
        <div className="data-overlay-actions">
          <button className="control-btn" onClick={() => setPaused((p) => !p)}>
            {paused ? 'Resume' : 'Pause'}
          </button>
          <button className="control-btn" onClick={() => setLines([])}>
            Clear
          </button>
          <button className="control-btn reset" onClick={onClose}>
            Close
          </button>
        </div>
      </div>

      <div className="data-overlay-body font-mono">
        {lines.length === 0 && (
          <p className="data-overlay-empty">Awaiting events… nothing has streamed through yet.</p>
        )}
        {lines.map((l) => (
          <pre key={l.id} className="data-overlay-line">
            {JSON.stringify(l.raw)}
          </pre>
        ))}
        <div ref={bottomRef} />
      </div>
    </div>
  )
}
