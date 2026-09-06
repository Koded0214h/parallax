# Parallax

**A real-time, Go-based intent-inference layer for digital payments.**

Parallax sits between a banking application and its transaction authorization layer and
answers a question traditional fraud systems don't:

> Given everything we can observe about this session, **what is the user most likely trying to do?**

A valid PIN, a valid OTP, a known device and a successful authorization can still be a
legitimate transfer, an accidental one, a socially engineered one, or an account takeover.
Parallax treats **intent as a latent variable** and derives competing intent hypotheses
from behavioural, transactional, temporal and contextual evidence.

When the evidence isn't strong enough to decide, Parallax doesn't block. It runs an
**intent probe**, folds the new evidence back in, and re-evaluates.

```
evidence → inference → uncertainty → proportional action
```

instead of

```
transaction → black-box score → block
```

---

## Why it exists

Financial systems see the mechanics of a transaction — account, device, amount,
beneficiary, timing, interaction — but not the human intent behind it. Current fraud
approaches collapse every ambiguous situation into `FRAUD / NOT FRAUD`.

Parallax instead infers *which explanation best fits the evidence*, *how uncertain that
inference is*, and *what action is proportional* to both.

**Track:** Financial Services & Digital Payments
**Challenge:** A1 — Spotting Account Takeover From Behaviour
**Status:** Hackathon prototype (synthetic data, self-contained)

---

## Core concept

Five stages, with clear boundaries between them:

| Stage | Question it answers |
| --- | --- |
| **Observe** | What happened? (session + transaction events) |
| **Structure** | What measurable evidence does that create? |
| **Infer** | Which intent hypotheses fit the evidence? |
| **Reduce uncertainty** | Are we confident enough to act? If not, probe. |
| **Decide** | What should the system do, and how do we explain it? |

### Intent model

The engine outputs a distribution, not a label:

```
LEGITIMATE          18%
ACCIDENTAL           7%
SOCIAL_ENGINEERING  68%
ACCOUNT_TAKEOVER     7%
```

Plus an `UNKNOWN` state when the evidence doesn't support a reliable classification.
The output is not "Fraud = 83%" — it's "of the plausible explanations, social engineering
currently has the strongest evidence."

### Risk ≠ Uncertainty

Parallax keeps two ideas separate:

- **Risk** — how concerning the observed activity is.
- **Intent uncertainty** — how confident we are about the *explanation* for it.

High risk + low uncertainty → strong intervention.
Moderate risk + high uncertainty → **ask**, don't block.

---

## Decision policy

```
LOW RISK      + HIGH CONFIDENCE   → ALLOW
MODERATE RISK + HIGH UNCERTAINTY  → INTENT PROBE
HIGH RISK     + STRONG EVIDENCE   → BLOCK / ESCALATE
CONFLICTING EVIDENCE              → VERIFY / ESCALATE
```

Every decision carries: **Action · Reason · Evidence · Confidence**.

Explanations are generated from the same structured evidence that drove the decision, so
they stay tied to observable facts:

> "This transaction differs from your normal behaviour because the recipient is new, the
> amount is substantially higher than your normal transfers, and the recipient was created
> immediately before the transfer."

---

## Where AI fits

AI is **not** the fraud classifier. The authorization decision is owned by the engineered
inference and policy layers and never depends on a synchronous LLM call.

The AI component operates *on structured evidence the system already produced* — evidence
synthesis, hypothesis interpretation, human-readable explanations, spotting missing
context. It reasons over existing evidence rather than inventing new evidence, and it can
run asynchronously without blocking the payment path.

---

## Architecture

```
                    ┌─────────────────┐
                    │ Event Producers │
                    └────────┬────────┘
                             ▼
                    ┌─────────────────┐
                    │ Event Gateway   │  HTTP / WS
                    └────────┬────────┘
                             ▼
                    ┌─────────────────┐
                    │ Event Bus /     │
                    │ Stream Layer    │
                    └────────┬────────┘
            ┌────────────────┼────────────────┐
            ▼                ▼                ▼
      Feature Worker   Sequence Worker   Context Worker
            └────────────────┼────────────────┘
                             ▼
                    ┌─────────────────┐
                    │ Intent Engine   │
                    └────────┬────────┘
                    ┌────────┴────────┐
                    ▼                 ▼
              Policy Engine     Explanation Layer
                    │                 │
                    ▼                 ▼
                Decision          Frontend Stream
```

Layer boundaries are enforced — no layer silently does another's job:

```
EVENT LAYER        → "What happened?"
FEATURE LAYER      → "What measurable evidence does that create?"
INFERENCE LAYER    → "What intent hypotheses fit the evidence?"
UNCERTAINTY LAYER  → "How confident are we?"
POLICY LAYER       → "What should the system do?"
EXPLANATION LAYER  → "How do we explain that decision?"
```

### Components

| Component | Responsibility |
| --- | --- |
| Event Gateway | Validate, timestamp, assign session IDs, normalize, publish |
| Feature Engine | Deterministic transform of raw events into measurable features |
| Behaviour Engine | Compare current behaviour to the customer's historical baseline |
| Sequence Engine | Evaluate the order and timing of events as contextual evidence |
| Intent Engine | Combine evidence into competing intent hypotheses |
| Uncertainty Engine | Detect conflicting / insufficient / ambiguous evidence |
| Intent Probe Service | Ask targeted context questions; the probe becomes evidence |
| Policy Engine | Map inference to `ALLOW / VERIFY / PROBE / BLOCK / ESCALATE` |
| Explanation Engine | Concise explanation built from structured evidence |
| Frontend Gateway | Stream events, intent distribution, evidence, actions, explanations |

