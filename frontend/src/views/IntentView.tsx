import { DecisionPanel } from '../components/DecisionPanel'
import { InfoTip } from '../components/InfoTip'
import { IntentDistribution } from '../components/IntentDistribution'
import type { IntentResponse } from '../types'

interface IntentViewProps {
  intent: IntentResponse | null
  prevIntent: IntentResponse | null
  onAnswerProbe: (response: string) => void
}

export function IntentView({ intent, prevIntent, onAnswerProbe }: IntentViewProps) {
  return (
    <main className="view-pane">
      <div className="view-pane-header">
        <h2>
          Intent state{' '}
          <InfoTip term="">
            <span className="tooltip-body">
              A distribution over four explanations for the same authenticated activity —
              not a single fraud score. Parallax asks "which of these fits the evidence?"
              instead of "is this fraud?"
            </span>
          </InfoTip>
        </h2>
        <p className="view-pane-sub">
          The live hypothesis distribution and resulting decision for the session currently
          in the Playground.
        </p>
      </div>
      <div className="view-pane-split">
        <IntentDistribution intent={intent} prevIntent={prevIntent} />
        <DecisionPanel intent={intent} onAnswerProbe={onAnswerProbe} />
      </div>
    </main>
  )
}
