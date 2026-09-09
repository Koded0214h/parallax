package probes

import (
	"testing"

	"github.com/holiday-heartbreaks/parallax/backend/internal/features"
	"github.com/holiday-heartbreaks/parallax/backend/internal/intent"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

func TestProbeLifecycle(t *testing.T) {
	svc := NewService()
	sessionID := "sess_probe_test"

	inf := intent.InferenceResult{
		Hypotheses: map[string]float64{
			contracts.IntentSocialEngineering: 0.65,
			contracts.IntentLegitimate:        0.25,
		},
		DominantIntent: contracts.IntentSocialEngineering,
		RiskScore:      0.65,
	}
	fs := features.FeatureSet{
		DeviceKnown:      true,
		IsNewBeneficiary: true,
		AmountDeviation:  5.0,
	}

	q := svc.Generate(sessionID, inf, fs)
	if q == nil {
		t.Fatalf("expected generated question")
	}
	if q.Completed {
		t.Errorf("probe should not be completed initially")
	}
	if len(q.Options) == 0 {
		t.Errorf("expected options to be provided")
	}

	got, found := svc.Get(sessionID)
	if !found || got.ProbeID != q.ProbeID {
		t.Fatalf("expected probe to be retrievable")
	}

	ans := "A bank officer told me to move money to safe account"
	completed, err := svc.SubmitResponse(sessionID, ans)
	if err != nil {
		t.Fatalf("failed to submit response: %v", err)
	}
	if !completed.Completed {
		t.Errorf("expected probe to be completed")
	}
	if completed.ResponseText != ans {
		t.Errorf("expected response text %q, got %q", ans, completed.ResponseText)
	}
}
