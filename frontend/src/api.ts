// Thin client for the Parallax gateway. Contracts mirror prd.md §33 and
// backend/internal/httpapi/router.go. No decision logic lives here — the
// frontend only observes.

export interface ParallaxEvent {
  session_id: string
  user_id: string
  type: string
  timestamp?: number
  metadata?: Record<string, unknown>
}

export interface IntentResponse {
  hypotheses: Record<string, number>
  uncertainty: number
  action: string
  evidence: string[]
}

async function json<T>(res: Response): Promise<T> {
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`)
  return res.json() as Promise<T>
}

export const api = {
  health: () => fetch('/healthz').then((r) => json<{ status: string }>(r)),

  sendEvent: (e: ParallaxEvent) =>
    fetch('/v1/events', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(e),
    }).then((r) => json<{ session_id: string; events: number }>(r)),

  intent: (sessionId: string) =>
    fetch(`/v1/sessions/${encodeURIComponent(sessionId)}/intent`).then((r) =>
      json<IntentResponse>(r),
    ),
}
