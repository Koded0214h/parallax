import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { api, DEMO_SCENARIOS } from './api'
import { BaselineModal } from './components/BaselineModal'
import { DataStreamOverlay } from './components/DataStreamOverlay'
import { EventComposer } from './components/EventComposer'
import { Header } from './components/Header'
import { NavBar, type ViewId } from './components/NavBar'
import type { SessionRow } from './components/SessionsTable'
import { EvaluationModal } from './components/EvaluationModal'
import { TourGuide } from './components/TourGuide'
import { TOUR_STEPS } from './tourSteps'
import { ActivityView } from './views/ActivityView'
import { DashboardView } from './views/DashboardView'
import { IntentView } from './views/IntentView'
import { PlaygroundView } from './views/PlaygroundView'
import { SessionsView } from './views/SessionsView'
import type {
  GatewayMetrics,
  IntentClass,
  IntentResponse,
  LiveSessionRow,
  ParallaxEvent,
  Scenario,
} from './types'

const TOUR_SEEN_KEY = 'parallax_tour_seen_v1'
import './App.css'

const INTENT_LABELS: Record<IntentClass, string> = {
  legitimate: 'Legitimate',
  accidental: 'Accidental',
  social_engineering: 'Social Engineering',
  account_takeover: 'Account Takeover',
}
const EMPTY_INTENT_COUNTS: Record<IntentClass, number> = {
  legitimate: 0,
  accidental: 0,
  social_engineering: 0,
  account_takeover: 0,
}

interface SessionRecord {
  sessionId: string
  userId: string
  events: ParallaxEvent[]
  intent: IntentResponse
  updatedAt: number
}

function dominantOf(intent: IntentResponse) {
  const entries = Object.entries(intent.hypotheses || {}) as [IntentClass, number][]
  if (entries.length === 0) return { key: 'legitimate' as IntentClass, confidence: 0 }
  entries.sort((a, b) => b[1] - a[1])
  const [key, confidence] = entries[0]
  return { key, confidence }
}

// Pull whatever looks like a transfer amount out of a session's events —
// scenario metadata isn't consistent about the key name (amount, amount_ngn, ...).
// Everything in this app is Naira.
function extractAmount(events: ParallaxEvent[]): number | undefined {
  for (let i = events.length - 1; i >= 0; i--) {
    const meta = events[i].metadata
    if (!meta) continue
    for (const [k, v] of Object.entries(meta)) {
      if (/amount/i.test(k) && typeof v === 'number') return v
    }
  }
  return undefined
}

