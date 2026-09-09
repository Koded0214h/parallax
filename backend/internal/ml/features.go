package ml

import (
	"math"

	"github.com/holiday-heartbreaks/parallax/backend/internal/features"
	"github.com/holiday-heartbreaks/parallax/backend/internal/sequence"
)

// FeatureVector represents numerical features passed to the decision ensemble.
type FeatureVector struct {
	AmountRatio              float64
	AmountZScore             float64
	ZeroPaddingAnomaly       float64
	IsNewBeneficiary         float64
	BeneficiaryAgeSecLog     float64
	BeneficiarySeenBefore    float64
	DeviceKnown              float64
	DeviceChangedRecent      float64
	CredentialChangeRecent   float64
	CredentialChangeSecLog   float64
	RapidSequence            float64
	IsAtypicalHour           float64
	InteractionSpeedRatio    float64
	SessionDurationSecLog    float64
	EventCount               float64
	FailedAttemptsCount      float64
	OTPVerified              float64
	ProbeResponded           float64
	ProbeImpersonationSignal float64
	ProbeUrgencySignal       float64
	MarkovAnomalyScore       float64
}

// ToVector extracts and normalizes the numerical feature vector from domain features.
func ToVector(fs features.FeatureSet, seq sequence.SequenceResult, markovScore float64) FeatureVector {
	var benAgeLog float64
	if fs.BeneficiaryAgeSeconds > 0 {
		benAgeLog = math.Log1p(fs.BeneficiaryAgeSeconds)
	}

	var credAgeLog float64
	if fs.CredentialChangeSeconds > 0 {
		credAgeLog = math.Log1p(fs.CredentialChangeSeconds)
	}

	var sessDurLog float64
	if fs.SessionDurationSeconds > 0 {
		sessDurLog = math.Log1p(fs.SessionDurationSeconds)
	}

	return FeatureVector{
		AmountRatio:              fs.AmountDeviation,
		AmountZScore:             fs.AmountZScore,
		ZeroPaddingAnomaly:       boolToFloat(fs.ZeroPaddingAnomaly),
		IsNewBeneficiary:         boolToFloat(fs.IsNewBeneficiary),
		BeneficiaryAgeSecLog:     benAgeLog,
		BeneficiarySeenBefore:    boolToFloat(fs.BeneficiarySeenBefore),
		DeviceKnown:              boolToFloat(fs.DeviceKnown),
		DeviceChangedRecent:      boolToFloat(fs.DeviceChangedRecent),
		CredentialChangeRecent:   boolToFloat(fs.CredentialChangeRecent),
		CredentialChangeSecLog:   credAgeLog,
		RapidSequence:            boolToFloat(fs.RapidSequence),
		IsAtypicalHour:           boolToFloat(fs.IsAtypicalHour),
		InteractionSpeedRatio:    fs.InteractionSpeedDeviation,
		SessionDurationSecLog:    sessDurLog,
		EventCount:               float64(fs.EventCount),
		FailedAttemptsCount:      float64(fs.FailedAttemptsCount),
		OTPVerified:              boolToFloat(fs.OTPVerified),
		ProbeResponded:           boolToFloat(fs.ProbeResponded),
		ProbeImpersonationSignal: boolToFloat(fs.ProbeImpersonationSignal),
		ProbeUrgencySignal:       boolToFloat(fs.ProbeUrgencySignal),
		MarkovAnomalyScore:       markovScore,
	}
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}
