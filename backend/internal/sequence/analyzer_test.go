package sequence

import (
	"testing"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/internal/features"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

func TestAnalyzeATOPattern(t *testing.T) {
	analyzer := NewAnalyzer()
	now := time.Now().UTC()

	trajectory := []contracts.Event{
		{Type: contracts.EventLogin, Timestamp: now.Unix()},
		{Type: contracts.EventDeviceChanged, Timestamp: now.Add(2 * time.Second).Unix()},
		{Type: contracts.EventPasswordChanged, Timestamp: now.Add(5 * time.Second).Unix()},
		{Type: contracts.EventBeneficiaryCreated, Timestamp: now.Add(10 * time.Second).Unix()},
		{Type: contracts.EventAmountEntered, Timestamp: now.Add(15 * time.Second).Unix()},
	}

	fs := features.FeatureSet{
		DeviceKnown:            false,
		DeviceChangedRecent:    true,
		CredentialChangeRecent: true,
		IsNewBeneficiary:       true,
		AmountDeviation:        8.0,
	}

	res := analyzer.Analyze(trajectory, fs)
	if res.MatchedPattern != PatternAccountTakeover {
		t.Fatalf("expected PatternAccountTakeover, got %v", res.MatchedPattern)
	}
	if res.ATOPatternScore < 0.9 {
		t.Errorf("expected high ATO pattern score, got %f", res.ATOPatternScore)
	}
}

func TestAnalyzeSocialEngineeringPattern(t *testing.T) {
	analyzer := NewAnalyzer()
	now := time.Now().UTC()

	trajectory := []contracts.Event{
		{Type: contracts.EventLogin, Timestamp: now.Unix()},
		{Type: contracts.EventBeneficiaryCreated, Timestamp: now.Add(5 * time.Second).Unix()},
		{Type: contracts.EventAmountEntered, Timestamp: now.Add(10 * time.Second).Unix()},
		{Type: contracts.EventOTPVerified, Timestamp: now.Add(15 * time.Second).Unix()},
		{Type: contracts.EventIntentProbeResponse, Timestamp: now.Add(20 * time.Second).Unix()},
	}

	fs := features.FeatureSet{
		DeviceKnown:              true,
		IsNewBeneficiary:         true,
		AmountDeviation:          6.5,
		OTPVerified:              true,
		ProbeResponded:           true,
		ProbeImpersonationSignal: true,
	}

	res := analyzer.Analyze(trajectory, fs)
	if res.MatchedPattern != PatternSocialEngineering {
		t.Fatalf("expected PatternSocialEngineering, got %v", res.MatchedPattern)
	}
	if res.SocEngPatternScore < 0.9 {
		t.Errorf("expected high Social Engineering pattern score, got %f", res.SocEngPatternScore)
	}
}

func TestAnalyzeAccidentalPattern(t *testing.T) {
	analyzer := NewAnalyzer()
	now := time.Now().UTC()

	trajectory := []contracts.Event{
		{Type: contracts.EventLogin, Timestamp: now.Unix()},
		{Type: contracts.EventAmountEntered, Timestamp: now.Add(5 * time.Second).Unix()},
	}

	fs := features.FeatureSet{
		DeviceKnown:           true,
		BeneficiarySeenBefore: true,
		AmountDeviation:       10.0,
		ZeroPaddingAnomaly:    true,
	}

	res := analyzer.Analyze(trajectory, fs)
	if res.MatchedPattern != PatternAccidentalTransfer {
		t.Fatalf("expected PatternAccidentalTransfer, got %v", res.MatchedPattern)
	}
}
