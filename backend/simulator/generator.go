package simulator

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

// LabeledSession represents a synthetic session trajectory with ground truth intent label (prd.md §25).
type LabeledSession struct {
	SessionID     string            `json:"session_id"`
	UserID        string            `json:"user_id"`
	GroundTruth   string            `json:"ground_truth"`
	Events        []contracts.Event `json:"events"`
	ProbeResponse string            `json:"probe_response,omitempty"`
}

// Dataset represents a generated collection of labeled sessions for evaluation.
type Dataset struct {
	LegitimateSessions        []LabeledSession `json:"legitimate_sessions"`
	AccountTakeoverSessions   []LabeledSession `json:"account_takeover_sessions"`
	SocialEngineeringSessions []LabeledSession `json:"social_engineering_sessions"`
	AccidentalSessions        []LabeledSession `json:"accidental_sessions"`
}

// Generator creates realistic synthetic financial sessions with known ground truth.
type Generator struct {
	rng *rand.Rand
}

// NewGenerator creates a deterministic generator with a fixed seed for reproducibility.
func NewGenerator(seed int64) *Generator {
	return &Generator{
		rng: rand.New(rand.NewSource(seed)),
	}
}

// GenerateDataset builds a realistic evaluation dataset (prd.md §26).
func (g *Generator) GenerateDataset(legitCount, atoCount, socEngCount, accidCount int) Dataset {
	d := Dataset{
		LegitimateSessions:        make([]LabeledSession, 0, legitCount),
		AccountTakeoverSessions:   make([]LabeledSession, 0, atoCount),
		SocialEngineeringSessions: make([]LabeledSession, 0, socEngCount),
		AccidentalSessions:        make([]LabeledSession, 0, accidCount),
	}

	for i := 0; i < legitCount; i++ {
		d.LegitimateSessions = append(d.LegitimateSessions, g.generateLegitimate(i))
	}
	for i := 0; i < atoCount; i++ {
		d.AccountTakeoverSessions = append(d.AccountTakeoverSessions, g.generateAccountTakeover(i))
	}
	for i := 0; i < socEngCount; i++ {
		d.SocialEngineeringSessions = append(d.SocialEngineeringSessions, g.generateSocialEngineering(i))
	}
	for i := 0; i < accidCount; i++ {
		d.AccidentalSessions = append(d.AccidentalSessions, g.generateAccidental(i))
	}

	return d
}

func (g *Generator) generateLegitimate(idx int) LabeledSession {
	sessID := fmt.Sprintf("sess_legit_%05d", idx)
	userID := "user_001"
	t := time.Date(2026, 9, 9, 8+g.rng.Intn(10), g.rng.Intn(60), 0, 0, time.UTC)

	// Habitual beneficiaries
	bens := []string{"ben_mother", "ben_landlord", "ben_groceries"}
	benID := bens[g.rng.Intn(len(bens))]

	// Normal amounts around 15,000 - 65,000 with 5% edge cases (e.g. rare higher amount 90k)
	amt := 15000.0 + g.rng.Float64()*45000.0
	if g.rng.Float64() < 0.05 {
		amt = 85000.0 + g.rng.Float64()*25000.0 // Realistic overlap edge case
	}

	events := []contracts.Event{
		{
			EventID:   fmt.Sprintf("ev_%s_1", sessID),
			SessionID: sessID,
			UserID:    userID,
			Type:      contracts.EventLogin,
			Timestamp: t.Unix(),
			Metadata:  map[string]any{"device_id": "device_primary"},
		},
		{
			EventID:   fmt.Sprintf("ev_%s_2", sessID),
			SessionID: sessID,
			UserID:    userID,
			Type:      contracts.EventAmountEntered,
			Timestamp: t.Add(10 * time.Second).Unix(),
			Metadata: map[string]any{
				"amount":         amt,
				"beneficiary_id": benID,
			},
		},
		{
			EventID:   fmt.Sprintf("ev_%s_3", sessID),
			SessionID: sessID,
			UserID:    userID,
			Type:      contracts.EventOTPRequested,
			Timestamp: t.Add(15 * time.Second).Unix(),
		},
		{
			EventID:   fmt.Sprintf("ev_%s_4", sessID),
			SessionID: sessID,
			UserID:    userID,
			Type:      contracts.EventOTPVerified,
			Timestamp: t.Add(25 * time.Second).Unix(),
		},
		{
			EventID:   fmt.Sprintf("ev_%s_5", sessID),
			SessionID: sessID,
			UserID:    userID,
			Type:      contracts.EventTransferCompleted,
			Timestamp: t.Add(30 * time.Second).Unix(),
		},
	}

	return LabeledSession{
		SessionID:   sessID,
		UserID:      userID,
		GroundTruth: contracts.IntentLegitimate,
		Events:      events,
	}
}

