# Parallax

## Product Requirements Document

**Track:** Financial Services & Digital Payments
**Challenge:** A1 — Spotting Account Takeover From Behaviour
**Implementation Language:** Go
**Status:** Hackathon Prototype

---

# 1. Product Overview

**Parallax** is a real-time financial security and intent-inference engine designed to determine whether an authenticated financial action is consistent with the user's likely intent.

Traditional fraud systems primarily answer:

> **“Does this transaction look suspicious?”**

Parallax asks a deeper question:

> **“Given everything we can observe about this session, what is the user most likely trying to do?”**

The system does not assume that authentication equals intent.

A valid PIN, OTP, known device, and successful transaction can still represent:

* a legitimate transaction;
* an accidental transaction;
* a socially engineered transaction;
* an account takeover;
* or another ambiguous situation.

Parallax therefore treats **intent as a latent variable** and derives competing intent hypotheses from structured behavioural, transactional, temporal, and contextual evidence.

When the evidence is insufficient to confidently determine intent, Parallax does not immediately block the transaction. Instead, it can request additional information through an **intent probe**, incorporate the new evidence, and reevaluate the decision.

The result is a real-time decision system that prioritizes:

**evidence → inference → uncertainty → proportional action**

rather than:

**transaction → black-box score → block**

---

# 2. Problem Statement

Financial systems have extensive visibility into the mechanics of a transaction but limited visibility into the human intent behind it.

A banking system can know:

* which account authenticated;
* which device was used;
* what transaction was attempted;
* how much money was transferred;
* which beneficiary received it;
* when it occurred;
* how the user interacted with the interface.

It cannot directly observe:

> **why the user decided to perform that action.**

This produces a critical ambiguity.

The following transactions may look technically similar:

```text
Correct credentials
Correct OTP
Valid session
Successful authorization
```

Yet their underlying intent may be completely different:

```text
                    AUTHENTICATED ACTION
                           │
        ┌──────────────────┼──────────────────┐
        ▼                  ▼                  ▼
    Legitimate         Accidental       Manipulated
                                           │
                                    ┌──────┴──────┐
                                    ▼             ▼
                             Social engineering  Coercion

                           OR

                       Account takeover
```

Current fraud approaches often collapse these situations into a single binary outcome:

```text
FRAUD / NOT FRAUD
```

Parallax instead attempts to infer **which explanation best fits the evidence**, how uncertain that inference is, and what action is appropriate.

---

# 3. Product Vision

Parallax is intended to become an **intent-aware security layer for digital financial infrastructure**.

The system should eventually operate between the financial application and its transaction authorization layer:

```text
Customer
   │
   ▼
Banking Application
   │
   ▼
┌────────────────────────────┐
│         PARALLAX           │
│                            │
│ Evidence → Inference       │
│            ↓               │
│        Uncertainty         │
│            ↓               │
│      Decision Policy       │
└──────────────┬─────────────┘
               │
        ┌──────┼──────┐
        ▼      ▼      ▼
      Allow  Verify  Block
```

The hackathon implementation will be a synthetic, self-contained demonstration of this architecture.

---

# 4. Goals

## 4.1 Primary Goal

Build a working real-time engine that can distinguish between legitimate and abnormal financial activity by reasoning about **probable user intent**, rather than relying on transaction amount or a single behavioural anomaly score.

## 4.2 Secondary Goals

Demonstrate that the system can:

1. Maintain behavioural baselines for synthetic customers.
2. Process financial session events in real time.
3. Analyse transaction and session sequences.
4. Generate competing intent hypotheses.
5. Explicitly represent uncertainty.
6. Request additional intent evidence when uncertainty is high.
7. Recalculate intent after additional evidence is provided.
8. Produce a human-readable explanation for every decision.
9. Operate within sub-second latency for live transaction scoring.
10. Remain useful when some behavioural signals are unavailable.

---

# 5. Non-Goals

The hackathon prototype will **not** attempt to:

