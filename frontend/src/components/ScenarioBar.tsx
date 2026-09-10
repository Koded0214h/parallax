import type { Scenario } from '../types'
import {
  ActivityIcon,
  PauseIcon,
  PlayIcon,
  ResetIcon,
  StepForwardIcon,
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
}: ScenarioBarProps) {
  const totalSteps = activeScenario.steps.length
  const isFinished = currentStepIndex >= totalSteps

  return (
    <div className="scenario-bar">
      <div className="scenario-selector">
        <span className="scenario-prefix">Scenario:</span>
        <div className="scenario-tabs">
          {scenarios.map((sc, idx) => {
            const isSelected = sc.id === activeScenario.id
            return (
              <button
                key={sc.id}
                onClick={() => onSelectScenario(sc)}
                className={`scenario-tab ${isSelected ? 'active' : ''}`}
                title={sc.description}
              >
                <span className="tab-code font-mono">{idx + 1}</span>
                <span className="tab-name">{sc.name}</span>
                {sc.code === 'C' && <span className="tab-badge attack-badge">Coercion Probe</span>}
              </button>
            )
          })}
        </div>
      </div>

      <div className="scenario-controls">
        <div className="step-counter font-mono">
          <span className="step-num tnum">{currentStepIndex}</span>
          <span className="step-sep">/</span>
          <span className="step-total tnum">{totalSteps}</span>
          <span className="step-label">steps</span>
        </div>

        <div className="controls-group">
          {/* Step-by-Step Play/Pause */}
          <button
            onClick={onTogglePlay}
            disabled={isFinished}
            className={`control-btn primary ${isPlaying ? 'playing' : ''}`}
            title={isPlaying ? 'Pause scenario playback' : 'Play scenario trajectory'}
          >
            {isPlaying ? (
              <>
                <PauseIcon size={12} />
                <span>Pause</span>
              </>
            ) : (
              <>
                <PlayIcon size={12} />
                <span>Play</span>
              </>
            )}
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

          {/* Run End-to-End on Live Go Gateway */}
          {backendHealthy && onRunBackend && (
            <button
              onClick={onRunBackend}
              disabled={backendRunning}
              className={`control-btn run-gateway ${backendRunning ? 'loading' : ''}`}
              title="Run entire scenario end-to-end directly on the live Go gateway pipeline (POST /v1/scenarios/:id/run)"
            >
              <ActivityIcon size={12} />
              <span>{backendRunning ? 'Running...' : 'Run on Gateway'}</span>
            </button>
          )}

          {/* Reset */}
          <button
            onClick={onReset}
            className="control-btn reset"
            title="Reset trajectory (Shortcut: R)"
          >
            <ResetIcon size={12} />
            <span>Reset</span>
          </button>
        </div>

        <div className="keyboard-hints font-mono">
          <span className="hint-pill">1–5</span>
          <span className="hint-pill">Space</span>
          <span className="hint-pill">R</span>
        </div>
      </div>
    </div>
  )
}
