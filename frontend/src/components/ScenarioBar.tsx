import type { Scenario } from '../types'
import {
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
}: ScenarioBarProps) {
  const totalSteps = activeScenario.steps.length
  const isFinished = currentStepIndex >= totalSteps

  return (
    <div className="scenario-bar">
      <div className="scenario-selector">
        <span className="scenario-prefix font-mono">SCENARIO:</span>
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
                {sc.code === 'C' && <span className="tab-badge">CENTERPIECE</span>}
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
          <span className="step-label">STEPS</span>
        </div>

        <div className="controls-group">
          <button
            onClick={onTogglePlay}
            disabled={isFinished}
            className={`control-btn primary ${isPlaying ? 'playing' : ''}`}
            title={isPlaying ? 'Pause scenario playback' : 'Play scenario trajectory'}
          >
            {isPlaying ? (
              <>
                <PauseIcon size={12} />
                <span>PAUSE</span>
              </>
            ) : (
              <>
                <PlayIcon size={12} />
                <span>PLAY</span>
              </>
            )}
          </button>

          <button
            onClick={onStepForward}
            disabled={isFinished || isPlaying}
            className="control-btn"
            title="Inject next event (Shortcut: Space)"
          >
            <StepForwardIcon size={12} />
            <span>STEP</span>
          </button>

          <button
            onClick={onReset}
            className="control-btn reset"
            title="Reset trajectory (Shortcut: R)"
          >
            <ResetIcon size={12} />
            <span>RESET</span>
          </button>
        </div>

        <div className="keyboard-hints font-mono">
          <span className="hint-pill">1-4</span>
          <span className="hint-pill">SPACE</span>
          <span className="hint-pill">R</span>
        </div>
      </div>
    </div>
  )
}