* integrate with real Nigerian banks;
* use real customer financial data;
* establish a production-grade banking core;
* guarantee that intent can be directly observed;
* perform psychological diagnosis;
* identify or profile real individuals;
* replace bank fraud operations teams;
* build a full consumer banking application;
* solve every type of financial crime;
* create a universal fraud model for every financial institution.

The system is a prototype demonstrating a new decision architecture.

---

# 6. Core Product Concept

Parallax is based on five stages.

```text
1. OBSERVE
      ↓
2. STRUCTURE
      ↓
3. INFER
      ↓
4. REDUCE UNCERTAINTY
      ↓
5. DECIDE
```

### Observe

Consume session and transaction events.

### Structure

Convert raw events into measurable evidence.

### Infer

Generate competing intent hypotheses.

### Reduce Uncertainty

When evidence is ambiguous, request additional contextual information.

### Decide

Apply a policy according to risk, intent confidence, and uncertainty.

---

# 7. Intent Model

The prototype will model four primary intent classes.

```text
LEGITIMATE
ACCIDENTAL
SOCIAL_ENGINEERING
ACCOUNT_TAKEOVER
```

An additional:

```text
UNKNOWN
```

state exists when the available evidence does not support a sufficiently reliable classification.

The engine produces a distribution rather than a single categorical label.

Example:

```text
Legitimate            18%
Accidental              7%
Social Engineering     68%
Account Takeover        7%
```

The output is therefore not:

> “Fraud = 83%.”

It is:

> “Of the plausible explanations for this activity, social engineering currently has the strongest evidence.”

---

# 8. Key Design Principle: Risk ≠ Uncertainty

Parallax maintains two separate concepts.

## Risk

How concerning the observed activity is.

## Intent uncertainty

How confident the system is about the explanation for that activity.

These are different.

Example:

```text
Case A

Risk:               High
Intent uncertainty: Low

→ Strong intervention
```

Another:

```text
Case B

Risk:               Moderate
Intent uncertainty: High

→ Request additional evidence
```

This distinction is central to Parallax.

A transaction should not automatically be blocked simply because it is unusual.

---

# 9. Evidence Model

Raw events are converted into structured evidence before inference.

Example:

```json
{
  "amount_deviation": 7.4,
  "new_beneficiary": true,
  "beneficiary_age_seconds": 19,
  "device_known": true,
  "credential_change_recent": false,
  "time_deviation": 3.1,
  "interaction_speed_deviation": 2.4,
  "recipient_seen_before": false
}
```

The engine transforms those values into interpretable evidence such as:

```text
• Transfer amount is 7.4× the customer's normal range.
• Recipient has never been used before.
• Recipient was created 19 seconds before payment.
• Device is recognized.
• Interaction occurred at an unusual time.
```

This evidence layer becomes the foundation for downstream inference.

---

# 10. Event Model

Every user interaction relevant to the transaction is represented as an event.

Example event types:

```text
LOGIN
LOGOUT
DEVICE_SEEN
DEVICE_CHANGED
PASSWORD_CHANGED
PIN_CHANGED
BENEFICIARY_CREATED
BENEFICIARY_MODIFIED
TRANSFER_STARTED
AMOUNT_ENTERED
OTP_REQUESTED
OTP_VERIFIED
TRANSFER_COMPLETED
TRANSFER_FAILED
INTENT_PROBE_STARTED
INTENT_PROBE_RESPONSE
```

A transaction is therefore represented as an ordered session trajectory.

Example:

```text
LOGIN
  ↓
DEVICE_CHANGED
  ↓
PASSWORD_CHANGED
  ↓
BENEFICIARY_CREATED
  ↓
AMOUNT_ENTERED
  ↓
OTP_VERIFIED
  ↓
TRANSFER
```

Rather than examining only the final transfer, Parallax considers the preceding sequence.

---

# 11. Customer Behavioural Baseline

Each synthetic customer receives a historical behavioural profile.

Example:

```text
Customer: A001

Typical transaction:
₦10,000 – ₦75,000

Typical transaction time:
08:00 – 18:00

Known beneficiaries:
6

Primary device:
Android / Device-A

Typical session duration:
25 – 50 seconds

Typical interaction rate:
Normal
```

