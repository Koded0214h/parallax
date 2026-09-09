package policy

import (
	"testing"

	"github.com/holiday-heartbreaks/parallax/backend/internal/features"
	"github.com/holiday-heartbreaks/parallax/backend/internal/intent"
	"github.com/holiday-heartbreaks/parallax/backend/internal/uncertainty"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

func TestPolicyAllow(t *testing.T) {
	eng := NewEngine(DefaultConfig())

	inf := intent.InferenceResult{
		Hypotheses: map[string]float64{
			contracts.IntentLegitimate: 0.95,
		},
		DominantIntent: contracts.IntentLegitimate,
		DominantScore:  0.95,
		RiskScore:      0.05,
	}
	u := uncertainty.Result{
		Uncertainty: 0.08,
		Confidence:  0.92,
	}
	fs := features.FeatureSet{
		DeviceKnown:           true,
		BeneficiarySeenBefore: true,
	}

	d := eng.Decide(inf, u, fs)
	if d.Action != contracts.ActionAllow {
		t.Fatalf("expected ActionAllow, got %s", d.Action)
	}
}

func TestPolicyBlockATO(t *testing.T) {
	eng := NewEngine(DefaultConfig())

	inf := intent.InferenceResult{
		Hypotheses: map[string]float64{
			contracts.IntentAccountTakeover: 0.92,
		},
		DominantIntent: contracts.IntentAccountTakeover,
		DominantScore:  0.92,
		RiskScore:      0.92,
	}
	u := uncertainty.Result{
		Uncertainty: 0.12,
		Confidence:  0.88,
	}
	fs := features.FeatureSet{
		DeviceKnown:            false,
		CredentialChangeRecent: true,
	}

	d := eng.Decide(inf, u, fs)
	if d.Action != contracts.ActionBlock {
		t.Fatalf("expected ActionBlock for high confidence ATO, got %s", d.Action)
	}
}

func TestPolicyProbeSocialEngineering(t *testing.T) {
	eng := NewEngine(DefaultConfig())

	inf := intent.InferenceResult{
		Hypotheses: map[string]float64{
			contracts.IntentSocialEngineering: 0.58,
			contracts.IntentLegitimate:        0.32,
		},
		DominantIntent: contracts.IntentSocialEngineering,
		DominantScore:  0.58,
		RiskScore:      0.55,
	}
	u := uncertainty.Result{
		Uncertainty: 0.48,
		Confidence:  0.52,
	}
	fs := features.FeatureSet{
		DeviceKnown:      true,
		IsNewBeneficiary: true,
		ProbeResponded:   false,
	}

	d := eng.Decide(inf, u, fs)
	if d.Action != contracts.ActionProbe {
		t.Fatalf("expected ActionProbe for high uncertainty social engineering, got %s", d.Action)
	}
	if !d.RequiresProbe {
		t.Errorf("expected RequiresProbe true")
	}
}

func TestPolicyVerifyAccidental(t *testing.T) {
	eng := NewEngine(DefaultConfig())

	inf := intent.InferenceResult{
		Hypotheses: map[string]float64{
			contracts.IntentAccidental: 0.75,
			contracts.IntentLegitimate: 0.20,
		},
		DominantIntent: contracts.IntentAccidental,
		DominantScore:  0.75,
		RiskScore:      0.35,
	}
	u := uncertainty.Result{
		Uncertainty: 0.25,
		Confidence:  0.75,
	}
	fs := features.FeatureSet{
		DeviceKnown:           true,
		BeneficiarySeenBefore: true,
		ZeroPaddingAnomaly:    true,
	}

	d := eng.Decide(inf, u, fs)
	if d.Action != contracts.ActionVerify {
		t.Fatalf("expected ActionVerify for accidental transfer, got %s", d.Action)
	}
}
