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
| M4 | `internal/stream` | In-process event bus: hub goroutine, fan-out to workers + frontend + WAL, per-subscription bounded buffers with overflow policy, `Publish` backpressure. | **done** |
| M5 | `internal/worker` | Bounded worker pool draining a stream subscription; `Processor` interface fluxx implements; panic isolation, graceful/forced drain, latency reservoir (p50/p95/p99). | **done** |
| M6 | `internal/httpapi` + `cmd/gateway` | HTTP transport wired end to end: `POST /v1/events` → ingest → session → bus → pool; `GET /v1/stream` (SSE); `GET /v1/metrics`, `GET /healthz` (bus + pool counters); `GET /v1/sessions/{id}/intent` (placeholder). | **done** |
| M7 | `internal/wal` | Offline event buffer + reconnect replay into storage (prd.md §28). | **done** (fluxx) |
| M8 | `internal/storage` | Persistence interface + implementations. | **done** (fluxx) |
| M9 | `bench/` + `cmd/loadgen` | End-to-end load harness (`bench.Harness`) + CLI: events/sec, p50/p95/p99, alloc, GC. `make bench`. | **done** |

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
 stream.Bus.Publish (M4)       ── fan-out (blocks on bounded intake => backpressure)
      │
      ├────────────────────┬────────────────────┐
      ▼                    ▼                    ▼
 worker pool (M5)     GET /v1/stream        wal (M7, todo)
 Block sub            SSE, DropOldest sub
      │                    │
      ▼                    ▼
 Processor            frontend
 (fluxx: feature / sequence / context / intent)
```

`GET /v1/metrics` and `GET /healthz` return `stream.Bus.Stats()` and
`worker.Pool.Stats()` (throughput, drops, p50/p95/p99).

## Contract notes for fluxx

- `contracts.Event` server-owned fields (`Seq`, `IngestedAtNanos`) are guaranteed
  set by the time an event reaches a `Processor`. `EventID` and `Timestamp` are
  always non-zero post-ingest.
- `Seq` is 1-based and monotonic **per session**. Use it for ordering, not the
  wire `Timestamp` (second precision). `IngestedAtNanos` is for sub-second gaps.
- `session.Store.Trajectory(id)` returns a copy — safe to read without locks.
- `IntentResponse` is yours to fill; the runtime only transports it. `EventsSeen`
  is set by the runtime.

## stream.Bus notes (M4)

- One hub goroutine owns the subscriber set and every channel send, so callers
  never race a closed channel. Consumers only `for e := range sub.C()`.
- `Publish` blocks on a bounded intake queue → backpressure reaches the HTTP
  handler under load. `TryPublish` is the non-blocking variant.
- Per-subscription overflow policy: `DropOldest` (frontend — keep latest),
  `DropNewest`, or `Block`. Use `Block` for **at most one** subscription (the
  primary consumer); a stalled `Block` consumer stalls the hub for everyone —
  that is the deliberate end-to-end backpressure knob. A wedged `Block` sub can
  still be detached (`Unsubscribe`) or the bus closed without deadlock.
- Counters per sub: `Seen == Received + Dropped` by construction. Bus tracks
  `Published` and total `Dropped`.

## worker.Pool notes (M5)

- `Processor` — `Process(ctx, contracts.Event) error` — is the seam fluxx owns.
  Must be concurrency-safe and must respect `ctx` (cancelled on a forced stop).
  `ProcessorFunc` adapts a plain function.
- Drains a `Source` (`*stream.Subscription` satisfies it) with N workers,
  default `GOMAXPROCS`.
- Panic in a `Process` call is recovered per event: counted (`Panicked`,
  `Failed`), logged with stack, the worker keeps going.
- `Stop(ctx)` unsubscribes from the source then waits for in-flight work; if
  `ctx` fires first it cancels the processor context, stops the workers, and
  returns `ctx.Err()`. Idempotent.
- `Stats()`: `Workers`, `Processed`, `Failed`, `Panicked`, `InFlight`, and
  `Latency` (bounded reservoir → exact count/min/max, estimated p50/p95/p99).
  Source of the pitch's latency numbers.

## Next up

1. **M9 `bench/`** — load generator + latency/throughput harness (`events/sec`,
   p50/p95/p99, alloc/op) hitting `POST /v1/events` and the in-process pipeline
   directly. Makes the pool's numbers real for the demo.
2. Hand fluxx the `Processor` seam: replace the stub in `cmd/gateway` with the
   real feature/sequence/intent chain; back `GET .../intent` with its output.
3. **M7 `wal`** + **M8 `storage`** — offline buffering and persistence.

## Running

```bash
make dev-backend           # :8080
make test                  # go test ./...
cd backend && go test ./... -race
```