The baseline is continuously updated as additional historical transactions occur.

The objective is not to define a universal “normal user.”

It is to determine:

> **What is normal for this particular customer?**

---

# 12. Behavioural Features

The prototype will consider features such as:

### Transaction

* amount;
* amount deviation from baseline;
* recipient history;
* transaction frequency;
* transaction velocity.

### Device

* known vs unknown device;
* recent device change;
* device consistency.

### Session

* session duration;
* navigation sequence;
* event ordering;
* interaction speed;
* failed attempts;
* retries.

### Timing

* time-of-day deviation;
* day-of-week deviation;
* transaction velocity.

### Account state

* password changes;
* PIN changes;
* beneficiary creation;
* profile changes;
* recent authentication events.

### Channel

* mobile application;
* browser;
* USSD simulation.

Feature availability will depend on the simulated channel.

---

# 13. Intent Inference Engine

The engine combines evidence from multiple analysis components.

Conceptually:

```text
                 Evidence
                    │
      ┌─────────────┼─────────────┐
      ▼             ▼             ▼
 Behaviour      Sequence       Context
 Model           Model          Model
      │             │             │
      └─────────────┼─────────────┘
                    ▼
            Intent Inference
                    │
          ┌─────────┴─────────┐
          ▼                   ▼
    Intent Distribution    Uncertainty
```

The prototype may combine:

* deterministic rules;
* statistical anomaly detection;
* historical baselines;
* sequence scoring;
* calibrated classification;
* contextual evidence.

The precise model may evolve during implementation.

The architecture should allow individual inference components to be independently evaluated.

---

# 14. AI Component

AI is **not** the primary fraud classifier.

The system should never fundamentally depend on:

```text
Raw Transaction JSON
        ↓
       LLM
        ↓
    Risk Score
```

Instead, AI operates on structured evidence generated by the system.

The AI component may perform:

* evidence synthesis;
* hypothesis interpretation;
* human-readable explanations;
* identification of missing context;
* constrained reasoning across known evidence.

Example input:

```json
{
  "hypotheses": {
    "legitimate": 0.21,
    "accidental": 0.08,
    "social_engineering": 0.61,
    "account_takeover": 0.10
  },
  "evidence": [
    "beneficiary_created_19_seconds_before_transfer",
    "amount_7.4x_user_baseline",
    "device_known",
    "interaction_speed_above_baseline"
  ]
}
```

The AI should reason over existing evidence rather than invent new evidence.

The numerical decision remains owned by the engineered inference and policy layers.

---

# 15. Intent Probes

When uncertainty is high, Parallax can ask the user for additional context.

The objective is not to ask:

> “Is this transaction legitimate?”

because such a question provides weak information.

Instead, the system seeks information about the user's understanding of the transaction.

Example:

> **What is this payment for?**

Potential response:

> “The bank told me to send it here to reverse a fraudulent transaction.”

That response can become additional evidence.

The flow becomes:

```text
Observed Evidence
       ↓
Intent Hypotheses
       ↓
High Uncertainty
       ↓
Intent Probe
       ↓
User Context
       ↓
Updated Evidence
       ↓
Updated Intent
       ↓
Decision
```

This introduces an important system behaviour:

> **The system can actively reduce uncertainty instead of merely observing passively.**

---

# 16. Decision Policy

The policy engine maps inference results to an action.

Initial prototype policy:

```text
LOW RISK + HIGH CONFIDENCE
→ ALLOW

MODERATE RISK + HIGH UNCERTAINTY
→ INTENT PROBE

HIGH RISK + STRONG EVIDENCE
→ BLOCK / ESCALATE

CONFLICTING EVIDENCE
→ VERIFY / ESCALATE
```

Each decision must include:

```text
Action
Reason
Evidence
Confidence
```

Example:

```text
ACTION:
Additional verification required

WHY:
• New beneficiary
• Transfer amount is 7.4× baseline
• Transaction sequence is abnormal
• User context indicates possible bank impersonation

PRIMARY HYPOTHESIS:
Social engineering — 91%
```

---

# 17. Explanation Engine

