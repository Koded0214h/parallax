package intent

import (
	"testing"

	"github.com/holiday-heartbreaks/parallax/backend/internal/baseline"
	"github.com/holiday-heartbreaks/parallax/backend/internal/features"
	"github.com/holiday-heartbreaks/parallax/backend/internal/sequence"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

func TestInferLegitimate(t *testing.T) {
	eng := NewEngine()
	b := baseline.DefaultBaseline("user_001")

	fs := features.FeatureSet{
		DeviceKnown:           true,
		BeneficiarySeenBefore: true,
		AmountDeviation:       1.0,
		EventCount:            5,
	}
	seq := sequence.SequenceResult{
		MatchedPattern:    sequence.PatternLegitimateFlow,
		LegitPatternScore: 0.9,
	}

	res := eng.Infer(fs, seq, b)
	if res.DominantIntent != contracts.IntentLegitimate {
		t.Fatalf("expected dominant intent legitimate, got %s", res.DominantIntent)
	}
	if res.Hypotheses[contracts.IntentLegitimate] < 0.8 {
		t.Errorf("expected legitimate probability >= 0.8, got %f", res.Hypotheses[contracts.IntentLegitimate])
	}
}

func TestInferAccountTakeover(t *testing.T) {
	eng := NewEngine()
	b := baseline.DefaultBaseline("user_001")

	fs := features.FeatureSet{
		DeviceKnown:            false,
		DeviceID:               "device_hacker",
		DeviceChangedRecent:    true,
		CredentialChangeRecent: true,
		IsNewBeneficiary:       true,
		AmountDeviation:        8.0,
		RapidSequence:          true,
	}
	seq := sequence.SequenceResult{
		MatchedPattern:  sequence.PatternAccountTakeover,
		ATOPatternScore: 0.95,
	}

	res := eng.Infer(fs, seq, b)
	if res.DominantIntent != contracts.IntentAccountTakeover {
		t.Fatalf("expected dominant intent account_takeover, got %s", res.DominantIntent)
	}
	if res.Hypotheses[contracts.IntentAccountTakeover] < 0.85 {
		t.Errorf("expected ATO probability >= 0.85, got %f", res.Hypotheses[contracts.IntentAccountTakeover])
	}
	if res.RiskScore < 0.8 {
		t.Errorf("expected high risk score, got %f", res.RiskScore)
	}
}

func TestInferSocialEngineeringWithProbe(t *testing.T) {
	eng := NewEngine()
	b := baseline.DefaultBaseline("user_001")

	// Prior to probe:
	fsBefore := features.FeatureSet{
		DeviceKnown:      true,
		IsNewBeneficiary: true,
		AmountDeviation:  6.0,
		OTPVerified:      true,
		EventCount:       5,
	}
	seqBefore := sequence.SequenceResult{
		MatchedPattern:     sequence.PatternSocialEngineering,
		SocEngPatternScore: 0.75,
	}
	resBefore := eng.Infer(fsBefore, seqBefore, b)

	// After probe response indicating impersonation:
	fsAfter := fsBefore
	fsAfter.ProbeResponded = true
	fsAfter.ProbeImpersonationSignal = true
	seqAfter := sequence.SequenceResult{
		MatchedPattern:     sequence.PatternSocialEngineering,
		SocEngPatternScore: 0.95,
	}
	resAfter := eng.Infer(fsAfter, seqAfter, b)

	if resAfter.Hypotheses[contracts.IntentSocialEngineering] <= resBefore.Hypotheses[contracts.IntentSocialEngineering] {
		t.Fatalf("expected social engineering probability to increase after probe response")
	}
	if resAfter.DominantIntent != contracts.IntentSocialEngineering {
		t.Fatalf("expected dominant intent social_engineering, got %s", resAfter.DominantIntent)
	}
}
