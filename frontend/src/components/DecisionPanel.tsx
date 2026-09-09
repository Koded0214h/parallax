import { useState } from 'react'
import type { IntentResponse, PolicyAction } from '../types'
import {
  CheckIcon,
  MessageSquareIcon,
  ShieldAlertIcon,
  ShieldCheckIcon,
  TerminalIcon,
} from './Icons'

interface DecisionPanelProps {
  intent: IntentResponse | null
  onAnswerProbe?: (response: string) => void
}

function getActionMeta(action?: PolicyAction) {
  switch (action) {
    case 'ALLOW':
      return {
        label: 'ALLOW',
        statusDesc: 'Transaction permitted without friction',
        colorClass: 'action-allow',
        icon: <ShieldCheckIcon size={18} />,
        tag: 'PERMITTED',
      }
    case 'VERIFY':
      return {
        label: 'VERIFY',
        statusDesc: 'Step-up authentication required',
        colorClass: 'action-verify',
        icon: <ShieldAlertIcon size={18} />,
        tag: 'CHALLENGE',
      }
    case 'PROBE':
      return {
        label: 'INTENT PROBE',
        statusDesc: 'High uncertainty: Context probe dispatched',
        colorClass: 'action-probe',
        icon: <MessageSquareIcon size={18} />,
        tag: 'PROBE ACTIVE',
      }
    case 'BLOCK':
      return {
        label: 'BLOCK',
        statusDesc: 'Immediate authorization denial',
        colorClass: 'action-block',
        icon: <ShieldAlertIcon size={18} />,
        tag: 'DENIED',
      }
    case 'ESCALATE':
      return {
        label: 'ESCALATE / INTERVENE',
        statusDesc: 'Payment paused for fraud specialist review',
        colorClass: 'action-escalate',
        icon: <ShieldAlertIcon size={18} />,
        tag: 'INTERVENTION',
      }
    default:
      return {
        label: 'AWAITING INGESTION',
        statusDesc: 'No policy decision rendered yet',
        colorClass: 'action-pending',
        icon: <TerminalIcon size={18} />,
        tag: 'IDLE',
      }
  }
}

function formatEvidenceTag(e: string): string {
  return e
    .replace(/_/g, ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase())
}

