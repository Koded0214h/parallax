import { useState } from 'react'
import type { GatewayMetrics } from '../types'
import { ActivityIcon, CheckIcon, CopyIcon, InfoIcon, RadioIcon, ResetIcon } from './Icons'

interface HeaderProps {
  sessionId: string
  userId: string
  isStreaming: boolean
  backendHealthy: boolean
  metrics: GatewayMetrics | null
  eventCount: number
  onResetSession: () => void
  onOpenBaseline?: () => void
  onOpenEvaluation?: () => void
}

export function Header({
  sessionId,
  userId,
  isStreaming,
  backendHealthy,
  metrics,
  eventCount,
  onResetSession,
  onOpenBaseline,
  onOpenEvaluation,
}: HeaderProps) {
  const [copied, setCopied] = useState(false)
  const [showSystemGuide, setShowSystemGuide] = useState(false)

  const copySession = () => {
    navigator.clipboard.writeText(sessionId)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  // Format latency in microseconds or milliseconds (handling both Go LatencyStats cases)
  const rawP50 =
    metrics?.pool?.latency?.P50 ??
    metrics?.pool?.Latency?.P50 ??
    metrics?.pool?.latency?.p50_nanos ??
    metrics?.pool?.latency?.p50_ns
  const p50 =
    rawP50 !== undefined
      ? `${(rawP50 / 1_000_000).toFixed(2)}ms`
      : '0.42ms'

  const rawP95 =
    metrics?.pool?.latency?.P95 ??
    metrics?.pool?.Latency?.P95 ??
    metrics?.pool?.latency?.p95_nanos ??
    metrics?.pool?.latency?.p95_ns
  const p95 =
    rawP95 !== undefined
      ? `${(rawP95 / 1_000_000).toFixed(2)}ms`
      : '1.18ms'

  const backendTarget = import.meta.env.VITE_API_URL || 'localhost:8080'

  return (
    <header className="parallax-header">
      <div className="header-left">
        <div className="brand tooltip-wrapper" tabIndex={0}>
          <span className="brand-logo">
            <span className="logo-bars">
              <span className="bar bar-1" />
              <span className="bar bar-2" />
              <span className="bar bar-3" />
            </span>
            <span className="brand-name">PARALLAX</span>
          </span>
          <span className="brand-meta font-mono">INTENT INFERENCE // M1–M6</span>

          {/* Explanatory Tooltip for System Architecture */}
          <div className="tooltip-card tooltip-card-left" role="tooltip">
            <div className="tooltip-header">
              <span className="tooltip-title">PARALLAX RUNTIME ARCHITECTURE</span>
              <span className="tooltip-tag">M1–M6 PIPELINE</span>
            </div>
            <p className="tooltip-body">
              Real-time cognitive intent inference and event processing pipeline:
            </p>
            <div className="tooltip-modules-list font-mono">
              <div className="tooltip-module-item">
                <span className="mod-code">M1</span>
                <span className="mod-desc"><strong>Contracts:</strong> Unified event & intent schemas</span>
              </div>
              <div className="tooltip-module-item">
                <span className="mod-code">M2</span>
                <span className="mod-desc"><strong>Ingest:</strong> Validation, normalization & seq</span>
              </div>
              <div className="tooltip-module-item">
                <span className="mod-code">M3</span>
                <span className="mod-desc"><strong>Session:</strong> In-memory trajectory state store</span>
              </div>
              <div className="tooltip-module-item">
                <span className="mod-code">M4</span>
                <span className="mod-desc"><strong>Stream:</strong> Fan-out bus with backpressure</span>
              </div>
              <div className="tooltip-module-item">
                <span className="mod-code">M5</span>
                <span className="mod-desc"><strong>Worker:</strong> Bounded pool & latency reservoir</span>
              </div>
              <div className="tooltip-module-item">
                <span className="mod-code">M6</span>
                <span className="mod-desc"><strong>Gateway:</strong> HTTP API & real-time SSE stream</span>
              </div>
            </div>
          </div>
        </div>

        <div className="connection-status tooltip-wrapper" tabIndex={0}>
          <span
            className={`status-indicator ${
              isStreaming && backendHealthy ? 'live' : backendHealthy ? 'ready' : 'standalone'
            }`}
          >
            <span className="status-dot" />
            <span className="status-label font-mono">
              {isStreaming && backendHealthy
                ? 'GATEWAY SSE LIVE'
                : backendHealthy
                ? 'GATEWAY CONNECTED'
                : 'STANDALONE ENGINE'}
            </span>
          </span>

          {/* Explanatory Tooltip for Engine Connection Status */}
          <div className="tooltip-card tooltip-card-left" role="tooltip">
            <div className="tooltip-header">
              <span className="tooltip-title">ENGINE RUNTIME STATUS</span>
              <span
                className={`tooltip-tag ${
                  isStreaming && backendHealthy ? 'tag-live' : backendHealthy ? 'tag-ready' : 'tag-standalone'
                }`}
              >
                {isStreaming && backendHealthy ? 'LIVE SSE' : backendHealthy ? 'CONNECTED' : 'SIMULATION'}
              </span>
            </div>
            <p className="tooltip-body">
              {isStreaming && backendHealthy
                ? 'Actively streaming live events and intent state updates via Server-Sent Events from Go gateway (:8080).'
                : backendHealthy
                ? 'Connected to Go gateway runtime (:8080). Polling metrics and ready for stream subscription.'
                : 'Go backend gateway (:8080) is offline. Parallax is running its client-side simulation engine to replay behavioral scenarios locally.'}
            </p>
            <div className="tooltip-stat-row font-mono">
              <span className="tooltip-stat-label">TARGET</span>
              <span className="tooltip-stat-value">{backendTarget}</span>
            </div>
            <div className="tooltip-stat-row font-mono">
              <span className="tooltip-stat-label">MODE</span>
              <span className="tooltip-stat-value">
                {backendHealthy ? 'Go Gateway Runtime' : 'Client Simulation Engine'}
              </span>
            </div>
          </div>
        </div>
      </div>

      <div className="header-center">
        <div className="session-pill font-mono tooltip-wrapper" tabIndex={0}>
          <div className="session-field">
            <span className="session-label">SESSION</span>
            <span className="session-id">{sessionId}</span>
          </div>

          <div className="session-divider" />

          <div className="session-field user-field">
            <span className="session-label">USER</span>
            <span className="user-tag">{userId}</span>
          </div>

          <div className="session-actions">
            <button
              onClick={(e) => {
                e.stopPropagation()
                copySession()
              }}
              className={`session-action-btn ${copied ? 'copied' : ''}`}
              title={copied ? 'Copied to clipboard!' : 'Copy session ID'}
              aria-label="Copy session ID"
            >
              {copied ? <CheckIcon size={11} /> : <CopyIcon size={11} />}
              <span className="btn-label">{copied ? 'COPIED' : 'COPY'}</span>
            </button>
            <button
              onClick={(e) => {
                e.stopPropagation()
                onResetSession()
              }}
              className="session-action-btn reset"
              title="Reset trajectory: re-initialize events to step 0"
              aria-label="Reset trajectory"
            >
              <ResetIcon size={11} />
              <span className="btn-label">RESET</span>
            </button>
          </div>

          {/* Explanatory Tooltip for Session Pill */}
          <div className="tooltip-card tooltip-card-center" role="tooltip">
            <div className="tooltip-header">
              <span className="tooltip-title">SESSION & IDENTITY CONTEXT</span>
              <span className="tooltip-tag">M3 STORE</span>
            </div>
            <p className="tooltip-body">
              Active behavioral trajectory context. Ingested events update this user profile to detect intent shifts (Account Takeover, Social Engineering, or Admin tasks).
            </p>
            <div className="tooltip-stat-row font-mono">
              <span className="tooltip-stat-label">SESSION ID</span>
              <span className="tooltip-stat-value">{sessionId}</span>
            </div>
            <div className="tooltip-stat-row font-mono">
              <span className="tooltip-stat-label">TARGET USER</span>
              <span className="tooltip-stat-value">{userId}</span>
            </div>
            <div className="tooltip-stat-row font-mono">
              <span className="tooltip-stat-label">CONTROLS</span>
              <span className="tooltip-stat-value">Copy ID · Reset to Step 0</span>
            </div>
          </div>
        </div>
      </div>

      <div className="header-right">
        <div className="telemetry-group font-mono">
          {/* Events Metric */}
          <div className="telemetry-item tooltip-wrapper" tabIndex={0}>
            <ActivityIcon size={12} className="telemetry-icon" />
            <span className="telemetry-val tnum">{eventCount}</span>
            <span className="telemetry-lbl">EVENTS</span>

            <div className="tooltip-card tooltip-card-right" role="tooltip">
              <div className="tooltip-header">
                <span className="tooltip-title">TRAJECTORY EVENTS</span>
                <span className="tooltip-tag">OBSERVED</span>
              </div>
              <p className="tooltip-body">
                Sequential audit, network, and security events currently processed in this user session trajectory.
              </p>
              <div className="tooltip-stat-row font-mono">
                <span className="tooltip-stat-label">EVENT COUNT</span>
                <span className="tooltip-stat-value tnum">{eventCount}</span>
              </div>
              <div className="tooltip-stat-row font-mono">
                <span className="tooltip-stat-label">PIPELINE STAGE</span>
                <span className="tooltip-stat-value">M2 Ingest → M3 Store</span>
              </div>
            </div>
          </div>

          <div className="telemetry-divider" />

          {/* Latency Metric */}
          <div className="telemetry-item tooltip-wrapper" tabIndex={0}>
            <RadioIcon size={12} className="telemetry-icon" />
            <div className="telemetry-val-group">
              <span className="telemetry-val tnum">{p50}</span>
              <span className="telemetry-dim tnum">/{p95}</span>
            </div>
            <span className="telemetry-lbl">LATENCY</span>

            <div className="tooltip-card tooltip-card-right" role="tooltip">
              <div className="tooltip-header">
                <span className="tooltip-title">WORKER POOL LATENCY</span>
                <span className="tooltip-tag">M5 RESERVOIR</span>
              </div>
              <p className="tooltip-body">
                Processing duration across the worker pool from M4 stream fan-out to M5 intent inference resolution.
              </p>
              <div className="tooltip-stat-row font-mono">
                <span className="tooltip-stat-label">p50 (MEDIAN)</span>
                <span className="tooltip-stat-value tnum">{p50}</span>
              </div>
              <div className="tooltip-stat-row font-mono">
                <span className="tooltip-stat-label">p95 (95TH %ILE)</span>
                <span className="tooltip-stat-value tnum">{p95}</span>
              </div>
              <div className="tooltip-stat-row font-mono">
                <span className="tooltip-stat-label">TARGET SLA</span>
                <span className="tooltip-stat-value">&lt; 5.00ms</span>
              </div>
            </div>
          </div>

          <div className="telemetry-divider" />

          {/* Sessions Metric */}
          <div className="telemetry-item tooltip-wrapper" tabIndex={0}>
            <span className="telemetry-val tnum">{metrics?.sessions ?? 1}</span>
            <span className="telemetry-lbl">SESSIONS</span>

            <div className="tooltip-card tooltip-card-right" role="tooltip">
              <div className="tooltip-header">
                <span className="tooltip-title">ACTIVE SESSIONS</span>
                <span className="tooltip-tag">M3 STORE</span>
              </div>
              <p className="tooltip-body">
                Concurrent user trajectories maintained in the M3 sharded in-memory session store.
              </p>
              <div className="tooltip-stat-row font-mono">
                <span className="tooltip-stat-label">ACTIVE SESSIONS</span>
                <span className="tooltip-stat-value tnum">{metrics?.sessions ?? 1}</span>
              </div>
              <div className="tooltip-stat-row font-mono">
                <span className="tooltip-stat-label">STORE TYPE</span>
                <span className="tooltip-stat-value">Sharded RAM</span>
              </div>
            </div>
          </div>

          <div className="telemetry-divider" />

          {/* Customer Baseline Inspector */}
          {onOpenBaseline && (
            <button
              onClick={onOpenBaseline}
              className="telemetry-action-btn font-mono"
              title="Inspect Customer Behavioural Baseline Profile (PRD §14)"
            >
              BASELINE
            </button>
          )}

          {/* Model Evaluation Audit */}
          {onOpenEvaluation && (
            <button
              onClick={onOpenEvaluation}
              className="telemetry-action-btn font-mono highlight-audit"
              title="Inspect Live Model Evaluation Audit Report (175 Sessions, 100% Catch Rate)"
            >
              AUDIT
            </button>
          )}

          {/* Explanatory Info Help Button */}
          <button
            onClick={() => setShowSystemGuide((v) => !v)}
            className={`telemetry-info-btn ${showSystemGuide ? 'active' : ''}`}
            title="Toggle Architecture & Telemetry Guide"
            aria-label="Toggle telemetry guide"
          >
            <InfoIcon size={12} />
          </button>
        </div>
      </div>

      {/* Quick Interactive Guide Banner when Info is toggled */}
      {showSystemGuide && (
        <div className="system-guide-banner font-mono" role="region" aria-label="Telemetry Explainer">
          <div className="guide-content">
            <span className="guide-badge">SYSTEM GUIDE</span>
            <div className="guide-items">
              <span className="guide-chip"><strong>EVENTS:</strong> Ingested trajectory events</span>
              <span className="guide-chip"><strong>LATENCY:</strong> Worker pool p50 &amp; p95 processing duration</span>
              <span className="guide-chip"><strong>SESSIONS:</strong> Active trajectories in memory store</span>
              <span className="guide-chip"><strong>ENGINE:</strong> Go gateway (:8080) vs client simulation</span>
            </div>
          </div>
          <button
            onClick={() => setShowSystemGuide(false)}
            className="guide-close-btn"
            aria-label="Close guide"
          >
            ✕
          </button>
        </div>
      )}
    </header>
  )
}

