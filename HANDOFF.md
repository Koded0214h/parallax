# Parallax Gateway & Engine API Specification: Frontend Handoff Guide

This document is the engineering reference for the frontend developer (Fiope) to integrate with the Parallax Intelligence & Decision Gateway.

---

## 1. Gateway Connection & Dev Setup

### 1.1 Local Development
- **Backend Gateway**: Runs on `http://localhost:8080` by default (configurable via `PARALLAX_ADDR`).
- **Frontend Dev Server**: Runs on `http://localhost:5173` via Vite.
- **Proxy Configuration**: Dev requests to `/v1/*`, `/health`, and `/healthz` are automatically forwarded to `:8080` through `vite.config.ts`. The frontend can invoke relative paths directly (e.g., `fetch('/v1/events')`).

### 1.2 Production / Render Deployment
- When deployed to Render, the backend serves all routes including `/health`.
- Render background ping / uptime cron jobs should target `GET /health` every 5 to 10 minutes to prevent container idling.

---

## 2. API Endpoints Catalog

### 2.1 Health & Liveness Checks

#### `GET /health` & `GET /healthz`
Returns gateway status, uptime, active session counts, bus metrics, worker pool metrics, and write-ahead log (WAL) status.

**Response `200 OK`**:
```json
{
  "status": "ok",
  "uptime_seconds": 3482,
  "sessions": 3,
  "bus": {
    "subscribers": 2,
    "published": 1420,
    "dropped": 0
  },
  "pool": {
    "workers": 4,
    "in_flight": 0,
    "processed": 1420,
    "failed": 0,
    "panicked": 0,
    "latency": {
      "count": 1420,
      "min_ns": 127073,
      "max_ns": 39392442,
      "p50_ns": 738279,
      "p95_ns": 11202342,
      "p99_ns": 20419254
    }
  },
  "wal": {
    "is_online": true,
    "buffered_events": 0,
    "synced_total": 1420,
    "last_sync_time": 1788594121,
    "mode": "ONLINE_SYNCHRONIZED"
  }
}
```

---

### 2.2 Event Ingestion

#### `POST /v1/events`
Ingests a session or transaction event into the pipeline.

**Headers**:
- `Content-Type: application/json`

**Request Body**:
```json
{
  "session_id": "sess_demo_001",
  "user_id": "user_001",
  "type": "AMOUNT_ENTERED",
  "timestamp": 1788594121,
  "metadata": {
    "amount": 450000.0,
    "beneficiary_id": "ben_mother",
    "device_id": "device_primary"
  }
}
```

**Recognized Event Types**:
- `LOGIN`
- `LOGOUT`
- `DEVICE_SEEN`
- `DEVICE_CHANGED`
- `PASSWORD_CHANGED`
- `PIN_CHANGED`
- `BENEFICIARY_CREATED`
- `BENEFICIARY_MODIFIED`
- `TRANSFER_STARTED`
- `AMOUNT_ENTERED`
- `OTP_REQUESTED`
- `OTP_VERIFIED`
- `TRANSFER_COMPLETED`
- `TRANSFER_FAILED`
- `INTENT_PROBE_STARTED`
- `INTENT_PROBE_RESPONSE`

**Response `202 Accepted`**:
```json
{
  "event_id": "evt_01J8ABC123...",
  "session_id": "sess_demo_001",
  "seq": 2,
  "events": 2
}
```

**Errors**:
- `400 Bad Request`: Validation failure (missing `session_id`, missing `user_id`, unknown `type`).
  ```json
  { "error": "invalid event: unknown type \"INVALID_TYPE\"" }
  ```

---

### 2.3 Real-Time Intent Inference

#### `GET /v1/sessions/{id}/intent`
Retrieves the latest evaluated intent state, probability distribution, risk score, uncertainty, policy action, and explanation for a session.

**Response `200 OK`**:
```json
{
  "session_id": "sess_demo_001",
  "hypotheses": {
    "legitimate": 0.18,
    "accidental": 0.07,
    "social_engineering": 0.68,
    "account_takeover": 0.07
  },
  "uncertainty": 0.48,
  "action": "PROBE",
  "evidence": [
    "Transfer amount is significantly higher than user baseline",
    "Recipient beneficiary has never been sent money before",
    "Recipient was added immediately before transfer",
    "Conflicting evidence: known device and valid credentials conflict with anomalous recipient and amount"
  ],
  "events_seen": 4,
  "explanation": "This transaction has been paused for an intent probe: while authentic credentials were used, the transfer details deviate significantly from normal activity, creating high uncertainty between legitimate intent and social engineering.",
  "confidence": "MODERATE",
  "risk_score": 0.65,
  "dominant_intent": "social_engineering",
  "probe": {
    "probe_id": "probe_sess_demo_001_1788594121",
    "prompt": "To ensure your transaction is safe, please tell us the primary purpose of this transfer:",
    "options": [
      {
        "id": "opt_vendor_friend",
        "text": "Paying a trusted friend, family member, or verified merchant.",
        "category": "legitimate"
      },
      {
        "id": "opt_bank_agent",
        "text": "A bank representative or security officer told me to transfer funds to protect my account.",
        "category": "impersonation"
      },
      {
        "id": "opt_lottery_job",
        "text": "Processing fee for a prize, lottery, grant, job offer, or investment payout.",
        "category": "advance_fee"
      },
      {
        "id": "opt_emergency_call",
        "text": "Responding to an urgent call or message about an unexpected crisis.",
        "category": "urgency"
      }
    ],
    "allow_freeform": true,
    "completed": false
  }
}
```