export default function App() {
  const [activeView, setActiveView] = useState<ViewId>('dashboard')

  const [activeScenario, setActiveScenario] = useState<Scenario>(DEMO_SCENARIOS[2]) // Scenario C as centerpiece default
  const [currentStepIndex, setCurrentStepIndex] = useState<number>(0)
  const [isPlaying, setIsPlaying] = useState<boolean>(false)

  const [events, setEvents] = useState<ParallaxEvent[]>([])
  const [intent, setIntent] = useState<IntentResponse | null>(null)
  const [prevIntent, setPrevIntent] = useState<IntentResponse | null>(null)

  const [backendHealthy, setBackendHealthy] = useState<boolean>(false)
  const [isStreaming, setIsStreaming] = useState<boolean>(false)
  const [metrics, setMetrics] = useState<GatewayMetrics | null>(null)
  const [backendRunning, setBackendRunning] = useState<boolean>(false)

  const [showBaselineModal, setShowBaselineModal] = useState<boolean>(false)
  const [showEvaluationModal, setShowEvaluationModal] = useState<boolean>(false)
  const [showComposer, setShowComposer] = useState<boolean>(false)
  const [showDataStream, setShowDataStream] = useState<boolean>(false)
  const [composerBusy, setComposerBusy] = useState<boolean>(false)

  // Guided first-run walkthrough. null = not running.
  const [tourStep, setTourStep] = useState<number | null>(null)

  // Cross-session memory purely for the Dashboard/Sessions overview — every
  // session this client has driven (scenario replays + composed events),
  // independent of the live Playground state below.
  const [sessionLog, setSessionLog] = useState<Record<string, SessionRecord>>({})
  const [history, setHistory] = useState<{ events: number; risk: number }[]>([])

  // The real backend roster — the live feed plus anything any client has
  // driven. This is what actually makes "live sessions" live.
  const [liveRows, setLiveRows] = useState<LiveSessionRow[] | null>(null)

  const timerRef = useRef<number | null>(null)
  const runTokenRef = useRef(0)

  // Current session identification
  const currentSessionId =
    events[0]?.session_id ||
    activeScenario.steps[0]?.event.session_id ||
    'sess_soceng_505'
  const currentUserId =
    events[0]?.user_id ||
    activeScenario.steps[0]?.event.user_id ||
    'usr_elena_rostova'

  // Health check & metrics poll (graceful backoff when backend is offline)
  useEffect(() => {
    let mounted = true

    function checkBackend() {
      api
        .health()
        .then((h) => {
          if (!mounted) return
          if (h && h.status === 'ok') {
            setBackendHealthy(true)
            api
              .metrics()
              .then((m) => {
                if (mounted) setMetrics(m)
              })
              .catch(() => {})
          } else {
            setBackendHealthy(false)
          }
        })
        .catch(() => {
          if (mounted) setBackendHealthy(false)
        })
    }

    checkBackend()
    const interval = setInterval(checkBackend, 8000)

    return () => {
      mounted = false
      clearInterval(interval)
    }
  }, [])

  // Live SSE stream subscription (only connects when backend is healthy)
  useEffect(() => {
    if (!backendHealthy) {
      return
    }

    const unsubscribe = api.streamEvents(
      (incomingEvent) => {
        setEvents((prev) => {
          // Avoid duplicate event_id
          if (
            incomingEvent.event_id &&
            prev.some((e) => e.event_id === incomingEvent.event_id)
          ) {
            return prev
          }
          return [...prev, incomingEvent]
        })
      },
      (connected) => {
        setIsStreaming(connected)
      },
    )

    return () => unsubscribe()
  }, [backendHealthy])

  // Poll the real session roster — this is what makes "live sessions"
  // actually live: the backend's own live feed plus anything any client has
  // driven, not just what this browser tab remembers.
  useEffect(() => {
    if (!backendHealthy) {
      setLiveRows(null)
      return
    }
    let mounted = true
    function poll() {
      api
        .listSessions(50)
        .then((r) => {
          if (mounted) setLiveRows(r.sessions)
        })
        .catch(() => {})
    }
    poll()
    const interval = setInterval(poll, 3000)
    return () => {
      mounted = false
      clearInterval(interval)
    }
  }, [backendHealthy])

  // Record every session as it evolves, so the Dashboard/Sessions views have
  // something to summarize regardless of which view is currently open.
  useEffect(() => {
    if (!intent || events.length === 0) return
    const sid = intent.session_id || currentSessionId
    setSessionLog((prev) => ({
      ...prev,
      [sid]: { sessionId: sid, userId: currentUserId, events, intent, updatedAt: Date.now() },
    }))
    setHistory((prev) => [
      ...prev.slice(-23),
      { events: events.length, risk: 1 - (intent.hypotheses?.legitimate ?? 0) },
    ])
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [intent])

  // Step Forward execution
  const stepForward = useCallback(() => {
    if (currentStepIndex >= activeScenario.steps.length) {
      setIsPlaying(false)
      return
    }

    const step = activeScenario.steps[currentStepIndex]
    const enrichedEvent: ParallaxEvent = {
      ...step.event,
      timestamp: Date.now(),
      seq: currentStepIndex + 1,
      event_id: `evt_${Math.random().toString(36).substring(2, 9)}`,
    }

    // Try posting to Go gateway if online, and retrieve live intent inference
    if (backendHealthy) {
      api
        .sendEvent(enrichedEvent)
        .then(() => api.intent(enrichedEvent.session_id))
        .then((liveIntent) => {
          if (liveIntent) {
            setPrevIntent(intent)
            setIntent(liveIntent)
          }
        })
        .catch(() => {})
    }

    setEvents((prev) => [...prev, enrichedEvent])

    if (step.targetIntent) {
      setPrevIntent(intent)
      setIntent(step.targetIntent)
    }

    setCurrentStepIndex((prev) => prev + 1)
  }, [activeScenario, currentStepIndex, backendHealthy, intent])

  // Play / Pause auto-step loop
  useEffect(() => {
    if (!isPlaying || currentStepIndex >= activeScenario.steps.length) {
      if (timerRef.current) clearTimeout(timerRef.current)
      return
    }

    const currentStep = activeScenario.steps[currentStepIndex]
    const delay = currentStep ? currentStep.delayMs : 800

    timerRef.current = window.setTimeout(() => {
      stepForward()
    }, delay)

    return () => {
      if (timerRef.current) clearTimeout(timerRef.current)
    }
  }, [isPlaying, currentStepIndex, activeScenario, stepForward])

  // Reset session
  const resetSession = useCallback(() => {
    runTokenRef.current++ // invalidate any in-flight "Run on Gateway" reveal
    setBackendRunning(false)
    setIsPlaying(false)
    if (timerRef.current) clearTimeout(timerRef.current)
    setCurrentStepIndex(0)
    setEvents([])
    setPrevIntent(null)
    setIntent(null)

    if (backendHealthy) {
      api.resetState().catch(() => {})
    }
  }, [backendHealthy])

  // Switch scenario
  const handleSelectScenario = useCallback((scenario: Scenario) => {
    runTokenRef.current++ // invalidate any in-flight "Run on Gateway" reveal
    setBackendRunning(false)
    setIsPlaying(false)
    if (timerRef.current) clearTimeout(timerRef.current)
    setActiveScenario(scenario)
    setCurrentStepIndex(0)
    setEvents([])
    setPrevIntent(null)
    setIntent(null)
  }, [])

  // One-click live execution on Go gateway
  const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))

  const runScenarioOnBackend = useCallback(async () => {
    if (!backendHealthy) return

    const idMap: Record<string, string> = {
      'scenario-a': 'legitimate',
      'scenario-b': 'account_takeover',
      'scenario-c': 'social_engineering',
      'scenario-d': 'accidental',
      'scenario-e': 'ambiguous',
    }
    const backendScenarioId = idMap[activeScenario.id] || 'social_engineering'

    // Guards against a second run (or a Reset) landing mid-animation.
    const token = ++runTokenRef.current

    setBackendRunning(true)
    setIsPlaying(false)
    if (timerRef.current) clearTimeout(timerRef.current)
    setPrevIntent(intent)
    setIntent(null) // back to "observing" while the trajectory reveals
    setEvents([])
    setCurrentStepIndex(0)

    try {
      // The backend already ran the whole scenario and decided instantly —
      // that's the point. But dumping the final answer in one frame reads as
      // fake. Replay the same events it saw, one at a time, then reveal the
      // decision — so watching it feels like the pipeline actually working.
      const res = await api.runScenario(backendScenarioId)
      if (runTokenRef.current !== token) return
      if (!res || !res.intent) return

      const generatedEvents = activeScenario.steps.map((s, idx) => ({
        ...s.event,
        session_id: res.session_id,
        timestamp: Date.now() - (activeScenario.steps.length - idx) * 1000,
        seq: idx + 1,
        event_id: `evt_live_${idx + 1}_${Math.random().toString(36).substring(2, 7)}`,
      }))

      for (const e of generatedEvents) {
        await sleep(420)
        if (runTokenRef.current !== token) return
        setEvents((prev) => [...prev, e])
        setCurrentStepIndex((prev) => prev + 1)
      }

      await sleep(350)
      if (runTokenRef.current !== token) return
      setIntent(res.intent)
    } catch {
      // Keep existing manual state if remote call has issues
    } finally {
      if (runTokenRef.current === token) setBackendRunning(false)
    }
  }, [backendHealthy, activeScenario, intent])

  // Fire a single hand-composed event (from the side drawer) at the pipeline
  const sendComposedEvent = useCallback(
    (draft: ParallaxEvent) => {
      const enrichedEvent: ParallaxEvent = {
        ...draft,
        timestamp: Date.now(),
        seq: events.length + 1,
        event_id: `evt_custom_${Math.random().toString(36).substring(2, 9)}`,
      }

      setEvents((prev) => [...prev, enrichedEvent])

      if (backendHealthy) {
        setComposerBusy(true)
        api
          .sendEvent(enrichedEvent)
          .then(() => api.intent(enrichedEvent.session_id))
          .then((liveIntent) => {
            if (liveIntent) {
              setPrevIntent(intent)
              setIntent(liveIntent)
            }
          })
          .catch(() => {})
          .finally(() => setComposerBusy(false))
      }
    },
    [backendHealthy, events.length, intent],
  )

  // Interactive Intent Probe reply
  const handleAnswerProbe = useCallback(
    (responseTxt: string) => {
      // Find or generate response event
      const probeResponseStep = activeScenario.steps.find(
        (s) => s.event.type === 'INTENT_PROBE_RESPONSE',
      )

      const responseEvent: ParallaxEvent = {
        session_id: currentSessionId,
        user_id: currentUserId,
        type: 'INTENT_PROBE_RESPONSE',
        timestamp: Date.now(),
        seq: events.length + 1,
        event_id: `evt_probe_${Math.random().toString(36).substring(2, 9)}`,
        metadata: {
          user_response_text: responseTxt,
          channel: 'in_app_chat_dialogue',
          nlp_signals: ['safe_account_keyword', 'urgent_duress'],
        },
      }

      // If backend is healthy, dispatch live probe response and get recalculated intent
      if (backendHealthy) {
        api
          .respondProbe(currentSessionId, responseTxt)
          .then((liveIntent) => {
            if (liveIntent) {
              setPrevIntent(intent)
              setIntent(liveIntent)
            }
          })
          .catch(() => {})
      }

      setEvents((prev) => [...prev, responseEvent])

      if (probeResponseStep?.targetIntent) {
        setPrevIntent(intent)
        setIntent({
          ...probeResponseStep.targetIntent,
          probe: {
            prompt:
              activeScenario.probePrompt ||
              'Please confirm: What is the purpose of this transfer?',
            response: responseTxt,
            status: 'answered',
          },
        })
      }

      setCurrentStepIndex(activeScenario.steps.length)
      setIsPlaying(false)
    },
    [activeScenario, currentSessionId, currentUserId, events.length, backendHealthy, intent],
  )

  // Open a previously observed session (from Dashboard/Sessions) in the Playground
  const selectSession = useCallback(
    (sessionId: string) => {
      setIsPlaying(false)
      if (timerRef.current) clearTimeout(timerRef.current)

      // Already known locally (a scenario/composer run this tab drove) —
      // open it immediately, no round trip needed.
      const rec = sessionLog[sessionId]
      if (rec) {
        setEvents(rec.events)
        setPrevIntent(intent)
        setIntent(rec.intent)
        setCurrentStepIndex(rec.events.length)
        setActiveView('playground')
        return
      }

      // Otherwise it's a session only the backend knows about (the live
      // feed, or another client) — fetch its real trajectory and decision.
      if (!backendHealthy) return
      Promise.all([api.sessionEvents(sessionId), api.intent(sessionId)])
        .then(([ev, liveIntent]) => {
          setEvents(ev.events)
          setPrevIntent(intent)
          setIntent(liveIntent)
          setCurrentStepIndex(ev.events.length)
          setActiveView('playground')
        })
        .catch(() => {})
    },
    [sessionLog, intent, backendHealthy],
  )

  // Keyboard Shortcuts
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (
        e.target instanceof HTMLInputElement ||
        e.target instanceof HTMLTextAreaElement
      ) {
        return
      }

      if (e.key === '1') {
        handleSelectScenario(DEMO_SCENARIOS[0])
      } else if (e.key === '2') {
        handleSelectScenario(DEMO_SCENARIOS[1])
      } else if (e.key === '3') {
        handleSelectScenario(DEMO_SCENARIOS[2])
      } else if (e.key === '4') {
        handleSelectScenario(DEMO_SCENARIOS[3])
      } else if (e.key === '5' && DEMO_SCENARIOS[4]) {
        handleSelectScenario(DEMO_SCENARIOS[4])
      } else if (e.code === 'Space') {
        e.preventDefault()
        stepForward()
      } else if (e.key.toLowerCase() === 'r') {
        resetSession()
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleSelectScenario, stepForward, resetSession])

  // Auto-start the guided walkthrough the very first time someone opens the
  // app. Let the initial paint settle first so the spotlight measures right.
  useEffect(() => {
    let seen = true
    try {
      seen = localStorage.getItem(TOUR_SEEN_KEY) === 'true'
    } catch {
      /* localStorage unavailable (private mode, etc.) — just skip auto-start */
    }
    if (seen) return
    const t = window.setTimeout(() => setTourStep(0), 700)
    return () => window.clearTimeout(t)
  }, [])

  const startTour = useCallback(() => {
    setActiveView('dashboard')
    setTourStep(0)
  }, [])

  const endTour = useCallback(() => {
    setTourStep(null)
    try {
      localStorage.setItem(TOUR_SEEN_KEY, 'true')
    } catch {
      /* ignore */
    }
  }, [])

  const tourNext = useCallback(() => {
    setTourStep((i) => {
      if (i === null) return null
      if (i + 1 >= TOUR_STEPS.length) {
        endTour()
        return null
      }
      return i + 1
    })
  }, [endTour])

  const tourBack = useCallback(() => {
    setTourStep((i) => (i === null || i === 0 ? i : i - 1))
  }, [])

  // ---- Derived data for the Dashboard / Sessions views ----

  const sessionRows: SessionRow[] = useMemo(() => {
    // Prefer the real backend roster — it's what's actually live: the
    // built-in synthetic feed plus anything any client has driven. Fall back
    // to this tab's own memory only when the backend is unreachable.
    if (liveRows) {
      return liveRows.map((r) => {
        const key = (r.dominant_intent as IntentClass) || 'legitimate'
        return {
          sessionId: r.session_id,
          userId: r.user_id,
          dominantLabel: INTENT_LABELS[key] || r.dominant_intent || 'Unknown',
          dominantKey: key,
          confidence: r.confidence ?? 0,
          action: r.action as SessionRow['action'],
          updatedAt: r.last_seen * 1000,
        }
      })
    }
    return Object.values(sessionLog)
      .sort((a, b) => b.updatedAt - a.updatedAt)
      .map((rec) => {
        const dom = dominantOf(rec.intent)
        return {
          sessionId: rec.sessionId,
          userId: rec.userId,
          amount: extractAmount(rec.events),
          dominantLabel: INTENT_LABELS[dom.key],
          dominantKey: dom.key,
          confidence: dom.confidence,
          action: rec.intent.action,
          updatedAt: rec.updatedAt,
        }
      })
  }, [liveRows, sessionLog])

  const donutCounts: Record<IntentClass, number> = useMemo(() => {
    const counts = { ...EMPTY_INTENT_COUNTS }
    for (const r of sessionRows) counts[r.dominantKey as IntentClass] += 1
    return counts
  }, [sessionRows])

  const barValues: Record<IntentClass, number> = useMemo(() => {
    if (intent?.hypotheses) return intent.hypotheses as Record<IntentClass, number>
    const total = Object.values(donutCounts).reduce((a, b) => a + b, 0)
    if (total === 0) return EMPTY_INTENT_COUNTS
    return Object.fromEntries(
      Object.entries(donutCounts).map(([k, v]) => [k, v / total]),
    ) as Record<IntentClass, number>
  }, [intent, donutCounts])

  const activitySeries = useMemo(
    () => [
      { label: 'Events', color: 'var(--intent-social)', points: history.map((h) => h.events) },
      { label: 'Risk %', color: 'var(--intent-ato)', points: history.map((h) => h.risk * 100) },
    ],
    [history],
  )

  return (
    <div className="parallax-app">
      <Header
        sessionId={currentSessionId}
        userId={currentUserId}
        isStreaming={isStreaming}
        backendHealthy={backendHealthy}
        metrics={metrics}
        eventCount={events.length}
        onOpenBaseline={() => setShowBaselineModal(true)}
        onOpenEvaluation={() => setShowEvaluationModal(true)}
        onStartTour={startTour}
        onOpenDataStream={() => setShowDataStream(true)}
      />

      <NavBar active={activeView} onChange={setActiveView} />

      {activeView === 'dashboard' && (
        <DashboardView
          eventCount={events.length}
          evidenceCount={intent?.evidence?.length || 0}
          intentLabel={intent ? INTENT_LABELS[dominantOf(intent).key] : 'Idle'}
          action={intent?.action || 'Observing'}
          live={isStreaming && backendHealthy}
          donutCounts={donutCounts}
          barValues={barValues}
          activitySeries={activitySeries}
          sessionRows={sessionRows}
          onSelectSession={selectSession}
          onViewAllSessions={() => setActiveView('sessions')}
          onStartTour={startTour}
        />
      )}

      {activeView === 'sessions' && (
        <SessionsView rows={sessionRows} onSelect={selectSession} />
      )}

      {activeView === 'intent' && (
        <IntentView intent={intent} prevIntent={prevIntent} onAnswerProbe={handleAnswerProbe} />
      )}

      {activeView === 'activity' && (
        <ActivityView
          events={events}
          onClear={resetSession}
          onOpenComposer={() => setShowComposer(true)}
        />
      )}

      {activeView === 'playground' && (
        <PlaygroundView
          scenarios={DEMO_SCENARIOS}
          activeScenario={activeScenario}
          onSelectScenario={handleSelectScenario}
          currentStepIndex={currentStepIndex}
          isPlaying={isPlaying}
          onTogglePlay={() => setIsPlaying((p) => !p)}
          onStepForward={stepForward}
          onReset={resetSession}
          backendHealthy={backendHealthy}
          backendRunning={backendRunning}
          onRunBackend={runScenarioOnBackend}
          events={events}
          intent={intent}
          prevIntent={prevIntent}
          onAnswerProbe={handleAnswerProbe}
          onOpenComposer={() => setShowComposer(true)}
        />
      )}

      {/* Customer Baseline Inspector Modal */}
      <BaselineModal
        userId={currentUserId}
        isOpen={showBaselineModal}
        onClose={() => setShowBaselineModal(false)}
      />

      {/* Model Evaluation Audit Modal */}
      <EvaluationModal
        isOpen={showEvaluationModal}
        onClose={() => setShowEvaluationModal(false)}
      />

      {/* Hand-build one event and fire it at the pipeline — reachable from any view */}
      <EventComposer
        isOpen={showComposer}
        onClose={() => setShowComposer(false)}
        sessionId={currentSessionId}
        userId={currentUserId}
        backendHealthy={backendHealthy}
        busy={composerBusy}
        onSend={sendComposedEvent}
      />

      {/* Raw live data stream overlay */}
      <DataStreamOverlay isOpen={showDataStream} onClose={() => setShowDataStream(false)} />

      {/* First-run guided walkthrough — spotlights real buttons, step by step */}
      {tourStep !== null && (
        <TourGuide
          steps={TOUR_STEPS}
          stepIndex={tourStep}
          currentView={activeView}
          onChangeView={setActiveView}
          onNext={tourNext}
          onBack={tourBack}
          onSkip={endTour}
          interactionDone={TOUR_STEPS[tourStep]?.target === 'tour-play' ? isPlaying : false}
        />
      )}
    </div>
  )
}
