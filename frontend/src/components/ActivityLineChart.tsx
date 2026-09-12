interface Series {
  label: string
  color: string
  points: number[] // most-recent-last
}

interface ActivityLineChartProps {
  series: Series[]
  width?: number
  height?: number
}

// Catmull-Rom -> cubic bezier smoothing so the line reads as "live telemetry"
// rather than a jagged spreadsheet chart.
function smoothPath(pts: { x: number; y: number }[]): string {
  if (pts.length === 0) return ''
  if (pts.length === 1) return `M ${pts[0].x} ${pts[0].y}`
  let d = `M ${pts[0].x} ${pts[0].y}`
  for (let i = 0; i < pts.length - 1; i++) {
    const p0 = pts[i - 1] || pts[i]
    const p1 = pts[i]
    const p2 = pts[i + 1]
    const p3 = pts[i + 2] || p2
    const c1x = p1.x + (p2.x - p0.x) / 6
    const c1y = p1.y + (p2.y - p0.y) / 6
    const c2x = p2.x - (p3.x - p1.x) / 6
    const c2y = p2.y - (p3.y - p1.y) / 6
    d += ` C ${c1x} ${c1y}, ${c2x} ${c2y}, ${p2.x} ${p2.y}`
  }
  return d
}

export function ActivityLineChart({ series, width = 100, height = 42 }: ActivityLineChartProps) {
  const globalMax = Math.max(1, ...series.flatMap((s) => s.points))
  const n = Math.max(1, ...series.map((s) => s.points.length))
  const pad = 3

  return (
    <div className="line-chart-wrap">
      <svg
        viewBox={`0 0 ${width} ${height}`}
        preserveAspectRatio="none"
        className="line-chart-svg"
        role="img"
        aria-label="System activity over time"
      >
        {[0.25, 0.5, 0.75].map((f) => (
          <line
            key={f}
            x1={0}
            x2={width}
            y1={pad + f * (height - 2 * pad)}
            y2={pad + f * (height - 2 * pad)}
            className="line-chart-grid"
          />
        ))}
        {series.map((s) => {
          const pts = s.points.map((v, i) => ({
            x: n === 1 ? 0 : (i / (n - 1)) * width,
            y: height - pad - (v / globalMax) * (height - 2 * pad),
          }))
          const d = smoothPath(pts)
          const last = pts[pts.length - 1]
          return (
            <g key={s.label}>
              <path d={d} fill="none" stroke={s.color} strokeWidth="1.6" className="line-chart-path" />
              {last && <circle cx={last.x} cy={last.y} r="1.8" fill={s.color} />}
            </g>
          )
        })}
      </svg>
      <div className="line-chart-legend">
        {series.map((s) => (
          <span key={s.label} className="line-chart-legend-item">
            <span className="line-chart-swatch" style={{ background: s.color }} />
            {s.label}
          </span>
        ))}
      </div>
    </div>
  )
}
