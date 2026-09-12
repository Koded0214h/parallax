import type { PolicyAction } from '../types'

export interface SessionRow {
  sessionId: string
  userId: string
  amount?: number
  dominantLabel: string
  dominantKey: string
  confidence: number // 0..1
  action?: PolicyAction
  updatedAt: number
}

interface SessionsTableProps {
  rows: SessionRow[]
  limit?: number
  onSelect?: (sessionId: string) => void
}

const ACTION_CLASS: Record<string, string> = {
  ALLOW: 'action-allow',
  VERIFY: 'action-verify',
  PROBE: 'action-probe',
  BLOCK: 'action-block',
  ESCALATE: 'action-escalate',
}

function formatAmount(amount?: number): string {
  if (amount === undefined || Number.isNaN(amount)) return '—'
  return `₦${amount.toLocaleString('en-NG', { maximumFractionDigits: 0 })}`
}

export function SessionsTable({ rows, limit, onSelect }: SessionsTableProps) {
  const shown = limit ? rows.slice(0, limit) : rows

  if (shown.length === 0) {
    return <p className="sessions-empty">No sessions yet — run a scenario or compose an event.</p>
  }

  return (
    <div className="sessions-table" role="table" aria-label="Recent sessions">
      <div className="sessions-row sessions-head" role="row">
        <span>Session</span>
        <span>Amount</span>
        <span>Dominant intent</span>
        <span>Confidence</span>
        <span>Action</span>
      </div>
      {shown.map((r) => (
        <button
          key={r.sessionId}
          className="sessions-row sessions-row-body"
          role="row"
          onClick={() => onSelect?.(r.sessionId)}
        >
          <span className="font-mono sessions-cell-id">{r.sessionId}</span>
          <span className="font-mono">{formatAmount(r.amount)}</span>
          <span className={`sessions-intent-tag intent-${r.dominantKey}`}>{r.dominantLabel}</span>
          <span className="font-mono">{Math.round(r.confidence * 100)}%</span>
          <span className={`sessions-action-tag ${ACTION_CLASS[r.action || ''] || ''}`}>
            {r.action || '—'}
          </span>
        </button>
      ))}
    </div>
  )
}
