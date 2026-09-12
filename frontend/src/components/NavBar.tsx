export type ViewId = 'dashboard' | 'sessions' | 'intent' | 'activity' | 'playground'

const TABS: { id: ViewId; label: string }[] = [
  { id: 'dashboard', label: 'Dashboard' },
  { id: 'sessions', label: 'Sessions' },
  { id: 'intent', label: 'Intent' },
  { id: 'activity', label: 'Activity' },
  { id: 'playground', label: 'Playground' },
]

interface NavBarProps {
  active: ViewId
  onChange: (v: ViewId) => void
}

export function NavBar({ active, onChange }: NavBarProps) {
  return (
    <nav className="top-nav" aria-label="Primary" data-tour="tour-nav">
      {TABS.map((t) => (
        <button
          key={t.id}
          className={`nav-tab ${active === t.id ? 'active' : ''}`}
          onClick={() => onChange(t.id)}
        >
          {t.label}
        </button>
      ))}
    </nav>
  )
}