Every decision should be explainable without exposing internal model complexity.

Bad:

> “Model score exceeded 0.81.”

Good:

> “This transaction differs significantly from your normal behaviour because the recipient is new, the amount is substantially higher than your normal transfers, and the recipient was created immediately before the transfer.”

For the hackathon, explanations will be generated from the same structured evidence that powers the decision.

This keeps explanations tied to observable facts.

---

# 18. Real-Time Requirements

The system is explicitly designed around low latency.

### Target

The core decision path should operate in **under one second**, matching the challenge requirement.

Preferred target:

```text
p50 < 50 ms
p95 < 150 ms
p99 < 300 ms
```

for the deterministic/statistical inference path.

Any optional AI-generated explanation may operate asynchronously where appropriate and must not become a dependency for the core authorization decision.

This distinction is important.

The payment decision path should remain functional even if an external AI service is slow or unavailable.

---

# 19. Throughput Requirements

The prototype will be implemented in Go to demonstrate that the inference layer can operate as high-throughput infrastructure rather than as a heavyweight synchronous application.

The system should be designed around:

* concurrent event ingestion;
* bounded worker pools;
* lock-efficient state access;
* in-memory feature processing;
* efficient serialization;
* streaming event processing;
* minimal allocations on the hot path.

The system should be benchmarked rather than merely described as “fast.”

Example benchmark output:

```text
Events processed/sec
p50 latency
p95 latency
p99 latency
CPU usage
Memory usage
```

Exact benchmark targets will depend on the final implementation and available hardware.

---

# 20. Proposed Go Architecture

```text
                    ┌─────────────────┐
                    │ Event Producers  │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │ Event Gateway    │
                    │ HTTP / WS        │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │ Event Bus /     │
                    │ Stream Layer    │
                    └────────┬────────┘
                             │
            ┌────────────────┼────────────────┐
            ▼                ▼                ▼
      Feature Worker    Sequence Worker   Context Worker
            │                │                │
            └────────────────┼────────────────┘
                             ▼
                    ┌─────────────────┐
                    │ Intent Engine   │
                    └────────┬────────┘
                             │
                    ┌────────┴────────┐
                    ▼                 ▼
              Policy Engine     Explanation Layer
                    │                 │
                    ▼                 ▼
                Decision          Frontend Stream
```

A lightweight implementation can initially use:

* Go HTTP/WebSocket services;
* goroutines and channels;
* in-memory state;
* SQLite/PostgreSQL for persistence where needed;
* JSON or protobuf for event transport;
* an optional message-broker abstraction.

The architecture should avoid unnecessary infrastructure during the hackathon.

---

# 21. System Components

## 21.1 Event Gateway

Receives session events.

Responsibilities:

* validate event structure;
* attach timestamps;
* assign session identifiers;
* normalize event format;
* publish events to the processing layer.

---

## 21.2 Feature Engine

Transforms raw events into measurable features.

Example:

```text
TRANSFER amount = ₦450,000

Customer baseline = ₦60,000

amount_deviation = 7.5×
```

The feature engine should be deterministic and independently testable.

---

## 21.3 Behaviour Engine

Compares current behaviour against the user's historical baseline.

It should detect:

* unusual transaction amounts;
* unusual times;
* unusual interaction speed;
* unfamiliar devices;
* unusual transaction frequency.

---

## 21.4 Sequence Engine

Evaluates the order and timing of events.

Example:

```text
DEVICE_CHANGE
→ PASSWORD_CHANGE
→ BENEFICIARY_CREATE
→ LARGE_TRANSFER
```

The engine should treat sequences as contextual evidence rather than isolated events.

---

## 21.5 Intent Engine

Combines evidence into competing intent hypotheses.

Output:

```text
{
    legitimate: 0.18,
    accidental: 0.06,
    social_engineering: 0.71,
    account_takeover: 0.05
}
```

---

## 21.6 Uncertainty Engine

Determines whether the system has enough evidence to make a confident decision.

It should detect:

* conflicting evidence;
* insufficient evidence;
* ambiguous situations;
* low-confidence classifications.

---

## 21.7 Intent Probe Service