**Policy Actions**:
- `ALLOW`: Approved automatically; low risk, high legitimate confidence.
- `VERIFY`: Step-up secondary verification (e.g., accidental zero-padding typo check).
- `PROBE`: Intent probe required; active contextual question presented.
- `BLOCK`: Payment stopped; critical account takeover or confirmed impersonation.
- `ESCALATE`: Contradictory evidence routed to fraud operations desk.

---

### 2.4 Contextual Intent Probing

When `action === "PROBE"` and `probe.completed === false`, the UI should prompt the user with the questions and options defined in `intent.probe`.

#### `POST /v1/probes/respond` or `POST /v1/sessions/{id}/probe`
Submits the user's selected option or typed context, stores the response event, immediately recalculates the intent distribution, and returns the updated `IntentResponse`.

**Request Body**:
```json
{
  "session_id": "sess_demo_001",
  "response": "A bank representative told me to transfer funds to protect my account"
}
```

**Response `200 OK`**:
Returns the recalculated `IntentResponse` where:
- `hypotheses.social_engineering` increases to $\ge 90\%$.
- `uncertainty` decreases to $\le 0.15$.
- `action` shifts from `PROBE` to `BLOCK`.
- `explanation` updates with context-specific fraud prevention guidance.

---

### 2.5 Scripted Demo Scenarios (One-Click Dashboard)

To make demonstrating the system effortless, the backend includes pre-scripted end-to-end scenarios.

#### `GET /v1/scenarios`
Lists all available scenarios.

**Response `200 OK`**:
```json
{
  "scenarios": [
    {
      "id": "legitimate",
      "title": "Scenario A — Legitimate Transfer",
      "description": "Habitual payee, normal amount, recognized device, standard hours. Allowed with high confidence.",
      "expected_action": "ALLOW"
    },
    {
      "id": "account_takeover",
      "title": "Scenario B — Account Takeover",
      "description": "Unrecognized device, rapid password change, mule beneficiary addition, high-value transfer. Blocked immediately.",
      "expected_action": "BLOCK"
    },
    {
      "id": "social_engineering",
      "title": "Scenario C — Social Engineering (Authorized Push Payment)",
      "description": "Customer's real phone and valid OTP, but urgent transfer to new recipient under external impersonation. Triggers Intent Probe.",
      "expected_action": "PROBE -> BLOCK"
    },
    {
      "id": "accidental",
      "title": "Scenario D — Accidental Transfer (Zero-Padding Error)",
      "description": "Habitual beneficiary with 10x normal amount (extra zero typo). Requires step-up confirmation.",
      "expected_action": "VERIFY"
    },
    {
      "id": "ambiguous",
      "title": "Scenario E — Ambiguous / Incomplete Session",
      "description": "Early session state with incomplete telemetry. System admits uncertainty rather than guessing.",
      "expected_action": "UNKNOWN / NEED MORE EVIDENCE"
    }
  ]
}
```

#### `POST /v1/scenarios/{id}/run`
Executes the specified scenario end-to-end and returns the newly generated session ID and computed intent.

**URL Parameter**:
- `id`: `legitimate`, `account_takeover`, `social_engineering`, `accidental`, or `ambiguous`.

**Response `200 OK`**:
```json
{
  "scenario": "legitimate",
  "session_id": "sess_legitimate_1788594200",
  "events_ingested": 5,
  "intent": {
    "session_id": "sess_legitimate_1788594200",
    "action": "ALLOW",
    "hypotheses": {
      "legitimate": 0.96,
      "accidental": 0.02,
      "social_engineering": 0.01,
      "account_takeover": 0.01
    },
    "uncertainty": 0.02,
    "confidence": "HIGH",
    "risk_score": 0.05,
    "dominant_intent": "legitimate",
    "explanation": "This transaction was approved because the session interaction, payee, and device align with normal customer history."
  }
}
```

---

### 2.6 Customer Behavioural Baselines

#### `GET /v1/baselines/{userId}`
Retrieves customer baseline profile metrics (typical transfer ranges, habitual recipients, authorized devices).

