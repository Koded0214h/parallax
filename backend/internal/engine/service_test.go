package engine

import (
	"context"
	"testing"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/internal/baseline"
	"github.com/holiday-heartbreaks/parallax/backend/internal/session"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

func TestServiceLegitimateFlow(t *testing.T) {
	sessions := session.New()
	baselines := baseline.NewStore()
	svc := NewService(Options{
		Sessions:  sessions,
		Baselines: baselines,
	})

	now := time.Now().UTC()
	events := []contracts.Event{
		{
			SessionID:       "sess_legit",
			UserID:          "user_001",
			Type:            contracts.EventLogin,
			Timestamp:       now.Unix(),
			IngestedAtNanos: now.UnixNano(),
			Metadata:        map[string]any{"device_id": "device_primary"},
		},
		{
			SessionID:       "sess_legit",
			UserID:          "user_001",
			Type:            contracts.EventAmountEntered,
			Timestamp:       now.Add(5 * time.Second).Unix(),
			IngestedAtNanos: now.Add(5 * time.Second).UnixNano(),
			Metadata: map[string]any{
				"amount":         30000.0,
				"beneficiary_id": "ben_mother",
			},
		},
	}

	for _, e := range events {
		stored, _ := sessions.Append(e)
		if err := svc.Process(context.Background(), stored); err != nil {
			t.Fatalf("process failed: %v", err)
		}
	}

	resp, ok := svc.GetIntent("sess_legit")
	if !ok {
		t.Fatalf("expected intent response for session")
	}

	if resp.Action != contracts.ActionAllow {
		t.Errorf("expected ActionAllow, got %s", resp.Action)
	}
	if resp.DominantIntent != contracts.IntentLegitimate {
		t.Errorf("expected legitimate dominant, got %s", resp.DominantIntent)
	}
	if resp.Uncertainty > 0.35 {
		t.Errorf("expected low uncertainty, got %f", resp.Uncertainty)
	}
}

func TestServiceAccountTakeoverFlow(t *testing.T) {
	sessions := session.New()
	baselines := baseline.NewStore()
	svc := NewService(Options{
		Sessions:  sessions,
		Baselines: baselines,
	})

	now := time.Now().UTC()
	events := []contracts.Event{
		{
			SessionID:       "sess_ato",
			UserID:          "user_001",
			Type:            contracts.EventDeviceChanged,
			Timestamp:       now.Unix(),
			IngestedAtNanos: now.UnixNano(),
			Metadata:        map[string]any{"device_id": "device_unknown_hacker"},
		},
		{
			SessionID:       "sess_ato",
			UserID:          "user_001",
			Type:            contracts.EventPasswordChanged,
			Timestamp:       now.Add(2 * time.Second).Unix(),
			IngestedAtNanos: now.Add(2 * time.Second).UnixNano(),
		},
		{
			SessionID:       "sess_ato",
			UserID:          "user_001",
			Type:            contracts.EventBeneficiaryCreated,
			Timestamp:       now.Add(4 * time.Second).Unix(),
			IngestedAtNanos: now.Add(4 * time.Second).UnixNano(),
			Metadata:        map[string]any{"beneficiary_id": "ben_hacker_mule"},
		},
		{
			SessionID:       "sess_ato",
			UserID:          "user_001",
			Type:            contracts.EventAmountEntered,
			Timestamp:       now.Add(6 * time.Second).Unix(),
			IngestedAtNanos: now.Add(6 * time.Second).UnixNano(),
			Metadata: map[string]any{
				"amount":         250000.0,
				"beneficiary_id": "ben_hacker_mule",
			},
		},
	}

	for _, e := range events {
		stored, _ := sessions.Append(e)
		if err := svc.Process(context.Background(), stored); err != nil {
			t.Fatalf("process failed: %v", err)
		}
	}

	resp, ok := svc.GetIntent("sess_ato")
	if !ok {
		t.Fatalf("expected intent response")
	}

	if resp.Action != contracts.ActionBlock {
		t.Errorf("expected ActionBlock, got %s", resp.Action)
	}
	if resp.DominantIntent != contracts.IntentAccountTakeover {
		t.Errorf("expected account_takeover, got %s", resp.DominantIntent)
	}
}

func TestServiceSocialEngineeringProbeFlow(t *testing.T) {
	sessions := session.New()
	baselines := baseline.NewStore()
	svc := NewService(Options{
		Sessions:  sessions,
		Baselines: baselines,
	})

	now := time.Now().UTC()
	events := []contracts.Event{
		{
			SessionID:       "sess_soceng",
			UserID:          "user_001",
			Type:            contracts.EventLogin,
			Timestamp:       now.Unix(),
			IngestedAtNanos: now.UnixNano(),
			Metadata:        map[string]any{"device_id": "device_primary"},
		},
		{
			SessionID:       "sess_soceng",
			UserID:          "user_001",
			Type:            contracts.EventBeneficiaryCreated,
			Timestamp:       now.Add(5 * time.Second).Unix(),
			IngestedAtNanos: now.Add(5 * time.Second).UnixNano(),
			Metadata:        map[string]any{"beneficiary_id": "ben_impostor"},
		},
		{
			SessionID:       "sess_soceng",
			UserID:          "user_001",
			Type:            contracts.EventAmountEntered,
			Timestamp:       now.Add(10 * time.Second).Unix(),
			IngestedAtNanos: now.Add(10 * time.Second).UnixNano(),
			Metadata: map[string]any{
				"amount":         250000.0,
				"beneficiary_id": "ben_impostor",
			},
		},
		{
			SessionID:       "sess_soceng",
			UserID:          "user_001",
			Type:            contracts.EventOTPVerified,
			Timestamp:       now.Add(15 * time.Second).Unix(),
			IngestedAtNanos: now.Add(15 * time.Second).UnixNano(),
		},
	}

	for _, e := range events {
		stored, _ := sessions.Append(e)
		if err := svc.Process(context.Background(), stored); err != nil {
			t.Fatalf("process failed: %v", err)
		}
	}

	// Before probe response: should require PROBE
	respBefore, ok := svc.GetIntent("sess_soceng")
	if !ok {
		t.Fatalf("expected intent response")
	}

	if respBefore.Action != contracts.ActionProbe {
		t.Errorf("expected ActionProbe, got %s", respBefore.Action)
	}
	if respBefore.Probe == nil {
		t.Fatalf("expected Probe to be generated")
	}

	// Submit probe response
	respAfter, err := svc.SubmitProbeResponse("sess_soceng", "The bank security department instructed me to move funds")
	if err != nil {
		t.Fatalf("submit probe failed: %v", err)
	}

	if respAfter.Action != contracts.ActionBlock {
		t.Errorf("expected ActionBlock after probe reveals bank impersonation, got %s", respAfter.Action)
	}
	if respAfter.DominantIntent != contracts.IntentSocialEngineering {
		t.Errorf("expected social_engineering dominant, got %s", respAfter.DominantIntent)
	}
	if respAfter.Uncertainty >= respBefore.Uncertainty {
		t.Errorf("expected uncertainty to drop after probe response, before=%f, after=%f",
			respBefore.Uncertainty, respAfter.Uncertainty)
	}
}
