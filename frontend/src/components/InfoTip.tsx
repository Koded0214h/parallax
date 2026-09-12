import { useState, type ReactNode } from 'react'
import { InfoIcon } from './Icons'

interface InfoTipProps {
  term: string
  children: ReactNode
}

// A small contextual "mini tutorial" affordance: hover or click a concept
// (Intent, Evidence, Uncertainty, ...) to get a one-paragraph explanation
// in place, instead of a separate help system.
export function InfoTip({ term, children }: InfoTipProps) {
  const [open, setOpen] = useState(false)

  return (
    <span
      className="info-tip tooltip-wrapper"
      tabIndex={0}
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
      onClick={(e) => {
        e.stopPropagation()
        setOpen((o) => !o)
      }}
      onBlur={() => setOpen(false)}
    >
      <span className="info-tip-term">{term}</span>
      <InfoIcon size={11} className="info-tip-icon" />
      {open && (
        <span className="tooltip-card info-tip-card" role="tooltip">
          {children}
        </span>
      )}
    </span>
  )
}
