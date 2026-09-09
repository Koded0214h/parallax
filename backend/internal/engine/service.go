package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/internal/baseline"
	"github.com/holiday-heartbreaks/parallax/backend/internal/explanation"
	"github.com/holiday-heartbreaks/parallax/backend/internal/features"
	"github.com/holiday-heartbreaks/parallax/backend/internal/intent"
	"github.com/holiday-heartbreaks/parallax/backend/internal/policy"
	"github.com/holiday-heartbreaks/parallax/backend/internal/probes"
	"github.com/holiday-heartbreaks/parallax/backend/internal/sequence"
	"github.com/holiday-heartbreaks/parallax/backend/internal/session"
	"github.com/holiday-heartbreaks/parallax/backend/internal/stream"
	"github.com/holiday-heartbreaks/parallax/backend/internal/uncertainty"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

// Options configure the intent inference coordinator.
type Options struct {
	Logger            *slog.Logger
	Sessions          *session.Store
	Baselines         *baseline.Store
	Bus               *stream.Bus
	PolicyConfig      policy.Config
}

// Service is the unified intelligence and decision engine coordinator.
type Service struct {
	logger            *slog.Logger
	sessions          *session.Store
	baselines         *baseline.Store
	bus               *stream.Bus

	features          *features.Extractor
	sequence          *sequence.Analyzer
	intent            *intent.Engine
	uncertainty       *uncertainty.Engine
	probes            *probes.Service
	policy            *policy.Engine
	explanation       *explanation.Engine

	mu                sync.RWMutex
	latestDecisions   map[string]contracts.IntentResponse
}

// NewService constructs a fully wired intelligence engine service.
func NewService(opts Options) *Service {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	sessions := opts.Sessions
	if sessions == nil {
		sessions = session.New()
	}
	baselines := opts.Baselines
	if baselines == nil {
		baselines = baseline.NewStore()
	}
	pCfg := opts.PolicyConfig
	if pCfg.AllowMaxRisk == 0 {
		pCfg = policy.DefaultConfig()
	}

	return &Service{
		logger:          logger,
		sessions:        sessions,
		baselines:       baselines,
		bus:             opts.Bus,
		features:        features.NewExtractor(),
		sequence:        sequence.NewAnalyzer(),
		intent:          intent.NewEngine(),
		uncertainty:     uncertainty.NewEngine(),
		probes:          probes.NewService(),
		policy:          policy.NewEngine(pCfg),
		explanation:     explanation.NewEngine(),
		latestDecisions: make(map[string]contracts.IntentResponse),
	}
}

// Process implements worker.Processor, evaluating an event in real-time.
func (s *Service) Process(ctx context.Context, e contracts.Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	res, err := s.EvaluateSession(e.SessionID)
	if err != nil {
		s.logger.Error("evaluation failed", "session", e.SessionID, "err", err)
		return err
	}

	s.logger.Info("evaluated intent",
		"session", e.SessionID,
		"action", res.Action,
		"dominant", res.DominantIntent,
		"uncertainty", res.Uncertainty,
		"risk", res.RiskScore,
	)

	return nil
}