---

## API sketch

Ingest events:

```http
POST /v1/events
```

```json
{
  "session_id": "sess_123",
  "user_id": "user_001",
  "type": "BENEFICIARY_CREATED",
  "timestamp": 1788594121,
  "metadata": { "beneficiary_id": "ben_93" }
}
```

Query intent:

```http
GET /v1/sessions/{session_id}/intent
```

```json
{
  "hypotheses": {
    "legitimate": 0.18,
    "accidental": 0.07,
    "social_engineering": 0.68,
    "account_takeover": 0.07
  },
  "uncertainty": 0.31,
  "action": "PROBE",
  "evidence": ["new_beneficiary", "amount_above_baseline", "abnormal_sequence"]
}
```

---

## Performance targets

Core decision path (deterministic / statistical inference):

```
p50 < 50 ms
p95 < 150 ms
p99 < 300 ms
```

Sub-second end to end, matching the challenge requirement. Optional AI-generated
explanations run asynchronously and are never a dependency for authorization.

Built in Go for concurrent event ingestion, bounded worker pools, lock-efficient state,
in-memory feature processing and minimal hot-path allocation — and **benchmarked**, not
just described as fast (`events/sec`, `p50/p95/p99`, CPU, memory).

---

## Resilience

Nigeria's connectivity environment is a first-class constraint. The core inference path
runs without external cloud services.

| Failure | Behaviour |
| --- | --- |
| AI unavailable | Deterministic / statistical decision path |
| Database unavailable | In-memory state, buffer persistence |
| Missing behavioural telemetry | Fall back to transaction, sequence, contextual evidence |
| Unknown intent | Return `UNKNOWN / INSUFFICIENT EVIDENCE`, don't invent confidence |
| Offline | Baseline comparison, sequence analysis, core scoring, policy, local event buffering all continue; sync on reconnect |

---

## Demo scenarios

| Scenario | Setup | Expected |
| --- | --- | --- |
| **A — Legitimate** | Known device, known beneficiary, normal amount / time / interaction | `LEGITIMATE`, high confidence → **ALLOW** |
| **B — Account takeover** | New device, credential change, new beneficiary, rapid account changes, large transfer | `ACCOUNT_TAKEOVER` dominant → **BLOCK / ESCALATE** |
| **C — Social engineering** | Known device, valid auth, new beneficiary, large amount, abnormal interaction | High uncertainty → **intent probe** → user says "the bank told me to transfer the money so they can secure my account" → `SOCIAL_ENGINEERING` dominant → **PAUSE / INTERVENE** |
| **D — Accidental (optional)** | Known device, correct credentials, legitimate beneficiary, unusual amount, atypical sequence | High anomaly ≠ high-confidence takeover — demonstrates the false-positive philosophy |

Scenario C is the centrepiece: the transaction is *fully authenticated* and still wrong.

---

## Repository structure

```
parallax/
├── cmd/
│   ├── gateway/
│   ├── engine/
│   └── simulator/
├── internal/
│   ├── events/      features/     baseline/
│   ├── sequence/    intent/       uncertainty/
│   ├── policy/      probes/       explanation/
│   └── storage/
├── pkg/
│   └── contracts/          # shared event schema
├── simulator/
│   ├── users/  scenarios/  generators/
├── benchmarks/
├── evaluation/
├── frontend/
└── docs/                   # team split, planning
```

---

## Development

Requires Go 1.26+ and Node 20+.

```bash
# terminal 1 — Go gateway on :8080
make dev-backend

# terminal 2 — Vite dev server on :5173 (proxies /v1 and /healthz to :8080)
make dev-frontend
```

`make build` compiles both; `make test` runs the Go tests.

Current state is **base scaffolding only**: the gateway exposes `/healthz`,
`POST /v1/events` (in-memory buffer) and `GET /v1/sessions/{id}/intent` (returns a
placeholder uniform distribution). The frontend renders gateway health and that intent
payload. The feature / sequence / intent / uncertainty / policy engines are not wired yet.

```
backend/
├── cmd/gateway/            # HTTP entrypoint, graceful shutdown
└── internal/httpapi/       # router + request/response contracts
frontend/
└── src/
    ├── api.ts              # typed gateway client
    └── App.tsx             # health + intent view
```

---

## Evaluation

Target synthetic dataset: ~10,000 legitimate sessions, ~500 each of account takeover,
social engineering and accidental — with **realistic overlap** between legitimate and
malicious behaviour so the model can't cheat on obvious labels.

Metrics: precision / recall / F1 / confusion matrix; **false positive rate** (a primary
metric — blocking a real transfer has a real customer cost); intent-probe lift;
calibration; latency p50/p95/p99; throughput; resilience under missing telemetry,
AI outage, persistence outage and network loss.

---

## Team

See [`docs/team.md`](docs/team.md) for the full split.

| Person | Role |
| --- | --- |
| **koded** | Systems Engineer — Event & Runtime Infrastructure |
| **fluxx** | Systems Engineer — Intelligence & Decision Engine |
| **fiopefoluwa** | Frontend Engineer — Observability & Demonstration |
| **Quadri** | Pitch / Product Engineer — narrative, demo script, positioning |

---

## Thesis

> **A transaction can be authenticated without being intentional.**

Parallax treats a payment as the endpoint of a human decision process, not an isolated
event. When evidence is strong: **act**. When it's weak: **ask**. When it's
contradictory: **don't pretend to know**. When it intervenes: **explain why**.
