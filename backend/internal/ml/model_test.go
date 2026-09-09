package ml

import (
	"testing"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/internal/baseline"
	"github.com/holiday-heartbreaks/parallax/backend/internal/features"
	"github.com/holiday-heartbreaks/parallax/backend/internal/sequence"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

func TestClassifierLegitimateTrafficNoFalseAlarm(t *testing.T) {
	clf := NewClassifier()
	b := baseline.DefaultBaseline("user_001")

	now := time.Now().UTC()
	trajectory := []contracts.Event{
		{Type: contracts.EventLogin, Timestamp: now.Unix()},
		{Type: contracts.EventAmountEntered, Timestamp: now.Add(5 * time.Second).Unix()},
		{Type: contracts.EventOTPRequested, Timestamp: now.Add(8 * time.Second).Unix()},
		{Type: contracts.EventOTPVerified, Timestamp: now.Add(12 * time.Second).Unix()},
		{Type: contracts.EventTransferCompleted, Timestamp: now.Add(15 * time.Second).Unix()},
	}

	fs := features.FeatureSet{
		DeviceKnown:           true,
		BeneficiarySeenBefore: true,
		AmountDeviation:       1.1,
		EventCount:            5,
		OTPVerified:           true,
	}

	seq := sequence.SequenceResult{
		MatchedPattern: sequence.PatternLegitimateFlow,
	}

	pred := clf.Predict(trajectory, fs, seq, b)

	if pred.DominantIntent != contracts.IntentLegitimate {
		t.Fatalf("expected dominant legitimate, got %s", pred.DominantIntent)
	}

	// Crucial false-positive safety check: ATO probability must be near zero
	atoProb := pred.Probabilities[contracts.IntentAccountTakeover]
	if atoProb > 0.05 {
		t.Errorf("false alarm risk: ATO probability too high (%f) on clean legitimate traffic", atoProb)
	}

	if pred.Probabilities[contracts.IntentLegitimate] < 0.85 {
		t.Errorf("expected legitimate confidence >= 0.85, got %f", pred.Probabilities[contracts.IntentLegitimate])
	}
}

func TestClassifierAccountTakeoverCatch(t *testing.T) {
	clf := NewClassifier()
	b := baseline.DefaultBaseline("user_001")

	now := time.Now().UTC()
	trajectory := []contracts.Event{
		{Type: contracts.EventDeviceChanged, Timestamp: now.Unix()},
		{Type: contracts.EventPasswordChanged, Timestamp: now.Add(2 * time.Second).Unix()},
		{Type: contracts.EventBeneficiaryCreated, Timestamp: now.Add(4 * time.Second).Unix()},
		{Type: contracts.EventAmountEntered, Timestamp: now.Add(6 * time.Second).Unix()},
	}

	fs := features.FeatureSet{
		DeviceKnown:             false,
		DeviceChangedRecent:     true,
		CredentialChangeRecent:  true,
		CredentialChangeSeconds: 4.0,
		IsNewBeneficiary:        true,
		AmountDeviation:         7.5,
		RapidSequence:           true,
		EventCount:              4,
	}

	seq := sequence.SequenceResult{
		MatchedPattern: sequence.PatternAccountTakeover,
	}

	pred := clf.Predict(trajectory, fs, seq, b)

	if pred.DominantIntent != contracts.IntentAccountTakeover {
		t.Fatalf("expected dominant account_takeover, got %s", pred.DominantIntent)
	}

	if pred.Probabilities[contracts.IntentAccountTakeover] < 0.90 {
		t.Errorf("expected ATO probability >= 0.90, got %f", pred.Probabilities[contracts.IntentAccountTakeover])
	}
}

func TestClassifierSocialEngineeringWithProbe(t *testing.T) {
	clf := NewClassifier()
	b := baseline.DefaultBaseline("user_001")

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
		AmountDeviation:          6.0,
		OTPVerified:              true,
		ProbeResponded:           true,
		ProbeImpersonationSignal: true,
		EventCount:               5,
	}

	seq := sequence.SequenceResult{
		MatchedPattern: sequence.PatternSocialEngineering,
	}

	pred := clf.Predict(trajectory, fs, seq, b)

	if pred.DominantIntent != contracts.IntentSocialEngineering {
		t.Fatalf("expected dominant social_engineering, got %s", pred.DominantIntent)
	}

	if pred.Probabilities[contracts.IntentSocialEngineering] < 0.85 {
		t.Errorf("expected SocEng probability >= 0.85, got %f", pred.Probabilities[contracts.IntentSocialEngineering])
	}
}

func BenchmarkClassifierInferenceLatency(b *testing.B) {
	clf := NewClassifier()
	base := baseline.DefaultBaseline("user_001")
	now := time.Now().UTC()
	trajectory := []contracts.Event{
		{Type: contracts.EventLogin, Timestamp: now.Unix()},
		{Type: contracts.EventAmountEntered, Timestamp: now.Add(5 * time.Second).Unix()},
	}
	fs := features.FeatureSet{
		DeviceKnown:           true,
		BeneficiarySeenBefore: true,
		AmountDeviation:       1.0,
	}
	seq := sequence.SequenceResult{MatchedPattern: sequence.PatternLegitimateFlow}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = clf.Predict(trajectory, fs, seq, base)
	}
}
