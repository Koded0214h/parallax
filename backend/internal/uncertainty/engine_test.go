package uncertainty

import (
	"testing"

	"github.com/holiday-heartbreaks/parallax/backend/internal/features"
	"github.com/holiday-heartbreaks/parallax/backend/internal/intent"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

func TestEvaluateHighConfidence(t *testing.T) {
	eng := NewEngine()

	inf := intent.InferenceResult{
		Hypotheses: map[string]float64{
			contracts.IntentLegitimate:        0.94,
			contracts.IntentAccidental:        0.02,
			contracts.IntentSocialEngineering: 0.02,
			contracts.IntentAccountTakeover:   0.02,
		},
		DominantIntent: contracts.IntentLegitimate,
		DominantScore:  0.94,
		RiskScore:      0.08,
	}

	fs := features.FeatureSet{
		DeviceKnown:           true,
		BeneficiarySeenBefore: true,
		AmountDeviation:       1.0,
		EventCount:            6,
		Amount:                25000.0,
	}

	res := eng.Evaluate(inf, fs)
	if res.Uncertainty > 0.30 {
		t.Errorf("expected low uncertainty (< 0.30), got %f", res.Uncertainty)
	}
	if res.Confidence < 0.70 {
		t.Errorf("expected high confidence (>= 0.70), got %f", res.Confidence)
	}
}

func TestEvaluateConflictingEvidence(t *testing.T) {
	eng := NewEngine()

	inf := intent.InferenceResult{
		Hypotheses: map[string]float64{
			contracts.IntentLegitimate:        0.35,
			contracts.IntentAccidental:        0.05,
			contracts.IntentSocialEngineering: 0.52,
			contracts.IntentAccountTakeover:   0.08,
		},
		DominantIntent: contracts.IntentSocialEngineering,
		DominantScore:  0.52,
		RiskScore:      0.55,
	}

	fs := features.FeatureSet{
		DeviceKnown:      true,
		OTPVerified:      true,
		IsNewBeneficiary: true,
		AmountDeviation:  6.0,
		EventCount:       5,
		Amount:           150000.0,
		ProbeResponded:   false,
	}

	res := eng.Evaluate(inf, fs)
	if res.Uncertainty < 0.45 {
		t.Errorf("expected high uncertainty (>= 0.45) for conflicting evidence, got %f", res.Uncertainty)
	}
	if !eng.IsProbeRequired(res, inf, fs) {
		t.Errorf("expected probe to be required for unprobed social engineering ambiguity")
	}
}

func TestEvaluateProbeReducesUncertainty(t *testing.T) {
	eng := NewEngine()

	inf := intent.InferenceResult{
		Hypotheses: map[string]float64{
			contracts.IntentLegitimate:        0.03,
			contracts.IntentAccidental:        0.02,
			contracts.IntentSocialEngineering: 0.93,
			contracts.IntentAccountTakeover:   0.02,
		},
		DominantIntent: contracts.IntentSocialEngineering,
		DominantScore:  0.93,
		RiskScore:      0.82,
	}

	fs := features.FeatureSet{
		DeviceKnown:              true,
		OTPVerified:              true,
		IsNewBeneficiary:         true,
		AmountDeviation:          6.0,
		EventCount:               6,
		Amount:                   150000.0,
		ProbeResponded:           true,
		ProbeImpersonationSignal: true,
	}

	res := eng.Evaluate(inf, fs)
	if res.Uncertainty > 0.25 {
		t.Errorf("expected low uncertainty after probe (< 0.25), got %f", res.Uncertainty)
	}
	if eng.IsProbeRequired(res, inf, fs) {
		t.Errorf("probe should not be required after probe is answered")
	}
}
