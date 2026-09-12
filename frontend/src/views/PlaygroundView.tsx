import { DecisionPanel } from '../components/DecisionPanel'
import { EventTimeline } from '../components/EventTimeline'
import { IntentDistribution } from '../components/IntentDistribution'
import { ScenarioBar } from '../components/ScenarioBar'
import type { IntentResponse, ParallaxEvent, Scenario } from '../types'

interface PlaygroundViewProps {
  scenarios: Scenario[]
  activeScenario: Scenario
  onSelectScenario: (s: Scenario) => void
  currentStepIndex: number
  isPlaying: boolean
  onTogglePlay: () => void
  onStepForward: () => void
  onReset: () => void
  backendHealthy: boolean
  backendRunning: boolean
  onRunBackend: () => void

  events: ParallaxEvent[]
  intent: IntentResponse | null
  prevIntent: IntentResponse | null
  onAnswerProbe: (response: string) => void

  onOpenComposer: () => void
}

// The step-through / hand-build-an-event experience — this is the "detailed
// session view" the rest of the dashboard links out to.
export function PlaygroundView(props: PlaygroundViewProps) {
  return (
    <>
      <ScenarioBar
        scenarios={props.scenarios}
        activeScenario={props.activeScenario}
        onSelectScenario={props.onSelectScenario}
        currentStepIndex={props.currentStepIndex}
        isPlaying={props.isPlaying}
        onTogglePlay={props.onTogglePlay}
        onStepForward={props.onStepForward}
        onReset={props.onReset}
        backendHealthy={props.backendHealthy}
        backendRunning={props.backendRunning}
        onRunBackend={props.onRunBackend}
        onOpenComposer={props.onOpenComposer}
      />

      <main className="dashboard-grid">
        <section className="dashboard-column col-stream">
          <EventTimeline events={props.events} onClear={props.onReset} />
        </section>
        <section className="dashboard-column col-intent">
          <IntentDistribution intent={props.intent} prevIntent={props.prevIntent} />
        </section>
        <section className="dashboard-column col-decision">
          <DecisionPanel intent={props.intent} onAnswerProbe={props.onAnswerProbe} />
        </section>
      </main>
    </>
  )
}
