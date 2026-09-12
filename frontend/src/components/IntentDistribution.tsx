import { useMemo } from 'react'
import type { IntentClass, IntentResponse } from '../types'
import { ActivityIcon } from './Icons'
import { IntentBarChart } from './IntentBarChart'

interface IntentDistributionProps {
  intent: IntentResponse | null
  prevIntent: IntentResponse | null
}

const DEFAULT_HYPOTHESES: Record<IntentClass, number> = {
  legitimate: 0.25,
  accidental: 0.25,
  social_engineering: 0.25,
  account_takeover: 0.25,
}

export function IntentDistribution({ intent, prevIntent }: IntentDistributionProps) {
  const hypotheses = (intent?.hypotheses as Record<IntentClass, number>) ?? DEFAULT_HYPOTHESES

  const dominantKey = useMemo(() => {
    let maxK: IntentClass = 'legitimate'
    let maxV = -1
    for (const [k, v] of Object.entries(hypotheses)) {
      if (v > maxV) {
        maxV = v
        maxK = k as IntentClass
      }
    }
    return maxV >= 0.4 ? maxK : null
  }, [hypotheses])

  const dominantDelta = useMemo(() => {
    if (!dominantKey || !prevIntent) return 0
    const prev = prevIntent.hypotheses[dominantKey] ?? 0
    return Math.round((hypotheses[dominantKey] - prev) * 100)
  }, [dominantKey, hypotheses, prevIntent])

  const uncertainty = intent?.uncertainty ?? 1.0
  const uncertaintySeverity =
    uncertainty >= 0.65 ? 'high' : uncertainty >= 0.35 ? 'moderate' : 'low'

  return (
    <div className="panel intent-distribution-panel">
      <div className="panel-header">
        <div className="panel-title-group">
          <h2 className="panel-title">Intent State</h2>
        </div>
        <div className="panel-actions">
          <span className="state-badge font-mono">
            {intent ? `${intent.events_seen} events observed` : 'Uninitialized'}
          </span>
        </div>
      </div>

      <div className="intent-model-row font-mono">
        <span>Model: Bayesian Trajectory Assessment</span>
        <span className="intent-model-sep">·</span>
        <span>Dynamic Prior</span>
      </div>

      <div className="panel-body intent-body">
        <div className="intent-bar-section">
          {dominantKey && (
            <p className="intent-dominant-line">
              <span className="h-dot" style={{ background: 'var(--intent-social)' }} />
              Dominant: <strong>{dominantKey.replace('_', ' ')}</strong>
              {dominantDelta !== 0 && (
                <span className={`delta-badge tnum ${dominantDelta > 0 ? 'delta-up' : 'delta-down'}`}>
                  {dominantDelta > 0 ? `+${dominantDelta}%` : `${dominantDelta}%`}
                </span>
              )}
            </p>
          )}
          <IntentBarChart values={hypotheses} />
        </div>

        {/* Latent Uncertainty & Entropy Card */}
        <div className="uncertainty-card">
          <div className="uncertainty-header">
            <div className="u-label-group">
              <ActivityIcon size={13} className="u-icon" />
              <span className="u-title">Intent Uncertainty (Entropy)</span>
            </div>
            <div className="u-value-group font-mono">
              <span className={`u-status-tag ${uncertaintySeverity}`}>
                {uncertaintySeverity.toUpperCase()}
              </span>
              <span className="u-score tnum">{uncertainty.toFixed(2)}</span>
            </div>
          </div>

          <div className="uncertainty-bar-track">
            <div
              className={`uncertainty-bar-fill ${uncertaintySeverity}`}
              style={{ width: `${Math.min(100, Math.max(4, uncertainty * 100))}%` }}
            />
          </div>

          <div className="uncertainty-ticks font-mono">
            <span>0.0 Deterministic</span>
            <span>0.5 Ambiguous</span>
            <span>1.0 High Entropy</span>
          </div>

          <p className="uncertainty-explainer">
            {uncertainty >= 0.65
              ? "HIGH means several explanations fit the evidence about equally well — the system genuinely doesn't know yet. Parallax avoids blocking on a guess and asks a clarifying question instead."
              : uncertainty >= 0.35
                ? 'MODERATE means one explanation looks most likely, but there is enough ambiguity left that extra verification is worth the friction.'
                : 'LOW means the evidence converges strongly on one explanation. The system acts on it directly — no need to ask.'}
          </p>
        </div>
      </div>
    </div>
  )
}
