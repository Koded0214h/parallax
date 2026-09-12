import type { Scenario } from '../types'
import {
  ActivityIcon,
  PauseIcon,
  PlayIcon,
  ResetIcon,
  StepForwardIcon,
  TerminalIcon,
} from './Icons'

interface ScenarioBarProps {
  scenarios: Scenario[]
  activeScenario: Scenario
  onSelectScenario: (scenario: Scenario) => void
  currentStepIndex: number
  isPlaying: boolean
  onTogglePlay: () => void
  onStepForward: () => void
  onReset: () => void
  backendHealthy?: boolean
  backendRunning?: boolean
  onRunBackend?: () => void
  onOpenComposer?: () => void
}

export function ScenarioBar({
  scenarios,
  activeScenario,
  onSelectScenario,
  currentStepIndex,
  isPlaying,
  onTogglePlay,
  onStepForward,
  onReset,
  backendHealthy,
  backendRunning,
  onRunBackend,
  onOpenComposer,
}: ScenarioBarProps) {
  const totalSteps = activeScenario.steps.length
  const isFinished = currentStepIndex >= totalSteps
  const canResume = !isPlaying && !isFinished && currentStepIndex > 0

  return (
    <div className="scenario-bar">
      <div className="scenario-selector" data-tour="tour-scenarios">
        <span className="scenario-prefix">Scenario:</span>
        <select
          className="scenario-select"
          value={activeScenario.id}
          onChange={(e) => {
            const next = scenarios.find((s) => s.id === e.target.value)
            if (next) onSelectScenario(next)
          }}
        >
          {scenarios.map((sc, idx) => (
            <option key={sc.id} value={sc.id}>
              {idx + 1}. {sc.name}
              {sc.code === 'C' ? ' — Coercion Probe' : ''}
            </option>
          ))}
        </select>
        <span className="step-counter font-mono">
          <span className="step-num tnum">{currentStepIndex}</span>
          <span className="step-sep">/</span>
          <span className="step-total tnum">{totalSteps}</span>
          <span className="step-label">steps</span>
        </span>
      </div>

      <div className="scenario-controls">
        <div className="controls-group">
          {/* Play / Stop */}
          <button
            onClick={onTogglePlay}
            disabled={isFinished}
            className={`control-btn primary ${isPlaying ? 'playing' : ''}`}
            title={isPlaying ? 'Stop scenario playback' : 'Play scenario trajectory'}
            data-tour="tour-play"
          >
            {isPlaying ? (
              <>
                <PauseIcon size={12} />
                <span>Stop</span>
              </>
            ) : (
              <>
                <PlayIcon size={12} />
                <span>Play</span>
              </>
            )}
          </button>

          {/* Resume from a paused mid-scenario position */}
          <button
            onClick={onTogglePlay}
            disabled={!canResume}
            className="control-btn"
            title="Resume auto-play from where you stopped"
          >
            <PlayIcon size={12} />
            <span>Resume</span>
          </button>

          {/* Single Step Forward */}
          <button
            onClick={onStepForward}
            disabled={isFinished || isPlaying}
            className="control-btn"
            title="Inject next event (Shortcut: Space)"
          >
            <StepForwardIcon size={12} />
            <span>Step</span>
          </button>

          {/* Reset */}
          <button onClick={onReset} className="control-btn reset" title="Reset trajectory (Shortcut: R)">
            <ResetIcon size={12} />
            <span>Reset</span>
          </button>
        </div>

        <div className="controls-group controls-group-secondary">
          {/* Run End-to-End on Live Go Gateway */}
          {backendHealthy && onRunBackend && (
            <button
              onClick={onRunBackend}
              disabled={backendRunning}
              className={`control-btn run-gateway ${backendRunning ? 'loading' : ''}`}
              title="Run entire scenario end-to-end directly on the live Go gateway pipeline (POST /v1/scenarios/:id/run)"
            >
              <ActivityIcon size={12} />
              <span>{backendRunning ? 'Running…' : 'Run on Gateway'}</span>
            </button>
          )}

          {/* Hand-build one event */}
          {onOpenComposer && (
            <button
              onClick={onOpenComposer}
              className="control-btn compose"
              title="Compose a single custom event by hand"
              data-tour="tour-compose"
            >
              <TerminalIcon size={12} />
              <span>Compose Event</span>
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
