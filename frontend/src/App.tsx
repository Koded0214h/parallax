import { useCallback, useEffect, useRef, useState } from 'react'
import { api, DEMO_SCENARIOS } from './api'
import { DecisionPanel } from './components/DecisionPanel'
import { EventTimeline } from './components/EventTimeline'
import { Header } from './components/Header'
import { IntentDistribution } from './components/IntentDistribution'
import { ScenarioBar } from './components/ScenarioBar'
import type {
  GatewayMetrics,
  IntentResponse,
  ParallaxEvent,
  Scenario,
} from './types'
import './App.css'

export default function App() {
  const [activeScenario, setActiveScenario] = useState<Scenario>(DEMO_SCENARIOS[2]) // Scenario C as centerpiece default
  const [currentStepIndex, setCurrentStepIndex] = useState<number>(0)
  const [isPlaying, setIsPlaying] = useState<boolean>(false)

  const [events, setEvents] = useState<ParallaxEvent[]>([])
  const [intent, setIntent] = useState<IntentResponse | null>(null)
  const [prevIntent, setPrevIntent] = useState<IntentResponse | null>(null)

  const [backendHealthy, setBackendHealthy] = useState<boolean>(false)
  const [isStreaming, setIsStreaming] = useState<boolean>(false)
  const [metrics, setMetrics] = useState<GatewayMetrics | null>(null)

  const timerRef = useRef<number | null>(null)

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

    // Try posting to Go gateway if online
    if (backendHealthy) {
      api.sendEvent(enrichedEvent).catch(() => {})
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
    setIsPlaying(false)
    if (timerRef.current) clearTimeout(timerRef.current)
    setCurrentStepIndex(0)
    setEvents([])
    setPrevIntent(null)
    setIntent(null)
  }, [])

  // Switch scenario
  const handleSelectScenario = useCallback(
    (scenario: Scenario) => {
      setIsPlaying(false)
      if (timerRef.current) clearTimeout(timerRef.current)
      setActiveScenario(scenario)
      setCurrentStepIndex(0)
      setEvents([])
      setPrevIntent(null)
      setIntent(null)
    },
    [],
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

      if (backendHealthy) {
        api.sendEvent(responseEvent).catch(() => {})
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

  return (
    <div className="parallax-app">
      <Header
        sessionId={currentSessionId}
        userId={currentUserId}
        isStreaming={isStreaming}
        backendHealthy={backendHealthy}
        metrics={metrics}
        eventCount={events.length}
        onResetSession={resetSession}
      />

      <ScenarioBar
        scenarios={DEMO_SCENARIOS}
        activeScenario={activeScenario}
        onSelectScenario={handleSelectScenario}
        currentStepIndex={currentStepIndex}
        isPlaying={isPlaying}
        onTogglePlay={() => setIsPlaying((p) => !p)}
        onStepForward={stepForward}
        onReset={resetSession}
      />

      <main className="dashboard-grid">
        {/* View 1: Live Event Stream */}
        <section className="dashboard-column col-stream">
          <EventTimeline events={events} onClear={resetSession} />
        </section>

        {/* View 2: Intent State & Distribution */}
        <section className="dashboard-column col-intent">
          <IntentDistribution intent={intent} prevIntent={prevIntent} />
        </section>

        {/* View 3: Decision & Evidence Engine */}
        <section className="dashboard-column col-decision">
          <DecisionPanel intent={intent} onAnswerProbe={handleAnswerProbe} />
        </section>
      </main>
    </div>
  )
}
