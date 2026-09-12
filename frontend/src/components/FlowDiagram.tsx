import { InfoTip } from './InfoTip'
import { ChevronRightIcon } from './Icons'

interface FlowStep {
  label: string
  detail: string
  count?: string
  help?: string
}

interface FlowDiagramProps {
  eventCount: number
  evidenceCount: number
  intentLabel: string
  action: string
  live: boolean
}

// Customer -> Transaction -> Events -> Evidence -> Intent Engine -> Decision.
// The point of this diagram: Parallax watches everything AROUND a transfer,
// not just the transfer itself.
export function FlowDiagram({ eventCount, evidenceCount, intentLabel, action, live }: FlowDiagramProps) {
  const steps: FlowStep[] = [
    { label: 'Customer', detail: 'Acts on device' },
    { label: 'Transaction', detail: 'Transfer attempted' },
    {
      label: 'Events',
      detail: 'Session trajectory',
      count: String(eventCount),
      help: 'Every login, device change, beneficiary edit and amount entry around the transfer — not just the transfer itself.',
    },
    {
      label: 'Evidence',
      detail: 'Structured signals',
      count: String(evidenceCount),
      help: 'Raw events converted into measurable facts: "amount is 7x baseline", "recipient added 19s ago".',
    },
    {
      label: 'Intent Engine',
      detail: intentLabel,
      help: 'Competing hypotheses scored against the evidence: legitimate, accidental, social engineering, account takeover.',
    },
    {
      label: 'Decision',
      detail: action,
      help: 'Allow, verify, pause for a probe, or block — chosen from risk and how confident the engine actually is.',
    },
  ]

  return (
    <div className="flow-diagram" aria-label="Parallax processing flow">
      {steps.map((s, i) => (
        <div className="flow-step-group" key={s.label}>
          <div className={`flow-node ${live ? 'flow-node-live' : ''}`}>
            <span className="flow-node-label">
              {s.label}
              {s.help && (
                <InfoTip term="">
                  <span className="tooltip-body">{s.help}</span>
                </InfoTip>
              )}
            </span>
            <span className="flow-node-detail">{s.detail}</span>
            {s.count !== undefined && <span className="flow-node-count font-mono">{s.count}</span>}
          </div>
          {i < steps.length - 1 && (
            <span className={`flow-arrow ${live ? 'flow-arrow-live' : ''}`}>
              <ChevronRightIcon size={14} />
              {live && <span className="flow-packet" style={{ animationDelay: `${i * 0.3}s` }} />}
            </span>
          )}
        </div>
      ))}
    </div>
  )
}
