# Backend — Event & Runtime Infrastructure (koded)

The runtime side of Parallax: get events in, normalize them, keep session state,
fan them out to processors, survive network loss, and stay fast. The intelligence
engine (fluxx) consumes what this layer produces and never reaches around it.

## Module map

| # | Package | Responsibility | Status |
|---|---------|---------------|--------|
| M1 | `pkg/contracts` | Stable schema: `Event`, `EventType`, `IntentResponse`, `Action`. The agreed contract with fluxx & frontend. | **done** |
| M2 | `internal/ingest` | Validate structure, canonicalize type, generate `EventID`, stamp `Timestamp` / `IngestedAtNanos`. | **done** |
| M3 | `internal/session` | Sharded in-memory session store: ordered trajectory, per-session `Seq`, lifecycle metadata. | **done** |
| M4 | `internal/stream` | In-process event bus: fan-out to workers + frontend subscribers, bounded buffers, backpressure policy. | todo |
| M5 | `internal/worker` | Bounded worker pool consuming the stream; `Processor` interface fluxx implements (feature / sequence / context). | todo |
| M6 | `internal/httpapi` | HTTP transport: `POST /v1/events`, `GET /v1/sessions/{id}/intent`, `GET /healthz`. WS `/v1/stream` pending. | **partial** |
| M7 | `internal/wal` | Offline event buffer + reconnect replay into storage (prd.md §28). | todo |
| M8 | `internal/storage` | Persistence interface + in-memory and SQLite implementations. | todo |
| M9 | `bench/` | Load generator + latency/throughput harness: events/sec, p50/p95/p99, CPU, mem. | todo |

## Pipeline

```
POST /v1/events
      │
      ▼
 ingest.Normalizer.Normalize   ── validates, stamps, canonicalizes
      │
      ▼
 session.Store.Append          ── assigns per-session Seq, appends to trajectory
      │
      ▼
 stream.Bus.Publish (M4)       ── fan-out
      │
      ├──────────────┬──────────────┐
      ▼              ▼              ▼
 worker pool (M5)  frontend WS   wal (M7)
      │
      ▼
 Processor (fluxx: feature / sequence / context / intent)
```

## Contract notes for fluxx

- `contracts.Event` server-owned fields (`Seq`, `IngestedAtNanos`) are guaranteed
  set by the time an event reaches a `Processor`. `EventID` and `Timestamp` are
  always non-zero post-ingest.
- `Seq` is 1-based and monotonic **per session**. Use it for ordering, not the
  wire `Timestamp` (second precision). `IngestedAtNanos` is for sub-second gaps.
- `session.Store.Trajectory(id)` returns a copy — safe to read without locks.
- `IntentResponse` is yours to fill; the runtime only transports it. `EventsSeen`
  is set by the runtime.

## Next up

1. **M4 `stream`** — `Bus` with `Publish(Event)` and `Subscribe() <-chan Event`,
   bounded per-subscriber buffers, drop-oldest or block policy (config).
2. **M5 `worker`** — `Pool` of N goroutines draining the bus into a `Processor`;
   graceful drain on shutdown; per-event latency histogram hook.
3. Wire both into `cmd/gateway` and add `WS /v1/stream` for the frontend.

## Running

```bash
make dev-backend           # :8080
make test                  # go test ./...
cd backend && go test ./... -race
```
