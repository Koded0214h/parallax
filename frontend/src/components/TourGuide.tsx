import { useEffect, useRef, useState } from 'react'
import type { ViewId } from './NavBar'

export interface TourStep {
  target: string // data-tour value to spotlight
  view?: ViewId // switch to this view before showing the step
  title: string
  body: string
  // If true, the spotlighted element stays clickable (the backdrop never
  // blocks it) and the tour advances itself once interactionDone flips true —
  // a real "try it" step instead of pure narration.
  interactive?: boolean
}

interface TourGuideProps {
  steps: TourStep[]
  stepIndex: number
  currentView: ViewId
  onChangeView: (v: ViewId) => void
  onNext: () => void
  onBack: () => void
  onSkip: () => void
  // Whether the current interactive step's real-world action has happened.
  interactionDone?: boolean
}

interface Rect {
  top: number
  left: number
  width: number
  height: number
}

const PAD = 8
const CARD_WIDTH = 330

export function TourGuide({
  steps,
  stepIndex,
  currentView,
  onChangeView,
  onNext,
  onBack,
  onSkip,
  interactionDone,
}: TourGuideProps) {
  const [rect, setRect] = useState<Rect | null>(null)
  const step = steps[stepIndex]
  const firedRef = useRef(-1)

  // Interactive steps advance themselves once the real action happens —
  // guarded so it only fires once per step even if the condition stays true.
  useEffect(() => {
    if (!step?.interactive || !interactionDone) return
    if (firedRef.current === stepIndex) return
    firedRef.current = stepIndex
    const t = window.setTimeout(onNext, 700)
    return () => window.clearTimeout(t)
  }, [step, interactionDone, stepIndex, onNext])

  // Switch to the view this step needs, if we're not already there.
  useEffect(() => {
    if (step?.view && step.view !== currentView) {
      onChangeView(step.view)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [stepIndex])

  // Measure the target after the right view has had a chance to paint.
  useEffect(() => {
    let raf1 = 0
    let raf2 = 0

    function measure() {
      const el = step ? document.querySelector(`[data-tour="${step.target}"]`) : null
      if (!el) {
        setRect(null)
        return
      }
      el.scrollIntoView({ block: 'center', behavior: 'auto' })
      const r = el.getBoundingClientRect()
      setRect({ top: r.top, left: r.left, width: r.width, height: r.height })
    }

    // Double rAF: one for the view switch to render, one for layout to settle.
    raf1 = requestAnimationFrame(() => {
      raf2 = requestAnimationFrame(measure)
    })

    window.addEventListener('resize', measure)
    window.addEventListener('scroll', measure, true)
    return () => {
      cancelAnimationFrame(raf1)
      cancelAnimationFrame(raf2)
      window.removeEventListener('resize', measure)
      window.removeEventListener('scroll', measure, true)
    }
  }, [stepIndex, currentView, step])

  if (!step) return null

  const spotlightStyle = rect
    ? {
        top: rect.top - PAD,
        left: rect.left - PAD,
        width: rect.width + PAD * 2,
        height: rect.height + PAD * 2,
      }
    : undefined

  // Position the card below the target if there's room, else above; clamp
  // horizontally to the viewport.
  let cardTop = 0
  let cardLeft = 0
  let placement: 'below' | 'above' | 'center' = 'center'
  if (rect) {
    const spaceBelow = window.innerHeight - (rect.top + rect.height)
    if (spaceBelow > 190) {
      placement = 'below'
      cardTop = rect.top + rect.height + PAD + 10
    } else {
      placement = 'above'
      cardTop = Math.max(70, rect.top - PAD - 10)
    }
    cardLeft = Math.min(
      Math.max(16, rect.left + rect.width / 2 - CARD_WIDTH / 2),
      window.innerWidth - CARD_WIDTH - 16,
    )
  } else {
    cardTop = window.innerHeight / 2 - 90
    cardLeft = window.innerWidth / 2 - CARD_WIDTH / 2
  }

  return (
    <div className="tour-root" role="dialog" aria-modal="true" aria-label="Guided walkthrough">
      <div className="tour-backdrop" />
      {spotlightStyle && (
        <div
          className={`tour-spotlight ${step.interactive ? 'tour-spotlight-interactive' : ''}`}
          style={spotlightStyle}
        />
      )}

      <div
        className={`tour-card tour-card-${placement}`}
        style={{ top: cardTop, left: cardLeft, width: CARD_WIDTH }}
      >
        <div className="tour-card-progress">
          Step {stepIndex + 1} of {steps.length}
        </div>
        <h4 className="tour-card-title">{step.title}</h4>
        <p className="tour-card-body">{step.body}</p>

        {step.interactive && (
          <div className={`tour-waiting ${interactionDone ? 'tour-waiting-done' : ''}`}>
            {interactionDone ? '✓ Nice — continuing…' : '● Waiting for you to try it…'}
          </div>
        )}

        <div className="tour-card-actions">
          <button className="tour-skip-btn" onClick={onSkip}>
            Skip tour
          </button>
          <div className="tour-nav-btns">
            {stepIndex > 0 && (
              <button className="tour-back-btn" onClick={onBack}>
                Back
              </button>
            )}
            <button className="tour-next-btn" onClick={onNext}>
              {stepIndex === steps.length - 1 ? 'Done' : 'Next'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
