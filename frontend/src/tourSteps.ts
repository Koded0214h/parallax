import type { TourStep } from './components/TourGuide'

export const TOUR_STEPS: TourStep[] = [
  {
    target: 'tour-nav',
    view: 'dashboard',
    title: 'Welcome to Parallax',
    body: "Five tabs, always here. Dashboard is the big picture — you can get back to it anytime. Let's walk through what each piece does.",
  },
  {
    target: 'tour-concepts',
    view: 'dashboard',
    title: "Don't know a term?",
    body: 'Click any of these — Intent, Evidence, Account Takeover, Social Engineering — for a plain-language explanation. They work the same way everywhere in the app.',
  },
  {
    target: 'tour-flow',
    view: 'dashboard',
    title: 'The whole system, in one line',
    body: 'A transaction becomes a stream of events. Events become measurable evidence. Evidence becomes a guess at intent. That guess becomes a decision — allow, verify, pause, or block.',
  },
  {
    target: 'tour-charts',
    view: 'dashboard',
    title: 'What Parallax is seeing right now',
    body: 'The intent mix across sessions, a side-by-side comparison of the four categories, and activity/risk over time — all update live as sessions run.',
  },
  {
    target: 'tour-sessions',
    view: 'dashboard',
    title: 'Every session, one table',
    body: 'Every scenario you run or event you compose shows up here. Click a row to reopen its full detail in the Playground.',
  },
  {
    target: 'tour-scenarios',
    view: 'playground',
    title: 'This is the Playground',
    body: 'Four scripted scenarios — a normal transfer, an account takeover, a social-engineering scam, and an accidental transfer. Click one to load it.',
  },
  {
    target: 'tour-play',
    view: 'playground',
    title: 'Try it — for real',
    body: "This isn't a mockup. Click Play below and watch the timeline and intent panel react live. I'll continue on my own once it's running.",
    interactive: true,
  },
  {
    target: 'tour-compose',
    view: 'playground',
    title: 'Or write your own',
    body: "Skip the script entirely. Compose Event builds one event by hand — pick the type, set an amount or a beneficiary — and fires it straight at the pipeline.",
  },
  {
    target: 'tour-decision',
    view: 'playground',
    title: "That's the verdict",
    body: "Every decision — allow, verify, pause-and-ask, or block — lands here with the evidence behind it, in plain English. That's the whole point of Parallax.",
  },
]
