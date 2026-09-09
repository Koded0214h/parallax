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
  type: EventType | (string & {})
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
    Subscribers?: number
    published?: number
    Published?: number
    dropped?: number
    Dropped?: number
  }
  pool?: {
    workers?: number
    Workers?: number
    processed?: number
    Processed?: number
    failed?: number
    Failed?: number
    panicked?: number
    Panicked?: number
    in_flight?: number
    InFlight?: number
    latency?: {
      count?: number
      Count?: number
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
      P50?: number
      P95?: number
      P99?: number
      Min?: number
      Max?: number
      Mean?: number
    }
    Latency?: {
      Count?: number
      Min?: number
      Max?: number
      Mean?: number
      P50?: number
      P95?: number
      P99?: number
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
  code: string // 'A' | 'B' | 'C' | 'D' | 'E'
  tagline: string
  description: string
  targetAction: PolicyAction
  targetIntentClass: IntentClass | string
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

export interface KnownBeneficiary {
  beneficiary_id: string
  name: string
  account_number: string
  bank_name: string
  transfer_count: number
  total_sent: number
  created_at?: string
}

export interface KnownDevice {
  device_id: string
  device_model: string
  use_count: number
  user_agent?: string
  first_seen?: string
  last_seen?: string
}

export interface CustomerBaseline {
  user_id: string
  typical_amount_min: number
  typical_amount_max: number
  typical_amount_mean: number
  typical_amount_std_dev: number
  total_transfers: number
  typical_start_hour: number
  typical_end_hour: number
  known_beneficiaries: Record<string, KnownBeneficiary>
  known_devices: Record<string, KnownDevice>
  typical_session_duration_sec?: number
  typical_events_per_minute?: number
  max_hourly_transfers?: number
  max_daily_amount?: number
  last_password_change?: string
  last_pin_change?: string
}

export interface ClassMetrics {
  class?: string
  true_positives?: number
  false_positives?: number
  true_negatives?: number
  false_negatives?: number
  precision: number
  recall: number
  f1: number
  false_positive_rate: number
}

export type ConfusionMatrix = Record<string, Record<string, number>>

export interface EvaluationReport {
  total_sessions: number
  overall_accuracy: number
  legitimate_false_positive_rate: number
  ato_catch_rate: number
  soc_eng_catch_rate: number
  metrics_by_class: Record<string, ClassMetrics>
  confusion_matrix?: ConfusionMatrix
  average_decision_latency_us?: number
}
