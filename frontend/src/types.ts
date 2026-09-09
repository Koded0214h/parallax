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

export interface IntentResponse {
  session_id: string
  hypotheses: Record<IntentClass, number>
  uncertainty: number
  action: PolicyAction
  evidence: string[]
  events_seen: number
  explanation?: string
  probe?: {
    prompt: string
    response?: string
    status: 'pending' | 'answered'
  }
}

export interface GatewayMetrics {
  status?: string
  sessions: number
  bus?: {
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
      min_nanos: number
      max_nanos: number
      p50_nanos: number
      p95_nanos: number
      p99_nanos: number
    }
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
