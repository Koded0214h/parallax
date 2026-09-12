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
        label: 'Allow Transaction',
        statusDesc: 'Autonomous genuine intent. Transaction permitted without user friction.',
        colorClass: 'action-allow',
        icon: <ShieldCheckIcon size={18} />,
        tag: 'Permitted',
      }
    case 'VERIFY':
      return {
        label: 'Step-Up Verification',
        statusDesc: 'Ambiguous behavioral trajectory. Step-up authentication challenge required.',
        colorClass: 'action-verify',
        icon: <ShieldAlertIcon size={18} />,
        tag: 'Challenge Required',
      }
    case 'PROBE':
      return {
        label: 'Interactive Intent Probe',
        statusDesc: 'High uncertainty detected. Conversational context probe dispatched.',
        colorClass: 'action-probe',
        icon: <MessageSquareIcon size={18} />,
        tag: 'Probe Active',
      }
    case 'BLOCK':
      return {
        label: 'Immediate Block',
        statusDesc: 'Severe credential anomaly detected. Immediate authorization denial enforced.',
        colorClass: 'action-block',
        icon: <ShieldAlertIcon size={18} />,
        tag: 'Blocked',
      }
    case 'ESCALATE':
      return {
        label: 'Escalate to Fraud Operations',
        statusDesc: 'Payment held for specialist intervention and payee corroboration.',
        colorClass: 'action-escalate',
        icon: <ShieldAlertIcon size={18} />,
        tag: 'Intervention',
      }
    default:
      return {
        label: 'Awaiting Ingestion',
        statusDesc: 'No policy decision rendered yet. Observing session stream.',
        colorClass: 'action-pending',
        icon: <TerminalIcon size={18} />,
        tag: 'Idle',
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
      ? 'None'
      : intent.uncertainty < 0.25
      ? 'High'
      : intent.uncertainty < 0.6
      ? 'Moderate'
      : 'Low'

  const hasActiveProbe =
    intent?.action === 'PROBE' ||
    (intent?.probe && intent.probe.status === 'pending')

  const hasAnsweredProbe = intent?.probe && intent.probe.status === 'answered'

  const defaultProbeReply =
    'The bank fraud squad phoned me saying my account was compromised and instructed me to move funds into this safe holding account.'

  return (
    <div className="panel decision-panel" data-tour="tour-decision">
      <div className="panel-header">
        <div className="panel-title-group">
          <h2 className="panel-title">Decision & Policy Engine</h2>
        </div>
        <div className="panel-actions">
          <span className={`confidence-tag font-mono ${confidenceLevel.toLowerCase()}`}>
            Confidence: {confidenceLevel}
          </span>
        </div>
      </div>

      <div className="panel-body decision-body">
        {/* Authoritative Action Card */}
        <div className={`action-card ${actionMeta.colorClass}`}>
          <div className="action-card-header">
            <span className="action-icon">{actionMeta.icon}</span>
            <div className="action-title-group">
              <span className="action-badge">{actionMeta.tag}</span>
              <h3 className="action-headline">{actionMeta.label}</h3>
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
                <span className="probe-title">Intent Probe Dialogue</span>
              </div>
              <span
                className={`probe-status-chip ${
                  hasAnsweredProbe ? 'answered' : 'active'
                }`}
              >
                {hasAnsweredProbe ? 'Resolved' : 'Awaiting Response'}
              </span>
            </div>

            <div className="probe-prompt">
              <span className="speaker-tag">Security Concierge:</span>
              <p className="prompt-text">
                {intent?.probe?.prompt ??
                  'Please confirm: What is the primary purpose of this transfer?'}
              </p>
            </div>

            {hasAnsweredProbe ? (
              <div className="probe-response-card">
                <span className="speaker-tag">Customer Response:</span>
                <p className="response-text">
                  "{intent?.probe?.response}"
                </p>
                <div className="nlp-badge-group">
                  <span className="nlp-chip">Impersonation Signals Detected</span>
                  <span className="nlp-chip">Safe Account Keywords</span>
                </div>
              </div>
            ) : (
              <div className="probe-interaction-area">
                {intent?.probe?.options && intent.probe.options.length > 0 ? (
                  <div className="probe-options-group">
                    <span className="quick-label">Select Contextual Response:</span>
                    <div className="probe-options-grid">
                      {intent.probe.options.map((opt) => (
                        <button
                          key={opt.id}
                          onClick={() => onAnswerProbe?.(opt.text)}
                          className={`probe-option-btn ${opt.category || ''}`}
                          title={`Category: ${opt.category || 'general'}`}
                        >
                          <span className="opt-text">{opt.text}</span>
                          {opt.category && (
                            <span className="opt-category">{opt.category}</span>
                          )}
                        </button>
                      ))}
                    </div>
                  </div>
                ) : (
                  <div className="probe-quick-actions">
                    <span className="quick-label">Simulate Impersonation Scam Response:</span>
                    <button
                      onClick={() => onAnswerProbe?.(defaultProbeReply)}
                      className="quick-action-btn"
                      title="Inject synthetic response: Bank fraud squad told me to transfer to safe account"
                    >
                      Inject "Bank instructed me to move funds to safe account"
                    </button>
                  </div>
                )}

                {intent?.probe?.allow_freeform !== false && (
                  <div className="probe-custom-input">
                    <input
                      type="text"
                      value={customReply}
                      onChange={(e) => setCustomReply(e.target.value)}
                      placeholder="Or type custom customer statement..."
                      className="custom-probe-field"
                    />
                    <button
                      onClick={() => {
                        if (customReply.trim()) {
                          onAnswerProbe?.(customReply.trim())
                          setCustomReply('')
                        }
                      }}
                      disabled={!customReply.trim()}
                      className="submit-probe-btn"
                    >
                      Submit
                    </button>
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        {/* Grounded Human-Readable Synthesis */}
        <div className="explanation-section">
          <div className="section-label">Evidence-Grounded Synthesis</div>
          <div className="explanation-box">
            <p className="explanation-text">
              {intent?.explanation ??
                'Awaiting session trajectory to assemble behavioral evidence matrix.'}
            </p>
          </div>
        </div>

        {/* Observable Structured Evidence */}
        <div className="evidence-section">
          <div className="section-label">
            Observed Signals ({intent?.evidence?.length ?? 0})
          </div>
          {(!intent?.evidence || intent.evidence.length === 0) ? (
            <div className="empty-evidence">No structured signals detected yet.</div>
          ) : (
            <ul className="evidence-list">
              {intent.evidence.map((item, idx) => (
                <li key={idx} className="evidence-item">
                  <span className="evidence-bullet">
                    <CheckIcon size={10} />
                  </span>
                  <div className="evidence-text-group">
                    <span className="evidence-tag">{formatEvidenceTag(item)}</span>
                    <span className="evidence-raw font-mono">{item}</span>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>

      <div className="panel-footer">
        <span className="footer-label">Policy Engine:</span>
        <span className="footer-val">Risk × Ambiguity Guardrail</span>
        <span className="footer-sep">·</span>
        <span className="footer-tag">Active Defense</span>
      </div>
    </div>
  )
}
