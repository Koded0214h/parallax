import { SessionsTable, type SessionRow } from '../components/SessionsTable'

interface SessionsViewProps {
  rows: SessionRow[]
  onSelect: (sessionId: string) => void
}

export function SessionsView({ rows, onSelect }: SessionsViewProps) {
  return (
    <main className="view-pane">
      <div className="view-pane-header">
        <h2>Sessions</h2>
        <p className="view-pane-sub">
          Every session observed this run — scenario replays and hand-composed events alike.
          Select one to open it in the Playground.
        </p>
      </div>
      <SessionsTable rows={rows} onSelect={onSelect} />
    </main>
  )
}