Triggers additional contextual questions when required.

The probe itself becomes part of the session evidence.

---

## 21.8 Policy Engine

Converts inference into action:

```text
ALLOW
VERIFY
PROBE
BLOCK
ESCALATE
```

The policy layer must be independent from the inference layer.

This allows policy thresholds to change without retraining models.

---

## 21.9 Explanation Engine

Generates a concise explanation based on structured evidence.

---

## 21.10 Frontend Gateway

Streams:

* session events;
* current intent distribution;
* evidence;
* uncertainty;
* policy actions;
* explanations.

The frontend is primarily an **observability and demonstration interface**.

---

# 22. Frontend Scope

The frontend developer should not build a full banking application.

The interface consists of three primary views.

## View 1 — Live Session

```text
09:41:02  LOGIN
09:41:06  DEVICE_CHANGE
09:41:11  BENEFICIARY_CREATE
09:41:24  AMOUNT_ENTER
09:41:38  TRANSFER
```

---

## View 2 — Intent State

```text
LEGITIMATE          18%
ACCIDENTAL           7%
SOCIAL ENGINEERING  68%
ACCOUNT TAKEOVER     7%
```

The distribution updates in real time.

---

## View 3 — Decision Explanation

```text
ACTION
Additional verification required

EVIDENCE
• New beneficiary
• Amount 7.4× baseline
• Unusual interaction sequence
• User-stated purpose indicates possible impersonation

CONFIDENCE
High
```

The UI should emphasize evidence and system state rather than visual complexity.

---

# 23. Demonstration Scenarios

The team will prepare a minimum of three scripted scenarios.

## Scenario A — Legitimate

```text
Known device
Known beneficiary
Normal amount
Normal time
Normal interaction
```

Expected:

```text
Legitimate → high confidence
Action → ALLOW
```

---

## Scenario B — Account Takeover

```text
New device
Credential change
New beneficiary
Rapid sequence of account changes
Large transfer
```

Expected:

```text
Account takeover → dominant hypothesis
Action → BLOCK / ESCALATE
```

---

## Scenario C — Social Engineering

```text
Known device
Valid authentication
New beneficiary
Large amount
Abnormal interaction
```

System initially produces:

```text
High uncertainty
```

Then an intent probe asks:

> What is this payment for?

Synthetic user responds:

> “The bank told me to transfer the money so they can secure my account.”

System updates:

```text
Social engineering → dominant hypothesis
Action → PAUSE / INTERVENE
```

This scenario is the centrepiece of the demonstration because the transaction is fully authenticated.

---

# 24. Optional Fourth Scenario — Accidental Transfer

A user performs a high-value transaction with:

* known device;
* correct credentials;
* legitimate beneficiary;
* unusual amount;
* atypical interaction sequence.

The system should distinguish:

```text
high anomaly
```

from:

```text
high confidence account takeover
```

and demonstrate why anomaly alone is insufficient.

This scenario helps demonstrate the project's false-positive philosophy.

---

# 25. Synthetic Data Generator

A dedicated Go or Python utility will generate training and evaluation data.

The generator should create:

```text
Users
Devices
Beneficiaries
Historical sessions
Transactions
Attack sessions
Accidental sessions
Social-engineering sessions
```

Each session should have a known ground-truth intent label.

Example:

```text
session_01482

ground_truth:
SOCIAL_ENGINEERING

events:
LOGIN
BENEFICIARY_CREATE
AMOUNT_ENTER
OTP_VERIFY
TRANSFER
INTENT_PROBE
USER_RESPONSE
```

Attack patterns must be reproducible so evaluation can be rerun.

---

# 26. Evaluation Dataset

A prototype target dataset could contain:

```text
10,000 legitimate sessions
500 account takeover sessions
500 social-engineering sessions
500 accidental transactions
```

The exact distribution may change depending on implementation constraints.

The critical requirement is that:

> **The dataset contains realistic overlap between legitimate and malicious behaviour.**

Otherwise the model can achieve misleadingly high performance by learning obvious attack labels.

---

# 27. Evaluation Metrics

## Fraud/Intent Classification