func (g *Generator) generateAccountTakeover(idx int) LabeledSession {
	sessID := fmt.Sprintf("sess_ato_%05d", idx)
	userID := "user_001"
	t := time.Date(2026, 9, 9, 2+g.rng.Intn(4), g.rng.Intn(60), 0, 0, time.UTC) // Off-hours (2am - 5am)

	events := []contracts.Event{
		{
			EventID:   fmt.Sprintf("ev_%s_1", sessID),
			SessionID: sessID,
			UserID:    userID,
			Type:      contracts.EventDeviceChanged,
			Timestamp: t.Unix(),
			Metadata:  map[string]any{"device_id": fmt.Sprintf("device_hacker_%d", idx)},
		},
		{
			EventID:   fmt.Sprintf("ev_%s_2", sessID),
			SessionID: sessID,
			UserID:    userID,
			Type:      contracts.EventPasswordChanged,
			Timestamp: t.Add(3 * time.Second).Unix(),
		},
		{
			EventID:   fmt.Sprintf("ev_%s_3", sessID),
			SessionID: sessID,
			UserID:    userID,
			Type:      contracts.EventBeneficiaryCreated,
			Timestamp: t.Add(7 * time.Second).Unix(),
			Metadata:  map[string]any{"beneficiary_id": fmt.Sprintf("ben_mule_%d", idx)},
		},
		{
			EventID:   fmt.Sprintf("ev_%s_4", sessID),
			SessionID: sessID,
			UserID:    userID,
			Type:      contracts.EventAmountEntered,
			Timestamp: t.Add(12 * time.Second).Unix(),
			Metadata: map[string]any{
				"amount":         200000.0 + g.rng.Float64()*150000.0,
				"beneficiary_id": fmt.Sprintf("ben_mule_%d", idx),
			},
		},
	}

	return LabeledSession{
		SessionID:   sessID,
		UserID:      userID,
		GroundTruth: contracts.IntentAccountTakeover,
		Events:      events,
	}
}

func (g *Generator) generateSocialEngineering(idx int) LabeledSession {
	sessID := fmt.Sprintf("sess_soceng_%05d", idx)
	userID := "user_001"
	t := time.Date(2026, 9, 9, 10+g.rng.Intn(8), g.rng.Intn(60), 0, 0, time.UTC)

	events := []contracts.Event{
		{
			EventID:   fmt.Sprintf("ev_%s_1", sessID),
			SessionID: sessID,
			UserID:    userID,
			Type:      contracts.EventLogin,
			Timestamp: t.Unix(),
			Metadata:  map[string]any{"device_id": "device_primary"},
		},
		{
			EventID:   fmt.Sprintf("ev_%s_2", sessID),
			SessionID: sessID,
			UserID:    userID,
			Type:      contracts.EventBeneficiaryCreated,
			Timestamp: t.Add(8 * time.Second).Unix(),
			Metadata:  map[string]any{"beneficiary_id": fmt.Sprintf("ben_scam_%d", idx)},
		},
		{
			EventID:   fmt.Sprintf("ev_%s_3", sessID),
			SessionID: sessID,
			UserID:    userID,
			Type:      contracts.EventAmountEntered,
			Timestamp: t.Add(18 * time.Second).Unix(),
			Metadata: map[string]any{
				"amount":         180000.0 + g.rng.Float64()*120000.0,
				"beneficiary_id": fmt.Sprintf("ben_scam_%d", idx),
			},
		},
		{
			EventID:   fmt.Sprintf("ev_%s_4", sessID),
			SessionID: sessID,
			UserID:    userID,
			Type:      contracts.EventOTPVerified,
			Timestamp: t.Add(28 * time.Second).Unix(),
		},
		{
			EventID:   fmt.Sprintf("ev_%s_5", sessID),
			SessionID: sessID,
			UserID:    userID,
			Type:      contracts.EventIntentProbeResponse,
			Timestamp: t.Add(40 * time.Second).Unix(),
			Metadata: map[string]any{
				"response": "The bank security department instructed me to move funds to safeguard my account immediately",
			},
		},
	}

	return LabeledSession{
		SessionID:     sessID,
		UserID:        userID,
		GroundTruth:   contracts.IntentSocialEngineering,
		Events:        events,
		ProbeResponse: "The bank security department instructed me to move funds to safeguard my account immediately",
	}
}

func (g *Generator) generateAccidental(idx int) LabeledSession {
	sessID := fmt.Sprintf("sess_accid_%05d", idx)
	userID := "user_001"
	t := time.Date(2026, 9, 9, 11+g.rng.Intn(6), g.rng.Intn(60), 0, 0, time.UTC)

	// Habitual beneficiary but 10x amount due to extra zero typo (e.g. 350,000 instead of 35,000)
	events := []contracts.Event{
		{
			EventID:   fmt.Sprintf("ev_%s_1", sessID),
			SessionID: sessID,
			UserID:    userID,
			Type:      contracts.EventLogin,
			Timestamp: t.Unix(),
			Metadata:  map[string]any{"device_id": "device_primary"},
		},
		{
			EventID:   fmt.Sprintf("ev_%s_2", sessID),
			SessionID: sessID,
			UserID:    userID,
			Type:      contracts.EventAmountEntered,
			Timestamp: t.Add(12 * time.Second).Unix(),
			Metadata: map[string]any{
				"amount":         350000.0,
				"beneficiary_id": "ben_mother",
			},
		},
	}

	return LabeledSession{
		SessionID:   sessID,
		UserID:      userID,
		GroundTruth: contracts.IntentAccidental,
		Events:      events,
	}
}
