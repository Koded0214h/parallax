# Frontend Architecture

React + TypeScript + Vite console for observing and driving the Parallax decision
engine. Single-page app, five views behind one shared header/nav, no router
dependency — view switching is local state.

## Views

| View | Path (component) | Purpose |
| --- | --- | --- |
| Dashboard | `views/DashboardView.tsx` | At-a-glance system state: live intent donut, intent comparison bar chart, activity over time, recent sessions table, guided tour entry point. |
| Sessions | `views/SessionsView.tsx` | Every session this client has observed (scenario replays + composed events), sortable, click-through into Playground. |
| Intent | `views/IntentView.tsx` | Deep dive on the current session's intent distribution, uncertainty/entropy, evidence, and policy decision. |
| Activity | `views/ActivityView.tsx` | Raw event timeline for the current session, with a compose-event CTA. |
| Playground | `views/PlaygroundView.tsx` | The demo driver: pick a scenario, step/play/reset it locally, or run it live against the Go gateway and watch the real decision stream back in. |

`App.tsx` owns all cross-view state (current session/events/intent, session log,
backend health, SSE connection) and passes it down — views are presentational.

## Components (`src/components/`)

- **NavBar** — the five-view tab switcher (`ViewId` union type is the source of truth for views).
- **Header** — logo, session/user id, live backend + streaming status, metrics summary, reset/baseline/evaluation/tour entry points.
- **EventComposer** — side-drawer form to hand-build and fire a single event at the pipeline from any view.
- **BaselineModal** / **EvaluationModal** — inspect a customer's behavioural baseline, and the offline evaluation-suite results (`docs/backend.md` evaluator), respectively.
- **IntentDonut**, **IntentBarChart**, **IntentDistribution**, **ActivityLineChart** — chart primitives, all hand-rolled SVG (no charting library dependency).
- **SessionsTable**, **EventTimeline**, **FlowDiagram**, **ScenarioBar** — tabular/visual building blocks reused across views.
- **TourGuide** + `tourSteps.ts` — first-run guided walkthrough; spotlights real DOM elements via `data-tour="..."` attributes and drives `App.tsx`'s view state directly so the tour can jump views mid-walkthrough.
- **InfoTip** — inline hover/tap explainer for jargon (entropy, uncertainty, etc.) used throughout the Intent view.

## Data flow

- **`api.ts`** is the only place that talks to the backend: REST calls (`sendEvent`,
  `intent`, `respondProbe`, `runScenario`, `resetState`, `health`, `metrics`,
  `baselines`) plus `streamEvents`, which opens the `/v1/stream` Server-Sent Events
  connection and reconnects with backoff.
- `API_BASE` comes from `VITE_API_URL` (set in `.env.production`, pointed at the
  Render backend). In dev it's empty, so calls go through Vite's proxy
  (`vite.config.ts`) to `http://localhost:8080` — override with `DEV_BACKEND=<url>
  npm run dev` to point a local frontend at the deployed backend without touching
  config.
- `App.tsx` polls `/health` and `/v1/metrics` every 8s, and layers a live SSE
  subscription on top when the backend is healthy. Every session it sees (scenario
  replay or hand-composed) gets folded into an in-memory `sessionLog`, which is what
  powers the Dashboard/Sessions views — this is client-side only and resets on reload.
- Scenario playback in the Playground has two modes: **local step/play**, which
  walks scripted `targetIntent` values with no network calls (works fully offline),
  and **"Run on Gateway"**, which posts the whole scenario to
  `/v1/scenarios/:id/run`, gets back the real decision instantly, then replays the
  same events one at a time client-side so the reveal reads as a live pipeline
  rather than a single frame flip.

## Keyboard shortcuts

`1`–`5` select demo scenarios A–E, `Space` steps the active scenario forward, `r`
resets the session — all disabled while focus is inside a text input.

## Development

```bash
make dev-frontend        # Vite dev server on :5173, proxies /v1, /health, /healthz to :8080
DEV_BACKEND=https://parallax-n4it.onrender.com npm run dev --prefix frontend   # point dev server at the deployed backend
npm run build --prefix frontend   # tsc -b && vite build
npm run lint --prefix frontend    # oxlint
```

Requires Node 20+. See [`render.yaml`](../render.yaml) for the static-site deploy
config (`VITE_API_URL` is injected at build time there).

## Directory layout

```
frontend/src/
├── App.tsx                # owns state, wires views together
├── api.ts                 # REST + SSE client, only file that talks to the backend
├── types.ts                # shared response/event/scenario types
├── tourSteps.ts            # guided-tour step definitions
├── components/              # presentational + shared building blocks (see above)
└── views/
    ├── DashboardView.tsx
    ├── SessionsView.tsx
    ├── IntentView.tsx
    ├── ActivityView.tsx
    └── PlaygroundView.tsx
```
