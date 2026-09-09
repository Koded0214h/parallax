// Parallax Gateway Client & Scenario Engine
// Supports live Go gateway ingestion + SSE streaming, with built-in
// demonstration scenarios matching prd.md §23, §24.

import type {
  GatewayMetrics,
  IntentResponse,
  ParallaxEvent,
  Scenario,
} from './types'

async function json<T>(res: Response): Promise<T> {
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`)
  return res.json() as Promise<T>
}

export const api = {
  health: () => fetch('/healthz').then((r) => json<{ status: string; sessions: number }>(r)),

  metrics: () => fetch('/v1/metrics').then((r) => json<GatewayMetrics>(r)),

  sendEvent: (e: ParallaxEvent) =>
    fetch('/v1/events', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(e),
    }).then((r) => json<{ event_id: string; session_id: string; seq: number; events: number }>(r)),

  intent: (sessionId: string) =>
    fetch(`/v1/sessions/${encodeURIComponent(sessionId)}/intent`).then((r) =>
      json<IntentResponse>(r),
    ),

  streamEvents(
    onEvent: (e: ParallaxEvent) => void,
    onStatusChange?: (connected: boolean) => void,
    sessionId?: string,
  ): () => void {
    const url = sessionId
      ? `/v1/stream?session_id=${encodeURIComponent(sessionId)}`
      : '/v1/stream'
    let es: EventSource | null = null
    let active = true

    function connect() {
      if (!active) return
      es = new EventSource(url)

      es.onopen = () => {
        onStatusChange?.(true)
      }

      es.onmessage = (m) => {
        try {
          const ev = JSON.parse(m.data) as ParallaxEvent
          onEvent(ev)
        } catch {
          /* ignore keep-alive frames */
        }
      }

      es.onerror = () => {
        onStatusChange?.(false)
        es?.close()
        if (active) {
          setTimeout(connect, 3000)
        }
      }
    }

    connect()

    return () => {
      active = false
      onStatusChange?.(false)
      es?.close()
    }
  },
}

// ============================================================================
// Scripted Scenarios (PRD §23, §24)
// ============================================================================

export const DEMO_SCENARIOS: Scenario[] = [
  {
    id: 'scenario-a',
    code: 'A',
    name: 'Legitimate Transfer',
    tagline: 'Routine transfer to recurring beneficiary from known device',
    description:
      'User logs in from primary device, selects established payee, enters baseline amount, and completes biometric OTP.',
    targetAction: 'ALLOW',
    targetIntentClass: 'legitimate',
    steps: [
      {
        delayMs: 400,
        event: {
          session_id: 'sess_legit_101',
          user_id: 'usr_sarah_chen',
          type: 'DEVICE_SEEN',
          metadata: {
            device_id: 'dev_iphone15_trusted',
            platform: 'iOS 18.2',
            ip_reputation: 'residential_clean',
            location: 'London, UK',
          },
        },
        targetIntent: {
          session_id: 'sess_legit_101',
          hypotheses: {
            legitimate: 0.65,
            accidental: 0.15,
            social_engineering: 0.1,
            account_takeover: 0.1,
          },
          uncertainty: 0.45,
          action: 'ALLOW',
          evidence: ['known_hardware_token', 'clean_network_asn'],
          events_seen: 1,
          explanation: 'Initial session handshake authenticated from primary device.',
        },
      },
      {
        delayMs: 650,
        event: {
          session_id: 'sess_legit_101',
          user_id: 'usr_sarah_chen',
          type: 'LOGIN',
          metadata: {
            auth_method: 'biometric_faceid',
            failed_attempts: 0,
            session_start: '12:14:02',
          },
        },
        targetIntent: {
          session_id: 'sess_legit_101',
          hypotheses: {
            legitimate: 0.78,
            accidental: 0.1,
            social_engineering: 0.08,
            account_takeover: 0.04,
          },
          uncertainty: 0.32,
          action: 'ALLOW',
          evidence: ['known_hardware_token', 'first_attempt_biometric_pass'],
          events_seen: 2,
          explanation: 'Fast biometric pass matches historical velocity envelope.',
        },
      },
      {
        delayMs: 800,
        event: {
          session_id: 'sess_legit_101',
          user_id: 'usr_sarah_chen',
          type: 'TRANSFER_STARTED',
          metadata: {
            beneficiary_id: 'ben_landlord_recurring',
            beneficiary_age_days: 482,
            cadence: 'monthly',
          },
        },
        targetIntent: {
          session_id: 'sess_legit_101',
          hypotheses: {
            legitimate: 0.85,
            accidental: 0.07,
            social_engineering: 0.05,
            account_takeover: 0.03,
          },
          uncertainty: 0.22,
          action: 'ALLOW',
          evidence: [
            'known_hardware_token',
            'established_beneficiary_480d',
            'expected_calendar_interval',
          ],
          events_seen: 3,
          explanation: 'Target account is an established payee with regular monthly cadence.',
        },
      },
      {
        delayMs: 900,
        event: {
          session_id: 'sess_legit_101',
          user_id: 'usr_sarah_chen',
          type: 'AMOUNT_ENTERED',
          metadata: {
            amount_gbp: 1250.0,
            user_baseline_avg: 1200.0,
            variance_ratio: 1.04,
          },
        },
        targetIntent: {
          session_id: 'sess_legit_101',
          hypotheses: {
            legitimate: 0.92,
            accidental: 0.04,
            social_engineering: 0.03,
            account_takeover: 0.01,
          },
          uncertainty: 0.12,
          action: 'ALLOW',
          evidence: [
            'known_hardware_token',
            'established_beneficiary_480d',
            'amount_within_1.05x_baseline',
          ],
          events_seen: 4,
          explanation: 'Amount (£1,250) matches 99% expected historical baseline.',
        },
      },
      {
        delayMs: 600,
        event: {
          session_id: 'sess_legit_101',
          user_id: 'usr_sarah_chen',
          type: 'TRANSFER_COMPLETED',
          metadata: {
            auth_factor: 'hardware_secure_enclave',
            settlement_status: 'authorized',
          },
        },
        targetIntent: {
          session_id: 'sess_legit_101',
          hypotheses: {
            legitimate: 0.96,
            accidental: 0.02,
            social_engineering: 0.01,
            account_takeover: 0.01,
          },
          uncertainty: 0.06,
          action: 'ALLOW',
          evidence: [
            'known_hardware_token',
            'established_beneficiary_480d',
            'amount_within_1.05x_baseline',
            'secure_enclave_signed',
          ],
          events_seen: 5,
          explanation: 'High confidence legitimate transaction. Proportional decision: ALLOW.',
        },
      },
    ],
  },

  {
    id: 'scenario-b',
    code: 'B',
    name: 'Account Takeover (ATO)',
    tagline: 'New device, credential change, new payee, rapid account drain',
    description:
      'Adversary gains access via new device fingerprint, modifies credentials, rapidly introduces external payee, and attempts maximum balance transfer.',
    targetAction: 'BLOCK',
    targetIntentClass: 'account_takeover',
    steps: [
      {
        delayMs: 400,
        event: {
          session_id: 'sess_ato_882',
          user_id: 'usr_marcus_vance',
          type: 'DEVICE_CHANGED',
          metadata: {
            new_device: 'Linux x86_64 / Headless Chrome',
            ip_origin: 'VPN Datacenter (Frankfurt)',
            fingerprint_mismatch: true,
          },
        },
        targetIntent: {
          session_id: 'sess_ato_882',
          hypotheses: {
            legitimate: 0.35,
            accidental: 0.1,
            social_engineering: 0.15,
            account_takeover: 0.4,
          },
          uncertainty: 0.72,
          action: 'VERIFY',
          evidence: ['unknown_device_architecture', 'datacenter_ip_subnet'],
          events_seen: 1,
          explanation: 'Access attempt from unrecognised headless Linux client via VPN.',
        },
      },
      {
        delayMs: 650,
        event: {
          session_id: 'sess_ato_882',
          user_id: 'usr_marcus_vance',
          type: 'PASSWORD_CHANGED',
          metadata: {
            previous_pw_age_days: 210,
            reset_channel: 'recovery_sms_intercepted',
          },
        },
        targetIntent: {
          session_id: 'sess_ato_882',
          hypotheses: {
            legitimate: 0.15,
            accidental: 0.05,
            social_engineering: 0.1,
            account_takeover: 0.7,
          },
          uncertainty: 0.44,
          action: 'VERIFY',
          evidence: [
            'unknown_device_architecture',
            'datacenter_ip_subnet',
            'credential_rotation_on_new_device',
          ],
          events_seen: 2,
          explanation: 'Immediate credential update following first appearance of foreign device.',
        },
      },
      {
        delayMs: 700,
        event: {
          session_id: 'sess_ato_882',
          user_id: 'usr_marcus_vance',
          type: 'BENEFICIARY_CREATED',
          metadata: {
            beneficiary_name: 'FastCrypt Settlement AG',
            sort_code: '04-00-04',
            account_num: '99214418',
            created_seconds_ago: 8,
          },
        },
        targetIntent: {
          session_id: 'sess_ato_882',
          hypotheses: {
            legitimate: 0.06,
            accidental: 0.02,
            social_engineering: 0.08,
            account_takeover: 0.84,
          },
          uncertainty: 0.28,
          action: 'BLOCK',
          evidence: [
            'unknown_device_architecture',
            'credential_rotation_on_new_device',
            'new_beneficiary_created_sub_60s',
            'high_risk_institution_sortcode',
          ],
          events_seen: 3,
          explanation: 'Rapid sequence of credential change followed immediately by new mule account creation.',
        },
      },
      {
        delayMs: 850,
        event: {
          session_id: 'sess_ato_882',
          user_id: 'usr_marcus_vance',
          type: 'AMOUNT_ENTERED',
          metadata: {
            amount_gbp: 9850.0,
            user_baseline_avg: 85.0,
            balance_drain_percent: 97.4,
            variance_ratio: 115.8,
          },
        },
        targetIntent: {
          session_id: 'sess_ato_882',
          hypotheses: {
            legitimate: 0.02,
            accidental: 0.01,
            social_engineering: 0.05,
            account_takeover: 0.92,
          },
          uncertainty: 0.14,
          action: 'BLOCK',
          evidence: [
            'unknown_device_architecture',
            'credential_rotation_on_new_device',
            'new_beneficiary_created_sub_60s',
            'amount_115x_baseline_drain',
            'rapid_automated_interaction_cadence',
          ],
          events_seen: 4,
          explanation: 'Observed behaviour exhibits definitive account takeover indicators. Complete balance liquidation attempted.',
        },
      },
      {
        delayMs: 500,
        event: {
          session_id: 'sess_ato_882',
          user_id: 'usr_marcus_vance',
          type: 'TRANSFER_FAILED',
          metadata: {
            policy_action: 'ENFORCED_BLOCK',
            escalation_ticket: 'SEC-8921',
          },
        },
        targetIntent: {
          session_id: 'sess_ato_882',
          hypotheses: {
            legitimate: 0.01,
            accidental: 0.01,
            social_engineering: 0.04,
            account_takeover: 0.94,
          },
          uncertainty: 0.08,
          action: 'BLOCK',
          evidence: [
            'unknown_device_architecture',
            'credential_rotation_on_new_device',
            'amount_115x_baseline_drain',
            'automated_sequence_timing',
          ],
          events_seen: 5,
          explanation: 'High confidence Account Takeover. Proportional decision: BLOCK & ESCALATE.',
        },
      },
    ],
  },

  {
    id: 'scenario-c',
    code: 'C',
    name: 'Social Engineering',
    tagline: 'Known device, valid auth, high uncertainty → Intent Probe unfolds',
    description:
      'Authorized transfer from legitimate phone, but recipient is brand new and amount is anomalous. Rather than blocking blindly, Parallax probes intent.',
    targetAction: 'ESCALATE',
    targetIntentClass: 'social_engineering',
    probePrompt: 'Please confirm: What is the purpose of this £4,250 transfer?',
    probeResponse:
      'The bank fraud squad phoned me saying my account was compromised and instructed me to move funds into this safe holding account.',
    steps: [
      {
        delayMs: 400,
        event: {
          session_id: 'sess_soceng_505',
          user_id: 'usr_elena_rostova',
          type: 'LOGIN',
          metadata: {
            device_id: 'dev_trusted_pixel8',
            auth: 'valid_passkey',
            location: 'Home (Manchester)',
          },
        },
        targetIntent: {
          session_id: 'sess_soceng_505',
          hypotheses: {
            legitimate: 0.72,
            accidental: 0.1,
            social_engineering: 0.12,
            account_takeover: 0.06,
          },
          uncertainty: 0.4,
          action: 'ALLOW',
          evidence: ['known_hardware_token', 'valid_passkey_local'],
          events_seen: 1,
          explanation: 'Legitimate customer authentication on recognised device.',
        },
      },
      {
        delayMs: 650,
        event: {
          session_id: 'sess_soceng_505',
          user_id: 'usr_elena_rostova',
          type: 'BENEFICIARY_CREATED',
          metadata: {
            beneficiary_name: 'SafeVault Holding Escrow',
            created_in_session: true,
            sort_code: '20-00-00',
          },
        },
        targetIntent: {
          session_id: 'sess_soceng_505',
          hypotheses: {
            legitimate: 0.42,
            accidental: 0.12,
            social_engineering: 0.38,
            account_takeover: 0.08,
          },
          uncertainty: 0.68,
          action: 'VERIFY',
          evidence: ['known_hardware_token', 'new_unverified_payee'],
          events_seen: 2,
          explanation: 'New payee created during active session with naming patterns typical of imposter accounts.',
        },
      },
      {
        delayMs: 800,
        event: {
          session_id: 'sess_soceng_505',
          user_id: 'usr_elena_rostova',
          type: 'AMOUNT_ENTERED',
          metadata: {
            amount_gbp: 4250.0,
            user_baseline_avg: 120.0,
            variance_ratio: 35.4,
            interaction_hesitation_ms: 14200,
          },
        },
        targetIntent: {
          session_id: 'sess_soceng_505',
          hypotheses: {
            legitimate: 0.28,
            accidental: 0.14,
            social_engineering: 0.52,
            account_takeover: 0.06,
          },
          uncertainty: 0.82,
          action: 'PROBE',
          evidence: [
            'known_hardware_token',
            'amount_35x_baseline',
            'unusual_input_hesitation_14s',
            'high_risk_uncertainty_boundary',
          ],
          events_seen: 3,
          explanation:
            'Conflicting signals: Fully authenticated known user, but extreme amount deviation and hesitation. Risk is elevated while intent explanation is uncertain. Decision: Trigger Intent Probe.',
          probe: {
            prompt: 'Please confirm: What is the purpose of this £4,250 transfer?',
            status: 'pending',
          },
        },
      },
      {
        delayMs: 1200,
        event: {
          session_id: 'sess_soceng_505',
          user_id: 'usr_elena_rostova',
          type: 'INTENT_PROBE_STARTED',
          metadata: {
            probe_id: 'prb_99182',
            trigger_policy: 'UNCERTAINTY_EXCEEDS_THRESHOLD',
            probe_type: 'PAYMENT_CONTEXT_FREE_TEXT',
          },
        },
        targetIntent: {
          session_id: 'sess_soceng_505',
          hypotheses: {
            legitimate: 0.25,
            accidental: 0.12,
            social_engineering: 0.55,
            account_takeover: 0.08,
          },
          uncertainty: 0.84,
          action: 'PROBE',
          evidence: [
            'amount_35x_baseline',
            'conflicting_auth_vs_profile',
            'intent_probe_dispatched_awaiting_reply',
          ],
          events_seen: 4,
          explanation: 'Intent probe active. Awaiting natural contextual feedback from user before irreversible authorization.',
          probe: {
            prompt: 'Please confirm: What is the purpose of this £4,250 transfer?',
            status: 'pending',
          },
        },
      },
      {
        delayMs: 1500,
        event: {
          session_id: 'sess_soceng_505',
          user_id: 'usr_elena_rostova',
          type: 'INTENT_PROBE_RESPONSE',
          metadata: {
            probe_id: 'prb_99182',
            user_response_text:
              'The bank fraud squad phoned me saying my account was compromised and instructed me to move funds into this safe holding account.',
            nlp_signals: ['safe_account_keyword', 'bank_impersonation', 'urgent_duress'],
          },
        },
        targetIntent: {
          session_id: 'sess_soceng_505',
          hypotheses: {
            legitimate: 0.03,
            accidental: 0.02,
            social_engineering: 0.91,
            account_takeover: 0.04,
          },
          uncertainty: 0.11,
          action: 'ESCALATE',
          evidence: [
            'known_hardware_token',
            'amount_35x_baseline',
            'probe_reply_indicates_bank_impersonation_scam',
            'safe_account_narrative_detected',
          ],
          events_seen: 5,
          explanation:
            'Intent probe response collapsed uncertainty: User states they were directed by a caller claiming to be bank security. Clear Authorized Push Payment (APP) scam in progress. Action: INTERVENE & ESCALATE to human fraud specialist.',
          probe: {
            prompt: 'Please confirm: What is the purpose of this £4,250 transfer?',
            response:
              'The bank fraud squad phoned me saying my account was compromised and instructed me to move funds into this safe holding account.',
            status: 'answered',
          },
        },
      },
    ],
  },

  {
    id: 'scenario-d',
    code: 'D',
    name: 'Accidental Transfer',
    tagline: 'Digit transpose, repeated frantic cancellation attempts',
    description:
      'User intends to pay £55.00 but accidentally enters £5,500.00, rapidly hits backspace and attempts quick retry.',
    targetAction: 'VERIFY',
    targetIntentClass: 'accidental',
    steps: [
      {
        delayMs: 400,
        event: {
          session_id: 'sess_acc_303',
          user_id: 'usr_david_okafor',
          type: 'LOGIN',
          metadata: {
            device_id: 'dev_samsung_trusted',
            auth: 'fingerprint_pass',
          },
        },
        targetIntent: {
          session_id: 'sess_acc_303',
          hypotheses: {
            legitimate: 0.8,
            accidental: 0.1,
            social_engineering: 0.05,
            account_takeover: 0.05,
          },
          uncertainty: 0.3,
          action: 'ALLOW',
          evidence: ['known_hardware_token', 'routine_login'],
          events_seen: 1,
          explanation: 'Standard authenticated session.',
        },
      },
      {
        delayMs: 650,
        event: {
          session_id: 'sess_acc_303',
          user_id: 'usr_david_okafor',
          type: 'TRANSFER_STARTED',
          metadata: {
            beneficiary_id: 'ben_utility_electric',
            typical_payment: 55.0,
          },
        },
        targetIntent: {
          session_id: 'sess_acc_303',
          hypotheses: {
            legitimate: 0.82,
            accidental: 0.1,
            social_engineering: 0.04,
            account_takeover: 0.04,
          },
          uncertainty: 0.28,
          action: 'ALLOW',
          evidence: ['known_hardware_token', 'utility_payee_history'],
          events_seen: 2,
          explanation: 'Transfer initiated for recurring utility provider.',
        },
      },
      {
        delayMs: 750,
        event: {
          session_id: 'sess_acc_303',
          user_id: 'usr_david_okafor',
          type: 'AMOUNT_ENTERED',
          metadata: {
            amount_gbp: 5500.0,
            intended_likely: 55.0,
            magnitude_error_factor: 100.0,
            interaction_rapid_backspaces: 4,
          },
        },
        targetIntent: {
          session_id: 'sess_acc_303',
          hypotheses: {
            legitimate: 0.18,
            accidental: 0.68,
            social_engineering: 0.08,
            account_takeover: 0.06,
          },
          uncertainty: 0.38,
          action: 'VERIFY',
          evidence: [
            'known_hardware_token',
            'utility_payee_history',
            'amount_100x_magnitude_slip',
            'rapid_backspace_keystroke_burst',
          ],
          events_seen: 3,
          explanation: 'Amount (£5,500) represents exactly 100× normal utility invoice (£55). Frantic keystroke edits detected. Probability strongly favors accidental slip.',
        },
      },
    ],
  },
]