* Precision
* Recall
* F1
* Confusion matrix

## False Positive Performance

Measure legitimate transactions incorrectly interrupted.

```text
False Positive Rate
```

is a primary metric because blocking a legitimate transfer has a real customer cost.

## Intent Probe Performance

Measure:

> How often does additional user context improve classification?

## Calibration

Measure whether:

```text
80% confidence
```

actually corresponds roughly to an 80% empirical frequency over the relevant evaluation population.

## Latency

Measure:

* p50;
* p95;
* p99.

## Throughput

Measure:

```text
events/sec
transactions/sec
```

## Resilience

Measure behaviour when:

* behavioural telemetry is missing;
* AI service is unavailable;
* persistence is temporarily unavailable;
* network connectivity is interrupted.

---

# 28. Offline and Network Failure Behaviour

Nigeria's connectivity environment makes this important.

The core inference path must not depend on continuous access to external cloud services.

Therefore:

```text
Event
  ↓
Local processing
  ↓
Decision
```

must remain possible without an external AI call.

The prototype should demonstrate that at least the following continue operating offline:

* baseline comparison;
* sequence analysis;
* core intent scoring;
* policy decisions;
* local event buffering.

When connectivity returns:

```text
Buffered events
      ↓
Synchronization
      ↓
Persistent store
```

This also supports the challenge's requirement to consider power and network interruptions.

---

# 29. Fault Tolerance

The system should degrade gracefully.

### AI unavailable

Use deterministic/statistical decision path.

### Database unavailable

Use in-memory state where possible and buffer persistence.

### Missing behavioural telemetry

Fall back to transaction, sequence and contextual evidence.

### Unknown intent

Return:

```text
UNKNOWN / INSUFFICIENT EVIDENCE
```

rather than inventing confidence.

---

# 30. Security and Privacy

All financial data will be synthetic.

No real:

* account numbers;
* names;
* bank credentials;
* financial histories;
* phone numbers;
* biometric data

will be collected.

Synthetic customer identifiers will be used throughout the prototype.

The project should demonstrate privacy-by-design by processing only the evidence necessary for intent inference.

---

# 31. Team Structure

The team consists of four functional responsibilities.

## Systems Engineer 1 — Event & Runtime Infrastructure

Owner:

* Go service architecture;
* event ingestion;
* event streaming;
* concurrency;
* session management;
* performance;
* latency;
* throughput benchmarking.

Primary deliverable:

> **High-performance real-time event pipeline.**

---

## Systems Engineer 2 — Intelligence & Decision Engine

Owner:

* behavioural baselines;
* feature extraction;
* sequence analysis;
* intent hypotheses;
* uncertainty;
* intent probes;
* policy engine;
* evaluation framework.

Primary deliverable:

> **The actual intent-inference engine.**

The systems developers should agree on a stable event schema early so both components can proceed independently.

---

## Frontend Engineer — Observability & Demonstration

Owner:

* live session visualisation;
* intent distribution;
* evidence graph/state;
* explanations;
* scenario controls;
* demo dashboard.

Primary deliverable:

> **A thin but visually compelling window into the infrastructure.**

The frontend should not duplicate decision logic.

---

## Pitch / Product Engineer

Owner:

* narrative;
* user problem;
* research findings;
* demonstration script;
* architecture storytelling;
* metric presentation;
* judge questions;
* limitations;
* product positioning.

Primary deliverable:

> **Make the technical system understandable and memorable.**

This role should work closely with the systems team so every claim made during the pitch is demonstrable.

---

# 32. Engineering Boundaries

The architecture should enforce clear boundaries.

```text
EVENT LAYER
knows:
"What happened?"

FEATURE LAYER
knows:
"What measurable evidence does that create?"

INFERENCE LAYER
knows:
"What intent hypotheses fit the evidence?"

UNCERTAINTY LAYER
knows:
"How confident are we?"

POLICY LAYER
knows:
"What should the system do?"

EXPLANATION LAYER
knows:
"How do we explain that decision?"
```

No layer should silently perform another layer's responsibility.

---

# 33. API Concept

