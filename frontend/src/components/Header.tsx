import { useState } from 'react'
import type { GatewayMetrics } from '../types'
import { CheckIcon, CompassIcon, CopyIcon, DatabaseIcon } from './Icons'
import { TechnicalitiesMenu } from './TechnicalitiesMenu'

interface HeaderProps {
  sessionId: string
  userId: string
  isStreaming: boolean
  backendHealthy: boolean
  metrics: GatewayMetrics | null
  eventCount: number
  onOpenBaseline?: () => void
  onOpenEvaluation?: () => void
  onStartTour?: () => void
  onOpenDataStream?: () => void
}

// Logo on the left. Session / User / Technicalities in the middle. Data,
// Tour, Baseline, Audit on the right. Nothing else — the tiny nav underneath
// carries the actual sections.
export function Header({
  sessionId,
  userId,
  isStreaming,
  backendHealthy,
  metrics,
  eventCount,
  onOpenBaseline,
  onOpenEvaluation,
  onStartTour,
  onOpenDataStream,
}: HeaderProps) {
  const [copied, setCopied] = useState(false)

  function copySession() {
    navigator.clipboard.writeText(sessionId).catch(() => {})
    setCopied(true)
    setTimeout(() => setCopied(false), 1200)
  }

  return (
    <header className="parallax-header">
      <div className="header-left">
        <div className="brand">
          <span className="brand-logo">
            <span className="logo-bars">
              <span className="bar bar-1" />
              <span className="bar bar-2" />
              <span className="bar bar-3" />
            </span>
            <span className="brand-name">PARALLAX</span>
          </span>
          <span
            className={`brand-live-dot ${isStreaming && backendHealthy ? 'live' : backendHealthy ? 'ready' : 'standalone'}`}
            title={
              isStreaming && backendHealthy
                ? 'Gateway SSE live'
                : backendHealthy
                  ? 'Gateway connected'
                  : 'Simulation mode — gateway offline'
            }
          />
          <span
            className="brand-sim-chip font-mono"
            title="No real accounts, banks, or people — every event, user and balance on this screen is generated."
          >
            SYNTHETIC DATA
          </span>
        </div>
      </div>

      <div className="header-center">
        <button className="nav-tab header-id-btn" onClick={copySession} title="Copy session ID">
          <span className="header-id-label">Session</span>
          <span className="font-mono">{sessionId}</span>
          {copied ? <CheckIcon size={11} /> : <CopyIcon size={11} />}
        </button>

        <button className="nav-tab header-id-btn" title={userId}>
          <span className="header-id-label">User</span>
          <span className="font-mono">{userId}</span>
        </button>

        <TechnicalitiesMenu metrics={metrics} eventCount={eventCount} />
      </div>

      <div className="header-right">
        {onOpenDataStream && (
          <button
            onClick={onOpenDataStream}
            className="telemetry-action-btn"
            title="Watch the raw event stream moving through Parallax"
          >
            <DatabaseIcon size={11} /> Data
          </button>
        )}
        {onStartTour && (
          <button onClick={onStartTour} className="telemetry-action-btn" title="Replay the guided walkthrough">
            <CompassIcon size={11} /> Tour
          </button>
        )}
        {onOpenBaseline && (
          <button
            onClick={onOpenBaseline}
            className="telemetry-action-btn"
            title="Inspect customer behavioural baseline profile"
          >
            Baseline
          </button>
        )}
        {onOpenEvaluation && (
          <button
            onClick={onOpenEvaluation}
            className="telemetry-action-btn highlight-audit"
            title="Inspect the live model evaluation / audit report"
          >
            Audit
          </button>
        )}
      </div>
    </header>
  )
}
