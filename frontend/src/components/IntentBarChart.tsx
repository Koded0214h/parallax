import type { IntentClass } from '../types'

interface IntentBarChartProps {
  // fractions 0..1 per class
  values: Record<IntentClass, number>
  height?: number
}

const ORDER: { key: IntentClass; label: string; color: string }[] = [
  { key: 'legitimate', label: 'Legitimate', color: 'var(--intent-legit)' },
  { key: 'accidental', label: 'Accidental', color: 'var(--intent-accidental)' },
  { key: 'social_engineering', label: 'Social Eng.', color: 'var(--intent-social)' },
  { key: 'account_takeover', label: 'Account Takeover', color: 'var(--intent-ato)' },
]

// Vertical columns — reads left-to-right like the four categories being
// compared side by side, rather than a ranked list.
export function IntentBarChart({ values, height = 140 }: IntentBarChartProps) {
  return (
    <div className="vbar-chart" role="img" aria-label="Intent category comparison" style={{ height }}>
      {ORDER.map((o) => {
        const pct = Math.round((values[o.key] || 0) * 100)
        return (
          <div className="vbar-col" key={o.key}>
            <span className="vbar-value font-mono">{pct}%</span>
            <div className="vbar-track">
              <div className="vbar-fill" style={{ height: `${pct}%`, background: o.color }} />
            </div>
            <span className="vbar-label">{o.label}</span>
          </div>
        )
      })}
    </div>
  )
}
