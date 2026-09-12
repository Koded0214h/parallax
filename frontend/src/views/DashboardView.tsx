import { ActivityLineChart } from '../components/ActivityLineChart'
import { FlowDiagram } from '../components/FlowDiagram'
import { InfoTip } from '../components/InfoTip'
import { IntentBarChart } from '../components/IntentBarChart'
import { IntentDonut } from '../components/IntentDonut'
import { SessionsTable, type SessionRow } from '../components/SessionsTable'
import type { IntentClass } from '../types'

interface DashboardViewProps {
  eventCount: number
  evidenceCount: number
  intentLabel: string
  action: string
  live: boolean

  donutCounts: Record<IntentClass, number>
  barValues: Record<IntentClass, number>
  activitySeries: { label: string; color: string; points: number[] }[]

  sessionRows: SessionRow[]
  onSelectSession: (sessionId: string) => void
  onViewAllSessions: () => void
  onStartTour: () => void
}

const CONCEPTS: { term: string; body: string }[] = [
  {
    term: 'Intent',
    body: 'What Parallax thinks the customer was actually trying to do — one of four competing explanations, scored as a distribution rather than a single score.',
  },
  {
    term: 'Evidence',
    body: 'Raw events converted into measurable facts ("amount is 7x baseline", "recipient added 19s ago") — the input to the intent engine.',
  },
  {
    term: 'Behavioural baseline',
    body: "What's normal for this specific customer — typical amount range, devices, timing — not a universal 'normal user'.",
  },
  {
    term: 'Intent uncertainty',
    body: 'How confident the system is in its explanation, kept separate from risk. High risk + low uncertainty acts. Moderate risk + high uncertainty asks first.',
  },
  {
    term: 'Account takeover',
    body: 'A stranger, not the customer, is driving the session — new device, credential changes, unfamiliar beneficiary, moving fast.',
  },
  {
    term: 'Social engineering',
    body: "The real customer is authenticated and in control, but acting on a scammer's instructions — same device, same OTP, wrong outcome.",
  },
]

export function DashboardView({
  eventCount,
  evidenceCount,
  intentLabel,
  action,
  live,
  donutCounts,
  barValues,
  activitySeries,
  sessionRows,
  onSelectSession,
  onViewAllSessions,
  onStartTour,
}: DashboardViewProps) {
  return (
    <main className="view-pane dashboard-view">
      <div className="dash-intro">
        <div className="dash-intro-col">
          <h2>Dashboard</h2>
          <p className="view-pane-sub">
            The operations view: what's happening right now, how it's flowing through the
            system, and what Parallax has decided about it.
          </p>
        </div>
        <div className="dash-intro-col dash-intro-right">
          <h3>About Parallax</h3>
          <p className="view-pane-sub">
            Money enters the system → Parallax observes what happened around it → evidence
            is generated → intent is inferred → uncertainty is evaluated → the system
            decides whether to allow, verify, pause, or block.
          </p>
          <button className="cta-tour" onClick={onStartTour}>
            Take the tour →
          </button>
        </div>
      </div>

      <div className="concept-strip" data-tour="tour-concepts">
        {CONCEPTS.map((c) => (
          <InfoTip term={c.term} key={c.term}>
            <span className="tooltip-body">{c.body}</span>
          </InfoTip>
        ))}
      </div>

      <section className="dash-section" data-tour="tour-flow">
        <FlowDiagram
          eventCount={eventCount}
          evidenceCount={evidenceCount}
          intentLabel={intentLabel}
          action={action}
          live={live}
        />
      </section>

      <section className="dash-grid" data-tour="tour-charts">
        <div className="dash-card">
          <h3>Intent distribution</h3>
          <p className="dash-card-sub">What kinds of intent Parallax is currently seeing</p>
          <IntentDonut counts={donutCounts} />
        </div>

        <div className="dash-card">
          <h3>Intent comparison</h3>
          <p className="dash-card-sub">Current session's hypothesis scores</p>
          <IntentBarChart values={barValues} />
        </div>

        <div className="dash-card dash-card-wide">
          <h3>Activity over time</h3>
          <p className="dash-card-sub">Events processed and risk signal, most recent first</p>
          <ActivityLineChart series={activitySeries} />
        </div>
      </section>

      <section className="dash-section" data-tour="tour-sessions">
        <div className="dash-section-header">
          <h3>Live sessions</h3>
          <button className="view-all-link" onClick={onViewAllSessions}>
            View all →
          </button>
        </div>
        <SessionsTable rows={sessionRows} limit={5} onSelect={onSelectSession} />
      </section>
    </main>
  )
}