**Response `200 OK`**:
```json
{
  "user_id": "user_001",
  "typical_amount_min": 10000.0,
  "typical_amount_max": 75000.0,
  "typical_amount_mean": 35000.0,
  "typical_amount_std_dev": 12500.0,
  "total_transfers": 48,
  "typical_start_hour": 8,
  "typical_end_hour": 21,
  "known_beneficiaries": {
    "ben_mother": {
      "beneficiary_id": "ben_mother",
      "name": "Amina Adeleke",
      "account_number": "0123456789",
      "bank_name": "First Bank of Nigeria",
      "transfer_count": 24,
      "total_sent": 480000.0
    }
  },
  "known_devices": {
    "device_primary": {
      "device_id": "device_primary",
      "device_model": "Samsung Galaxy S22",
      "use_count": 320
    }
  }
}
```

---

### 2.7 State Reset

#### `POST /v1/reset`
Resets demo sessions, baseline stores, active probes, and intent caches so judges or presenters can restart demonstrations cleanly.

**Response `200 OK`**:
```json
{
  "status": "ok",
  "message": "Demo state reset successfully"
}
```

---

### 2.8 Evaluation Audit Report

#### `GET /v1/evaluation`
Runs model evaluation on demand against synthetic datasets and returns classification metrics, confusion matrices, and false positive rates.

**Response `200 OK`**:
```json
{
  "total_sessions": 175,
  "overall_accuracy": 1.0,
  "legitimate_false_positive_rate": 0.0,
  "ato_catch_rate": 1.0,
  "soc_eng_catch_rate": 1.0,
  "metrics_by_class": {
    "legitimate": { "precision": 1.0, "recall": 1.0, "f1": 1.0, "false_positive_rate": 0.0 },
    "account_takeover": { "precision": 1.0, "recall": 1.0, "f1": 1.0, "false_positive_rate": 0.0 },
    "social_engineering": { "precision": 1.0, "recall": 1.0, "f1": 1.0, "false_positive_rate": 0.0 },
    "accidental": { "precision": 1.0, "recall": 1.0, "f1": 1.0, "false_positive_rate": 0.0 }
  }
}
```

---

### 2.9 Live Event Stream (Server-Sent Events)

#### `GET /v1/stream` (optional `?session_id=...`)
Delivers real-time server-sent events for all ingested events.

**Browser Usage**:
```typescript
const es = new EventSource('/v1/stream?session_id=sess_demo_001');
es.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Received event:', data.type, data.seq);
};
```

---

## 3. TypeScript Type Definitions

Fiope can copy these definitions directly into `src/api.ts`:

```typescript
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
  | 'INTENT_PROBE_RESPONSE';

export type PolicyAction = 'ALLOW' | 'VERIFY' | 'PROBE' | 'BLOCK' | 'ESCALATE';

export interface ParallaxEvent {
  event_id?: string;
  session_id: string;
  user_id: string;
  type: EventType;
  timestamp?: number;
  metadata?: Record<string, unknown>;
  seq?: number;
  ingested_at_nanos?: number;
}

export interface ProbeOption {
  id: string;
  text: string;
  category?: string;
}

export interface ProbePayload {
  probe_id: string;
  prompt: string;
  options: ProbeOption[];
  allow_freeform: boolean;
  completed: boolean;
  response?: string;
}

export interface IntentResponse {
  session_id: string;
  hypotheses: Record<string, number>;
  uncertainty: number;
  action: PolicyAction;
  evidence: string[];
  events_seen: number;
  explanation?: string;
  confidence?: string;
  risk_score?: number;
  dominant_intent?: string;
  probe?: ProbePayload;
}

export interface ScenarioInfo {
  id: string;
  title: string;
  description: string;
  expected_action: string;
}

export interface ScenarioRunResult {
  scenario: string;
  session_id: string;
  events_ingested: number;
  intent: IntentResponse;
}
```

---

## 4. Suggested UI Flow for the 3 Primary Views

1. **Live Session Timeline**:
   - Subscribe to `GET /v1/stream?session_id=...` or poll session events.
   - Display ordered events with icons and timestamps.
2. **Intent State Gauge / Distribution**:
   - Render horizontal or radial progress bars for the 4 classes:
     - Legitimate (`#10B981` / Emerald)
     - Accidental (`#F59E0B` / Amber)
     - Social Engineering (`#EF4444` / Red)
     - Account Takeover (`#7C3AED` / Purple)
   - Show numeric Uncertainty (`0.00` to `1.00`) and Risk Score (`0.00` to `1.00`).
3. **Decision & Explanation Box**:
   - Prominently show `action` badge (`ALLOW`, `VERIFY`, `PROBE`, `BLOCK`, `ESCALATE`).
   - Display `explanation` narrative text.
   - Render bulleted `evidence` items.
   - If `action === "PROBE"`, render the interactive probe questions with clickable option buttons or freeform text box. Clicking an option dispatches `POST /v1/probes/respond` and updates the view immediately.
4. **Scenario Controls Panel**:
   - Display buttons for Scenarios A, B, C, D, and E.
   - Clicking a button calls `POST /v1/scenarios/:id/run` and populates the dashboard in real-time.
   - A `Reset` button triggers `POST /v1/reset` to restore a clean slate.