export function DecisionPanel({ intent, onAnswerProbe }: DecisionPanelProps) {
  const [customReply, setCustomReply] = useState('')
  const actionMeta = getActionMeta(intent?.action)

  // Confidence computation
  const confidenceLevel =
    !intent
      ? 'NONE'
      : intent.uncertainty < 0.25
      ? 'HIGH'
      : intent.uncertainty < 0.6
      ? 'MODERATE'
      : 'LOW'

  const hasActiveProbe =
    intent?.action === 'PROBE' ||
    (intent?.probe && intent.probe.status === 'pending')

  const hasAnsweredProbe = intent?.probe && intent.probe.status === 'answered'

  const defaultProbeReply =
    'The bank fraud squad phoned me saying my account was compromised and instructed me to move funds into this safe holding account.'

  return (
    <div className="panel decision-panel">
      <div className="panel-header">
        <div className="panel-title-group">
          <span className="panel-code font-mono">03</span>
          <h2 className="panel-title">DECISION & EVIDENCE ENGINE</h2>
        </div>
        <div className="panel-actions">
          <span className={`confidence-tag font-mono ${confidenceLevel.toLowerCase()}`}>
            CONFIDENCE: {confidenceLevel}
          </span>
        </div>
      </div>

      <div className="panel-body decision-body">
        {/* Authoritative Action Card */}
        <div className={`action-card ${actionMeta.colorClass}`}>
          <div className="action-card-header">
            <span className="action-icon">{actionMeta.icon}</span>
            <div className="action-title-group">
              <span className="action-badge font-mono">{actionMeta.tag}</span>
              <h3 className="action-headline font-mono">{actionMeta.label}</h3>
            </div>
          </div>
          <p className="action-desc">{actionMeta.statusDesc}</p>
        </div>

        {/* Interactive Intent Probe Container */}
        {(hasActiveProbe || hasAnsweredProbe) && (
          <div className="intent-probe-box">
            <div className="probe-header">
              <div className="probe-title-group">
                <MessageSquareIcon size={14} className="probe-icon" />
                <span className="probe-title font-mono">INTENT PROBE DIALOGUE</span>
              </div>
              <span
                className={`probe-status-chip font-mono ${
                  hasAnsweredProbe ? 'answered' : 'active'
                }`}
              >
                {hasAnsweredProbe ? 'PROBE RESOLVED' : 'AWAITING RESPONSE'}
              </span>
            </div>

            <div className="probe-prompt">
              <span className="speaker-tag font-mono">PARALLAX BOT:</span>
              <p className="prompt-text">
                {intent?.probe?.prompt ??
                  'Please confirm: What is the primary purpose of this transfer?'}
              </p>
            </div>

            {hasAnsweredProbe ? (
              <div className="probe-response-card">
                <span className="speaker-tag font-mono">CUSTOMER RESPONSE:</span>
                <p className="response-text font-mono">
                  "{intent?.probe?.response}"
                </p>
                <div className="nlp-badge-group font-mono">
                  <span className="nlp-chip">IMPERSONATION SIGNALS DETECTED</span>
                  <span className="nlp-chip">SAFE ACCOUNT KEYWORDS</span>
                </div>
              </div>
            ) : (
              <div className="probe-interaction-area">
                <div className="probe-quick-actions">
                  <span className="quick-label font-mono">SIMULATE APP SCAM RESPONSE:</span>
                  <button
                    onClick={() => onAnswerProbe?.(defaultProbeReply)}
                    className="quick-action-btn font-mono"
                    title="Inject synthetic response: Bank fraud squad told me to transfer to safe account"
                  >
                    Inject "Bank instructed me to move funds to safe account"
                  </button>
                </div>

                <div className="probe-custom-input">
                  <input
                    type="text"
                    value={customReply}
                    onChange={(e) => setCustomReply(e.target.value)}
                    placeholder="Or type custom customer statement..."
                    className="custom-probe-field font-mono"
                  />
                  <button
                    onClick={() => {
                      if (customReply.trim()) {
                        onAnswerProbe?.(customReply.trim())
                        setCustomReply('')
                      }
                    }}
                    disabled={!customReply.trim()}
                    className="submit-probe-btn font-mono"
                  >
                    SUBMIT
                  </button>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Grounded Human-Readable Synthesis */}
        <div className="explanation-section">
          <div className="section-label font-mono">EVIDENCE-GROUNDED SYNTHESIS</div>
          <div className="explanation-box">
            <p className="explanation-text">
              {intent?.explanation ??
                'Awaiting session trajectory to assemble behavioral evidence matrix.'}
            </p>
          </div>
        </div>

        {/* Observable Structured Evidence */}
        <div className="evidence-section">
          <div className="section-label font-mono">
            OBSERVED EVIDENCE ({intent?.evidence?.length ?? 0})
          </div>
          {(!intent?.evidence || intent.evidence.length === 0) ? (
            <div className="empty-evidence font-mono">No structured signals detected yet.</div>
          ) : (
            <ul className="evidence-list">
              {intent.evidence.map((item, idx) => (
                <li key={idx} className="evidence-item">
                  <span className="evidence-bullet">
                    <CheckIcon size={10} />
                  </span>
                  <div className="evidence-text-group">
                    <span className="evidence-tag font-mono">{formatEvidenceTag(item)}</span>
                    <span className="evidence-raw font-mono">{item}</span>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>

      <div className="panel-footer font-mono">
        <span className="footer-label">POLICY ENGINE:</span>
        <span className="footer-val">Risk × Uncertainty Matrix (prd.md §21.8)</span>
      </div>
    </div>
  )
}
