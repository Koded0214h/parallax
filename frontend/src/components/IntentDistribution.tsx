import { useMemo } from 'react'
import type { IntentClass, IntentResponse } from '../types'
import { ActivityIcon } from './Icons'

interface IntentDistributionProps {
  intent: IntentResponse | null
  prevIntent: IntentResponse | null
}

interface ClassConfig {
  key: IntentClass
  label: string
  colorVar: string
  bgVar: string
  borderVar: string
  description: string
}

const INTENT_CLASSES: ClassConfig[] = [
  {
    key: 'legitimate',
    label: 'LEGITIMATE',
    colorVar: 'var(--intent-legit)',
    bgVar: 'var(--intent-legit-bg)',
    borderVar: 'var(--intent-legit-border)',
    description: 'Autonomous genuine customer intent',
  },
  {
    key: 'social_engineering',
    label: 'SOCIAL ENGINEERING',
    colorVar: 'var(--intent-social)',
    bgVar: 'var(--intent-social-bg)',
    borderVar: 'var(--intent-social-border)',
    description: 'Authorized by customer under external coercion / deception',
  },
  {
    key: 'account_takeover',
    label: 'ACCOUNT TAKEOVER',
    colorVar: 'var(--intent-ato)',
    bgVar: 'var(--intent-ato-bg)',
    borderVar: 'var(--intent-ato-border)',
    description: 'Direct hostile unauthorized compromise of credentials',
  },
  {
    key: 'accidental',
    label: 'ACCIDENTAL',
    colorVar: 'var(--intent-accidental)',
    bgVar: 'var(--intent-accidental-bg)',
    borderVar: 'var(--intent-accidental-border)',
    description: 'Human error, slip, typo, or unintentional transfer',
  },
]

const DEFAULT_HYPOTHESES: Record<IntentClass, number> = {
  legitimate: 0.25,
  accidental: 0.25,
  social_engineering: 0.25,
  account_takeover: 0.25,
}

export function IntentDistribution({ intent, prevIntent }: IntentDistributionProps) {
  const hypotheses = intent?.hypotheses ?? DEFAULT_HYPOTHESES

  // Determine dominant hypothesis
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

  const uncertainty = intent?.uncertainty ?? 1.0

  const uncertaintySeverity =
    uncertainty >= 0.65 ? 'high' : uncertainty >= 0.35 ? 'moderate' : 'low'

  return (
    <div className="panel intent-distribution-panel">
      <div className="panel-header">
        <div className="panel-title-group">
          <span className="panel-code font-mono">02</span>
          <h2 className="panel-title">INTENT STATE DISTRIBUTION</h2>
        </div>
        <div className="panel-actions">
          <span className="state-badge font-mono">
            {intent ? `${intent.events_seen} EVT OBSERVED` : 'UNINITIALIZED'}
          </span>
        </div>
      </div>

      <div className="panel-body intent-body">
        {/* Core 4-hypothesis probability meters */}
        <div className="hypotheses-list">
          {INTENT_CLASSES.map((cls) => {
            const rawProb = hypotheses[cls.key] ?? 0
            const percent = Math.round(rawProb * 100)
            const isDominant = dominantKey === cls.key

            // Calculate delta if previous intent exists
            const prevProb = prevIntent?.hypotheses[cls.key]
            const deltaPercent =
              prevProb !== undefined ? Math.round((rawProb - prevProb) * 100) : 0

            return (
              <div
                key={cls.key}
                className={`hypothesis-row ${isDominant ? 'dominant' : ''}`}
                style={
                  {
                    '--h-color': cls.colorVar,
                    '--h-bg': cls.bgVar,
                    '--h-border': cls.borderVar,
                  } as React.CSSProperties
                }
              >
                <div className="h-header">
                  <div className="h-title-group">
                    <span className="h-dot" />
                    <span className="h-name font-mono">{cls.label}</span>
                    {isDominant && <span className="dominant-pill font-mono">DOMINANT</span>}
                  </div>
                  <div className="h-metrics font-mono">
                    {deltaPercent !== 0 && (
                      <span
                        className={`delta-badge tnum ${
                          deltaPercent > 0 ? 'delta-up' : 'delta-down'
                        }`}
                      >
                        {deltaPercent > 0 ? `+${deltaPercent}%` : `${deltaPercent}%`}
                      </span>
                    )}
                    <span className="percent-val tnum">{percent}%</span>
                  </div>
                </div>

                <div className="h-track">
                  <div
                    className="h-fill"
                    style={{
                      width: `${percent}%`,
                      backgroundColor: cls.colorVar,
                    }}
                  />
                </div>

                <div className="h-caption font-mono">{cls.description}</div>
              </div>
            )
          })}
        </div>

        {/* Latent Uncertainty & Entropy Card */}
        <div className="uncertainty-card">
          <div className="uncertainty-header">
            <div className="u-label-group">
              <ActivityIcon size={13} className="u-icon" />
              <span className="u-title font-mono">INTENT UNCERTAINTY (ENTROPY)</span>
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
            <span>0.0 DETERMINISTIC</span>
            <span>0.5 AMBIGUOUS</span>
            <span>1.0 MAX ENTROPY</span>
          </div>

          <p className="uncertainty-explainer">
            {uncertainty >= 0.65
              ? 'High uncertainty: Competing explanations fit the evidence. Parallax avoids blocking prematurely and instead invokes an intent probe to query context.'
              : uncertainty >= 0.35
              ? 'Moderate uncertainty: Evidence points towards a primary explanation, but auxiliary factors warrant verification.'
              : 'Low uncertainty: Strong, converging behavioural evidence. Engine acts decisively with high confidence.'}
          </p>
        </div>
      </div>

      <div className="panel-footer font-mono">
        <span className="footer-label">INFERENCE:</span>
        <span className="footer-val">Latent Bayesian Distribution · P(Intent | E₁...Eₙ)</span>
      </div>
    </div>
  )
}