// EvaluateSession recalculates inference for an entire session trajectory.
func (s *Service) EvaluateSession(sessionID string) (contracts.IntentResponse, error) {
	trajectory, ok := s.sessions.Trajectory(sessionID)
	if !ok || len(trajectory) == 0 {
		return contracts.IntentResponse{}, errors.New("empty trajectory")
	}
	eventsSeen := len(trajectory)

	lastEvent := trajectory[eventsSeen-1]
	userBaseline := s.baselines.GetOrCreate(lastEvent.UserID)

	// Stage 1: Feature Extraction
	fs := s.features.Extract(trajectory, userBaseline)

	// Stage 2: Sequence Analysis
	seq := s.sequence.Analyze(trajectory, fs)

	// Stage 3: Intent Hypotheses Inference
	inf := s.intent.Infer(trajectory, fs, seq, userBaseline)

	// Stage 4: Uncertainty Evaluation
	u := s.uncertainty.Evaluate(inf, fs)

	// Stage 5: Context Probe Management
	var probePayload *contracts.ProbePayload
	if s.uncertainty.IsProbeRequired(u, inf, fs) {
		q := s.probes.Generate(sessionID, inf, fs)
		if q != nil {
			var opts []contracts.ProbeOption
			for _, o := range q.Options {
				opts = append(opts, contracts.ProbeOption{
					ID:       o.ID,
					Text:     o.Text,
					Category: o.Category,
				})
			}
			probePayload = &contracts.ProbePayload{
				ProbeID:       q.ProbeID,
				Prompt:        q.Prompt,
				Options:       opts,
				AllowFreeform: q.AllowFreeform,
				Completed:     q.Completed,
				Response:      q.ResponseText,
			}
		}
	} else if existingProbe, ok := s.probes.Get(sessionID); ok {
		var opts []contracts.ProbeOption
		for _, o := range existingProbe.Options {
			opts = append(opts, contracts.ProbeOption{
				ID:       o.ID,
				Text:     o.Text,
				Category: o.Category,
			})
		}
		probePayload = &contracts.ProbePayload{
			ProbeID:       existingProbe.ProbeID,
			Prompt:        existingProbe.Prompt,
			Options:       opts,
			AllowFreeform: existingProbe.AllowFreeform,
			Completed:     existingProbe.Completed,
			Response:      existingProbe.ResponseText,
		}
	}

	// Stage 6: Policy Decision
	dec := s.policy.Decide(inf, u, fs)

	// Stage 7: Human-readable Explanation
	expl := s.explanation.Explain(dec, inf, u, fs)

	// Update baseline on legitimate transfer completion
	if dec.Action == contracts.ActionAllow && lastEvent.Type == contracts.EventTransferCompleted && fs.Amount > 0 {
		s.baselines.RecordTransfer(
			lastEvent.UserID,
			fs.Amount,
			fs.BeneficiaryID,
			fs.DeviceID,
			time.Unix(lastEvent.Timestamp, 0).UTC(),
		)
	}

	response := contracts.IntentResponse{
		SessionID:      sessionID,
		Hypotheses:     inf.Hypotheses,
		Uncertainty:    u.Uncertainty,
		Action:         dec.Action,
		Evidence:       expl.EvidenceBullets,
		EventsSeen:     eventsSeen,
		Explanation:    expl.Narrative,
		Confidence:     expl.Confidence,
		RiskScore:      inf.RiskScore,
		DominantIntent: inf.DominantIntent,
		Probe:          probePayload,
	}

	s.mu.Lock()
	s.latestDecisions[sessionID] = response
	s.mu.Unlock()

	return response, nil
}

// GetIntent returns the latest computed intent for a session.
func (s *Service) GetIntent(sessionID string) (contracts.IntentResponse, bool) {
	s.mu.RLock()
	resp, ok := s.latestDecisions[sessionID]
	s.mu.RUnlock()

	if ok {
		return resp, true
	}

	// If not cached, attempt on-demand evaluation if session exists
	if s.sessions.Len(sessionID) > 0 {
		evaluated, err := s.EvaluateSession(sessionID)
		if err == nil {
			return evaluated, true
		}
	}

	return contracts.IntentResponse{}, false
}

// SubmitProbeResponse handles a user probe answer and triggers immediate re-inference.
func (s *Service) SubmitProbeResponse(sessionID, responseText string) (contracts.IntentResponse, error) {
	_, err := s.probes.SubmitResponse(sessionID, responseText)
	if err != nil {
		return contracts.IntentResponse{}, fmt.Errorf("probe submission failed: %w", err)
	}

	// Append INTENT_PROBE_RESPONSE to the session store
	now := time.Now().UTC()
	probeEvent := contracts.Event{
		EventID:         fmt.Sprintf("ev_probe_%d", now.UnixNano()),
		SessionID:       sessionID,
		UserID:          "unknown",
		Type:            contracts.EventIntentProbeResponse,
		Timestamp:       now.Unix(),
		IngestedAtNanos: now.UnixNano(),
		Metadata: map[string]any{
			"response": responseText,
		},
	}

	// Inherit user_id from prior events in session
	trajectory, ok := s.sessions.Trajectory(sessionID)
	if ok && len(trajectory) > 0 {
		probeEvent.UserID = trajectory[0].UserID
	}

	stored, _ := s.sessions.Append(probeEvent)
	if s.bus != nil {
		s.bus.Publish(stored)
	}

	// Immediate recalculation
	return s.EvaluateSession(sessionID)
}

// Reset clears state across sessions, probes, and caches.
func (s *Service) Reset() {
	s.mu.Lock()
	s.latestDecisions = make(map[string]contracts.IntentResponse)
	s.mu.Unlock()

	s.probes.Reset()
	s.baselines.Reset()
}
