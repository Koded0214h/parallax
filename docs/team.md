# Team Split

Four functional responsibilities. The two systems engineers agree on a stable event
schema (`pkg/contracts`) early so their components proceed independently.

---

## koded — Systems Engineer: Event & Runtime Infrastructure

Owns the pipeline that everything else runs on.

- Go service architecture
- Event ingestion and the Event Gateway (validate, timestamp, session IDs, normalize)
- Event streaming / stream layer
- Concurrency: goroutines, channels, bounded worker pools
- Session management and in-memory state
- Performance, latency, and throughput benchmarking
- Offline event buffering and reconnect synchronization

**Primary deliverable:** a high-performance real-time event pipeline.

---

## fluxx — Systems Engineer: Intelligence & Decision Engine

Owns the actual intent inference.

- Customer behavioural baselines
- Feature extraction (deterministic, independently testable)
- Sequence analysis
- Intent hypothesis generation (the intent distribution)
- Uncertainty engine (conflicting / insufficient / ambiguous evidence)
- Intent probe service
- Policy engine (`ALLOW / VERIFY / PROBE / BLOCK / ESCALATE`)
- Explanation engine
- Evaluation framework and metrics

**Primary deliverable:** the intent-inference engine.

---

## fiopefoluwa — Frontend Engineer: Observability & Demonstration

Builds a thin but visually compelling window into the infrastructure. Does **not**
duplicate decision logic.

- Live session view (streaming event timeline)
- Intent distribution view (updates in real time)
- Evidence / system-state view and decision explanation
- Scenario controls
- Demo dashboard

**Primary deliverable:** a thin but visually compelling window into the infrastructure.

Three primary views:

1. **Live Session** — timestamped event stream
2. **Intent State** — the four-class distribution, live
3. **Decision Explanation** — action, evidence bullets, confidence

---

## Quadri — Pitch / Product Engineer

Makes the technical system understandable and memorable. Works closely with the systems
team so every claim in the pitch is demonstrable.

- Narrative and user problem framing
- Research findings
- Demonstration script (the 8-step demo flow)
- Architecture storytelling
- Metric presentation
- Preparing for judge questions
- Limitations and non-goals
- Product positioning

**Primary deliverable:** make the technical system understandable and memorable.

---

## Interfaces between roles

| Between | Contract |
| --- | --- |
| koded ↔ fluxx | Stable event schema in `pkg/contracts`, agreed early |
| fluxx → fiopefoluwa | Intent/decision JSON shape (`hypotheses`, `uncertainty`, `action`, `evidence`) |
| koded → fiopefoluwa | Frontend event stream (WebSocket) format |
| systems → Quadri | Live benchmarks, evaluation metrics, working scenarios to cite in the pitch |
