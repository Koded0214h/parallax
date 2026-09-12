import { useEffect, useRef, useState } from 'react'
import type { GatewayMetrics } from '../types'
import { ChevronDownIcon, ExternalLinkIcon } from './Icons'

interface TechnicalitiesMenuProps {
  metrics: GatewayMetrics | null
  eventCount: number
}

const DOCS_URL = 'https://github.com/Koded0214h/parallax'

function readLatency(metrics: GatewayMetrics | null, key: 'P50' | 'P95' | 'P99') {
  const l = metrics?.pool?.latency ?? metrics?.pool?.Latency
  if (!l) return undefined
  const nanos =
    (l as Record<string, number | undefined>)[key] ??
    (l as Record<string, number | undefined>)[`${key.toLowerCase()}_ns`] ??
    (l as Record<string, number | undefined>)[`${key.toLowerCase()}_nanos`]
  return nanos
}

function formatMs(nanos?: number): string {
  if (nanos === undefined) return '—'
  return nanos >= 1_000_000
    ? `${(nanos / 1_000_000).toFixed(2)}ms`
    : `${(nanos / 1_000).toFixed(0)}µs`
}

// Rough throughput estimate: processed events over the observed latency
// window's count isn't a real rate, so instead we show it as "processed so
// far" — an honest cumulative counter rather than a fabricated rate.
export function TechnicalitiesMenu({ metrics, eventCount }: TechnicalitiesMenuProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    function onClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onClick)
    return () => document.removeEventListener('mousedown', onClick)
  }, [open])

  const processed = metrics?.pool?.processed ?? metrics?.pool?.Processed
  const workers = metrics?.pool?.workers ?? metrics?.pool?.Workers
  const sessions = metrics?.sessions

  return (
    <div className="tech-menu" ref={ref}>
      <button
        className={`nav-tab tech-menu-trigger ${open ? 'active' : ''}`}
        onClick={() => setOpen((o) => !o)}
      >
        Technicalities
        <ChevronDownIcon size={11} />
      </button>

      {open && (
        <div className="tech-menu-panel" role="menu">
          <div className="tech-row">
            <span>Events (session)</span>
            <span className="font-mono tnum">{eventCount}</span>
          </div>
          <div className="tech-row">
            <span>Events processed (total)</span>
            <span className="font-mono tnum">{processed ?? '—'}</span>
          </div>
          <div className="tech-row">
            <span>Latency p50 / p95</span>
            <span className="font-mono tnum">
              {formatMs(readLatency(metrics, 'P50'))} / {formatMs(readLatency(metrics, 'P95'))}
            </span>
          </div>
          <div className="tech-row">
            <span>Worker pool</span>
            <span className="font-mono tnum">{workers ?? '—'} workers</span>
          </div>
          <div className="tech-row">
            <span>Active sessions</span>
            <span className="font-mono tnum">{sessions ?? '—'}</span>
          </div>

          <a
            className="tech-docs-link"
            href={DOCS_URL}
            target="_blank"
            rel="noreferrer"
          >
            <ExternalLinkIcon size={11} />
            Docs
          </a>
        </div>
      )}
    </div>
  )
}
