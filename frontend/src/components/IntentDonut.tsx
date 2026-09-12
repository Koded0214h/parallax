import type { IntentClass } from '../types'

interface Slice {
  key: IntentClass
  label: string
  value: number
  color: string
}

interface IntentDonutProps {
  counts: Record<IntentClass, number>
}

const ORDER: { key: IntentClass; label: string; color: string }[] = [
  { key: 'legitimate', label: 'Legitimate', color: 'var(--intent-legit)' },
  { key: 'accidental', label: 'Accidental', color: 'var(--intent-accidental)' },
  { key: 'social_engineering', label: 'Social Eng.', color: 'var(--intent-social)' },
  { key: 'account_takeover', label: 'Account Takeover', color: 'var(--intent-ato)' },
]

const R = 34
const CIRC = 2 * Math.PI * R

export function IntentDonut({ counts }: IntentDonutProps) {
  const slices: Slice[] = ORDER.map((o) => ({ ...o, value: counts[o.key] || 0 }))
  const total = slices.reduce((s, x) => s + x.value, 0)

  const { segments } = slices.reduce<{
    offset: number
    segments: Array<Slice & { frac: number; dashArray: string; dashOffset: number }>
  }>(
    (acc, s) => {
      const frac = total > 0 ? s.value / total : 0
      const dash = frac * CIRC
      acc.segments.push({ ...s, frac, dashArray: `${dash} ${CIRC - dash}`, dashOffset: -acc.offset })
      return { offset: acc.offset + dash, segments: acc.segments }
    },
    { offset: 0, segments: [] },
  )

  return (
    <div className="donut-wrap">
      <svg viewBox="0 0 80 80" className="donut-svg" role="img" aria-label="Intent distribution">
        <circle cx="40" cy="40" r={R} fill="none" stroke="var(--border-hairline)" strokeWidth="10" />
        {total === 0 ? null : (
          segments.map((s) =>
            s.value === 0 ? null : (
              <circle
                key={s.key}
                cx="40"
                cy="40"
                r={R}
                fill="none"
                stroke={s.color}
                strokeWidth="10"
                strokeDasharray={s.dashArray}
                strokeDashoffset={s.dashOffset}
                transform="rotate(-90 40 40)"
                strokeLinecap="butt"
              />
            ),
          )
        )}
        <text x="40" y="37" textAnchor="middle" className="donut-center-value">
          {total}
        </text>
        <text x="40" y="49" textAnchor="middle" className="donut-center-label">
          {total === 1 ? 'session' : 'sessions'}
        </text>
      </svg>
      <ul className="donut-legend">
        {segments.map((s) => (
          <li key={s.key}>
            <span className="donut-swatch" style={{ background: s.color }} />
            <span className="donut-legend-label">{s.label}</span>
            <span className="donut-legend-value font-mono">
              {total > 0 ? `${Math.round(s.frac * 100)}%` : '—'}
            </span>
          </li>
        ))}
      </ul>
    </div>
  )
}
