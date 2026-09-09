// Thin client for the Parallax gateway. Contracts mirror prd.md §33 and
// backend/internal/httpapi/router.go. No decision logic lives here — the
// frontend only observes.

export type EventType =
  | 'LOGIN'
  | 'LOGOUT'
  | 'DEVICE_SEEN'
  | 'DEVICE_CHANGED'
  | 'PASSWORD_CHANGED'
  | 'PIN_CHANGED'
  | 'BENEFICIARY_CREATED'
  | 'BENEFICIARY_MODIFIED'
  | 'TRANSFER_STARTED'
  | 'AMOUNT_ENTERED'
  | 'OTP_REQUESTED'
  | 'OTP_VERIFIED'
  | 'TRANSFER_COMPLETED'
  | 'TRANSFER_FAILED'
  | 'INTENT_PROBE_STARTED'
  | 'INTENT_PROBE_RESPONSE'

export type PolicyAction = 'ALLOW' | 'VERIFY' | 'PROBE' | 'BLOCK' | 'ESCALATE'

export interface ParallaxEvent {
  event_id?: string
  session_id: string
  user_id: string
  type: string
  timestamp?: number
  metadata?: Record<string, unknown>
  seq?: number
  ingested_at_nanos?: number
}

export interface ProbeOption {
  id: string
  text: string
  category?: string
}

export interface ProbePayload {
  probe_id: string
  prompt: string
  options: ProbeOption[]
  allow_freeform: boolean
  completed: boolean
  response?: string
}

export interface IntentResponse {
  session_id: string
  hypotheses: Record<string, number>
  uncertainty: number
  action: string
  evidence: string[]
  events_seen: number
  explanation?: string
  confidence?: string
  risk_score?: number
  dominant_intent?: string
  probe?: ProbePayload
}

export interface ScenarioInfo {
  id: string
  title: string
  description: string
  expected_action: string
}

export interface ScenarioRunResult {
  scenario: string
  session_id: string
  events_ingested: number
  intent: IntentResponse
}

async function json<T>(res: Response): Promise<T> {
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`)
  return res.json() as Promise<T>
}

export const api = {
  health: () => fetch('/health').then((r) => json<{ status: string; uptime_seconds?: number }>(r)),

  sendEvent: (e: ParallaxEvent) =>
    fetch('/v1/events', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(e),
    }).then((r) => json<{ session_id: string; events: number; seq: number; event_id: string }>(r)),

  intent: (sessionId: string) =>
    fetch(`/v1/sessions/${encodeURIComponent(sessionId)}/intent`).then((r) =>
      json<IntentResponse>(r),
    ),

  respondProbe: (sessionId: string, responseText: string) =>
    fetch(`/v1/probes/respond`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ session_id: sessionId, response: responseText }),
    }).then((r) => json<IntentResponse>(r)),

  getScenarios: () =>
    fetch('/v1/scenarios').then((r) => json<{ scenarios: ScenarioInfo[] }>(r)),

  runScenario: (scenarioId: string) =>
    fetch(`/v1/scenarios/${encodeURIComponent(scenarioId)}/run`, {
      method: 'POST',
    }).then((r) => json<ScenarioRunResult>(r)),

  getBaseline: (userId: string) =>
    fetch(`/v1/baselines/${encodeURIComponent(userId)}`).then((r) => json<Record<string, unknown>>(r)),

  resetState: () =>
    fetch('/v1/reset', {
      method: 'POST',
    }).then((r) => json<{ status: string; message: string }>(r)),

  getEvaluation: () =>
    fetch('/v1/evaluation').then((r) => json<Record<string, unknown>>(r)),

  /**
   * Subscribe to the live event stream (SSE). Pass a sessionId to filter.
   * Returns an unsubscribe function.
   */
  streamEvents(
    onEvent: (e: ParallaxEvent) => void,
    sessionId?: string,
  ): () => void {
    const url = sessionId
      ? `/v1/stream?session_id=${encodeURIComponent(sessionId)}`
      : '/v1/stream'
    const es = new EventSource(url)
    es.onmessage = (m) => {
      try {
        onEvent(JSON.parse(m.data) as ParallaxEvent)
      } catch {
        /* ignore keep-alive / malformed frames */
      }
    }
    return () => es.close()
  },
}
