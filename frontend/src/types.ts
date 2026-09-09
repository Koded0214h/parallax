// Parallax System Types & Contracts
// Mirrors pkg/contracts and prd.md §10, §21, §33

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

export type IntentClass =
  | 'legitimate'
  | 'accidental'
  | 'social_engineering'
  | 'account_takeover'

export interface ParallaxEvent {
  event_id?: string
  session_id: string
  user_id: string
  type: EventType
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
  probe_id?: string
  prompt: string
  options?: ProbeOption[]
  allow_freeform?: boolean
  completed?: boolean
  response?: string
  status?: 'pending' | 'answered'
}

export interface IntentResponse {
  session_id: string
  hypotheses: Record<IntentClass, number> | Record<string, number>
  uncertainty: number
  action: PolicyAction
  evidence: string[]
  events_seen: number
  explanation?: string
  confidence?: string
  risk_score?: number
  dominant_intent?: string
  probe?: ProbePayload
}

export interface GatewayMetrics {
  status?: string
  uptime_seconds?: number
  sessions: number
  bus?: {
    subscribers?: number
    published: number
    dropped: number
  }
  pool?: {
    workers: number
    processed: number
    failed: number
    panicked: number
    in_flight: number
    latency?: {
      count: number
      min_nanos?: number
      max_nanos?: number
      p50_nanos?: number
      p95_nanos?: number
      p99_nanos?: number
      min_ns?: number
      max_ns?: number
      p50_ns?: number
      p95_ns?: number
      p99_ns?: number
    }
  }
  wal?: {
    is_online?: boolean
    buffered_events?: number
    synced_total?: number
    last_sync_time?: number
    mode?: string
  }
}

export interface ScenarioStep {
  delayMs: number
  event: ParallaxEvent
  // In simulated / demonstration mode:
  targetIntent?: IntentResponse
}

export interface Scenario {
  id: string
  name: string
  code: string // 'A' | 'B' | 'C' | 'D'
  tagline: string
  description: string
  targetAction: PolicyAction
  targetIntentClass: IntentClass
  steps: ScenarioStep[]
  probePrompt?: string
  probeResponse?: string
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
