package probes

import (
	"fmt"
	"sync"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/internal/features"
	"github.com/holiday-heartbreaks/parallax/backend/internal/intent"
)

// Option represents a structured multiple-choice answer presented in the UI.
type Option struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Category string `json:"category"`
}

// Question is the active context probe emitted to reduce intent uncertainty (prd.md §15).
type Question struct {
	ProbeID       string   `json:"probe_id"`
	SessionID     string   `json:"session_id"`
	Prompt        string   `json:"prompt"`
	Options       []Option `json:"options"`
	AllowFreeform bool     `json:"allow_freeform"`
	CreatedAt     int64    `json:"created_at"`
	RespondedAt   int64    `json:"responded_at,omitempty"`
	ResponseText  string   `json:"response_text,omitempty"`
	Completed     bool     `json:"completed"`
}

// Service manages the lifecycle of contextual intent probes.
type Service struct {
	mu     sync.RWMutex
	probes map[string]*Question // keyed by session_id
}

// NewService creates a new probe management service.
func NewService() *Service {
	return &Service{
		probes: make(map[string]*Question),
	}
}

// Generate creates a targeted contextual probe based on the session's uncertainty profile.
func (s *Service) Generate(sessionID string, inf intent.InferenceResult, fs features.FeatureSet) *Question {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If a completed or active probe already exists for this session, return it
	if existing, ok := s.probes[sessionID]; ok {
		return existing
	}

	probeID := fmt.Sprintf("probe_%s_%d", sessionID, time.Now().Unix())
	prompt := "To ensure your transaction is safe, please tell us the primary purpose of this transfer:"

	if fs.ZeroPaddingAnomaly {
		prompt = "This transfer amount is significantly higher than usual for this contact. Did you intend to send this exact amount?"
	}

	options := []Option{
		{
			ID:       "opt_vendor_friend",
			Text:     "Paying a trusted friend, family member, or verified merchant.",
			Category: "legitimate",
		},
		{
			ID:       "opt_bank_agent",
			Text:     "A bank representative or security officer told me to transfer funds to protect my account.",
			Category: "impersonation",
		},
		{
			ID:       "opt_lottery_job",
			Text:     "Processing fee for a prize, lottery, grant, job offer, or investment payout.",
			Category: "advance_fee",
		},
		{
			ID:       "opt_emergency_call",
			Text:     "Responding to an urgent call or message about an unexpected crisis.",
			Category: "urgency",
		},
	}

	q := &Question{
		ProbeID:       probeID,
		SessionID:     sessionID,
		Prompt:        prompt,
		Options:       options,
		AllowFreeform: true,
		CreatedAt:     time.Now().Unix(),
	}

	s.probes[sessionID] = q
	return q
}

// Get retrieves the probe for a session, if any.
func (s *Service) Get(sessionID string) (*Question, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q, ok := s.probes[sessionID]
	if !ok {
		return nil, false
	}
	cp := *q
	return &cp, true
}

// SubmitResponse records the user's answer and marks the probe completed.
func (s *Service) SubmitResponse(sessionID, responseText string) (*Question, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	q, ok := s.probes[sessionID]
	if !ok {
		// If probe hadn't been formally stored yet, instantiate on the fly
		q = &Question{
			ProbeID:       fmt.Sprintf("probe_%s_%d", sessionID, time.Now().Unix()),
			SessionID:     sessionID,
			Prompt:        "Context verification response",
			AllowFreeform: true,
			CreatedAt:     time.Now().Unix(),
		}
		s.probes[sessionID] = q
	}

	q.Completed = true
	q.ResponseText = responseText
	q.RespondedAt = time.Now().Unix()

	cp := *q
	return &cp, nil
}

// Reset clears all active probes.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.probes = make(map[string]*Question)
}
