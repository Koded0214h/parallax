import { EventTimeline } from '../components/EventTimeline'
import { InfoTip } from '../components/InfoTip'
import type { ParallaxEvent } from '../types'

interface ActivityViewProps {
  events: ParallaxEvent[]
  onClear: () => void
  onOpenComposer: () => void
}

export function ActivityView({ events, onClear, onOpenComposer }: ActivityViewProps) {
  return (
    <main className="view-pane">
      <div className="view-pane-header view-pane-header-row">
        <div>
          <h2>
            Event activity{' '}
            <InfoTip term="">
              <span className="tooltip-body">
                Every observed step in the session — not just the final transfer. A device
                change followed by a password change followed by a new beneficiary is a very
                different story than the same transfer on its own.
              </span>
            </InfoTip>
          </h2>
          <p className="view-pane-sub">
            The raw trajectory Parallax is reasoning over for the current session.
          </p>
        </div>
        <button className="cta-tour compose-cta" onClick={onOpenComposer}>
          + Compose event
        </button>
      </div>
      <div className="activity-timeline-wrap">
        <EventTimeline events={events} onClear={onClear} />
      </div>
    </main>
  )
}