Example:

```http
POST /v1/events
```

```json
{
  "session_id": "sess_123",
  "user_id": "user_001",
  "type": "BENEFICIARY_CREATED",
  "timestamp": 1788594121,
  "metadata": {
    "beneficiary_id": "ben_93"
  }
}
```

Decision endpoint:

```http
GET /v1/sessions/{session_id}/intent
```

Response:

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
  "evidence": [
    "new_beneficiary",
    "amount_above_baseline",
    "abnormal_sequence"
  ]
}
```

---

# 34. Suggested Repository Structure

```text
parallax/
│
├── cmd/
│   ├── gateway/
│   ├── engine/
│   └── simulator/
│
├── internal/
│   ├── events/
│   ├── features/
│   ├── baseline/
│   ├── sequence/
│   ├── intent/
│   ├── uncertainty/
│   ├── policy/
│   ├── probes/
│   ├── explanation/
│   └── storage/
│
├── pkg/
│   └── contracts/
│
├── simulator/
│   ├── users/
│   ├── scenarios/
│   └── generators/
│
├── benchmarks/
│
├── evaluation/
│
└── frontend/
```

This structure intentionally keeps the inference system independent from the frontend.

---

# 35. Hackathon Demo Flow

The presentation should follow this order.

### Step 1 — Establish normal behaviour

Show a synthetic user performing several normal transactions.

Parallax learns the baseline.

### Step 2 — Perform normal transaction

Demonstrate:

```text
ALLOW
```

### Step 3 — Replay an account takeover

Show the event stream accumulating.

The intent distribution changes.

Parallax produces:

```text
ACCOUNT TAKEOVER
```

and blocks/escalates.

### Step 4 — Reset and replay social engineering

Use the same legitimate device and valid authentication.

The transaction initially remains ambiguous.

### Step 5 — Intent probe

The system asks the synthetic user what the payment is for.

The response introduces new evidence.

The intent distribution changes dramatically.

### Step 6 — Explain the difference

Show:

```text
Same:
- user
- device
- authentication
- payment mechanism

Different:
- sequence
- context
- interaction
- inferred intent
```

### Step 7 — Technical reveal

Show the Go architecture, event throughput, latency benchmarks and evaluation metrics.

### Step 8 — Failure case

Intentionally provide ambiguous behaviour and demonstrate:

```text
UNKNOWN / NEED MORE EVIDENCE
```

This proves that the system does not pretend to know everything.

---

# 36. Success Criteria

The prototype is considered successful if it can demonstrate all of the following:

### Functional

* Real-time event ingestion.
* Per-user behavioural baselines.
* Sequence analysis.
* Multiple intent hypotheses.
* Explicit uncertainty.
* Intent probing.
* Policy-based action.
* Human-readable explanations.

### Performance

* Sub-second core decision latency.
* Measured throughput.
* No dependency on synchronous LLM calls for authorization.

### Accuracy

* Detect account takeover patterns.
* Identify social-engineering scenarios.
* Preserve legitimate activity where possible.
* Demonstrate accidental-transfer ambiguity.
* Report false positives and false negatives honestly.

### Demonstration

* Live replay of transaction events.
* Visible evolution of intent hypotheses.
* Visible evidence behind the decision.
* Clear distinction between authentication and intent.

---

# 37. Product Thesis

Parallax is built around one fundamental observation:

> **A transaction can be authenticated without being intentional.**

The system therefore treats a financial transaction not as an isolated event but as the endpoint of a human decision process.

The engineering objective is not to “read” the user.

It is to construct enough evidence to distinguish plausible explanations for what the user is doing.

When evidence is strong:

> **Act.**

When evidence is weak:

> **Ask.**

When evidence is contradictory:

> **Do not pretend to know.**

When the system does intervene:

> **Explain why.**

That is the core product behaviour.

---

# 38. Final One-Line Definition

> **Parallax is a real-time Go-based intent-inference layer that uses behavioural, transactional, temporal and contextual evidence to distinguish legitimate financial actions from accidental, socially engineered and compromised activity — and knows when it needs more evidence before deciding.**

