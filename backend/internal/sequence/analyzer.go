package sequence

import (
	"fmt"

	"github.com/holiday-heartbreaks/parallax/backend/internal/features"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

// PatternType describes identified behavioural sequence motifs.
type PatternType string

const (
	PatternLegitimateFlow      PatternType = "LEGITIMATE_FLOW"
	PatternAccountTakeover     PatternType = "ACCOUNT_TAKEOVER_CHAIN"
	PatternSocialEngineering   PatternType = "MANIPULATED_AUTHORIZATION"
	PatternAccidentalTransfer  PatternType = "ACCIDENTAL_ENTRY"
	PatternAmbiguousIncomplete PatternType = "INCOMPLETE_SESSION"
)

// SequenceResult encapsulates structural sequence evaluation (prd.md §21.4).
type SequenceResult struct {
	MatchedPattern      PatternType `json:"matched_pattern"`
	SequenceRiskScore   float64     `json:"sequence_risk_score"`
	ATOPatternScore     float64     `json:"ato_pattern_score"`
	SocEngPatternScore  float64     `json:"soc_eng_pattern_score"`
	LegitPatternScore   float64     `json:"legit_pattern_score"`
	AccidentScore       float64     `json:"accident_score"`
	TransitionAnomalies []string    `json:"transition_anomalies"`
}

// Analyzer evaluates event transitions, ordering, and timing across trajectories.
type Analyzer struct{}

// NewAnalyzer creates a new sequence analyzer.
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// Analyze processes an ordered trajectory and feature set into sequence evidence.
func (a *Analyzer) Analyze(trajectory []contracts.Event, fs features.FeatureSet) SequenceResult {
	res := SequenceResult{
		MatchedPattern: PatternAmbiguousIncomplete,
	}

	n := len(trajectory)
	if n < 2 {
		res.LegitPatternScore = 0.5
		res.SequenceRiskScore = 0.1
		return res
	}

	var (
		hasDeviceChange    bool
		hasCredChange      bool
		hasBenCreate       bool
		hasAmountEnter     bool
		hasOTPVerify       bool
		deviceChangeIndex  = -1
		credChangeIndex    = -1
		benCreateIndex     = -1
		amountEnterIndex   = -1
	)

	for i, ev := range trajectory {
		switch ev.Type {
		case contracts.EventDeviceChanged:
			hasDeviceChange = true
			if deviceChangeIndex == -1 {
				deviceChangeIndex = i
			}
		case contracts.EventPasswordChanged, contracts.EventPINChanged:
			hasCredChange = true
			if credChangeIndex == -1 {
				credChangeIndex = i
			}
		case contracts.EventBeneficiaryCreated:
			hasBenCreate = true
			if benCreateIndex == -1 {
				benCreateIndex = i
			}
		case contracts.EventAmountEntered, contracts.EventTransferStarted:
			hasAmountEnter = true
			if amountEnterIndex == -1 {
				amountEnterIndex = i
			}
		case contracts.EventOTPVerified:
			hasOTPVerify = true
		}
	}

	// Detect Account Takeover Motif:
	// DEVICE_CHANGE -> PASSWORD/PIN_CHANGE -> BENEFICIARY_CREATED -> TRANSFER
	if hasDeviceChange && hasCredChange && hasBenCreate && hasAmountEnter {
		if deviceChangeIndex <= credChangeIndex && credChangeIndex <= benCreateIndex && benCreateIndex <= amountEnterIndex {
			res.MatchedPattern = PatternAccountTakeover
			res.ATOPatternScore = 0.95
			res.SequenceRiskScore = 0.95
			res.TransitionAnomalies = append(res.TransitionAnomalies,
				"Classic account takeover sequence detected: device alteration precedes credential reset and immediate beneficiary drain")
			return res
		}
	}

	// Partial ATO sequence (e.g. Device change + Cred change + Transfer)
	if (hasDeviceChange && hasCredChange) || (!fs.DeviceKnown && hasBenCreate && fs.AmountDeviation > 3.0) {
		res.ATOPatternScore = 0.80
		res.SequenceRiskScore = 0.85
		res.TransitionAnomalies = append(res.TransitionAnomalies,
			"High-risk sequence: unauthenticated device change coupled with security credential modification")
		if res.MatchedPattern == PatternAmbiguousIncomplete {
			res.MatchedPattern = PatternAccountTakeover
		}
	}

	// Detect Social Engineering Motif:
	// Known device, authenticated user, newly added beneficiary, abnormally high amount, valid OTP,
	// and probe responses indicating external direction.
	if fs.DeviceKnown && hasOTPVerify && fs.IsNewBeneficiary && fs.AmountDeviation >= 2.5 {
		res.SocEngPatternScore = 0.75
		if fs.ProbeResponded && (fs.ProbeImpersonationSignal || fs.ProbeUrgencySignal) {
			res.SocEngPatternScore = 0.95
			res.SequenceRiskScore = 0.90
			res.MatchedPattern = PatternSocialEngineering
			res.TransitionAnomalies = append(res.TransitionAnomalies,
				"Social engineering pattern confirmed: authorized credentials used under external impersonation instruction")
			return res
		}

		res.SequenceRiskScore = 0.65
		res.MatchedPattern = PatternSocialEngineering
		res.TransitionAnomalies = append(res.TransitionAnomalies,
			"Legitimate device and OTP paired with urgent transfer to freshly introduced recipient")
	}

	// Detect Accidental Transfer Motif:
	// Known device, known beneficiary, valid authentication, but high amount multiplier (e.g. zero-padding error)
	if fs.DeviceKnown && fs.BeneficiarySeenBefore && fs.ZeroPaddingAnomaly {
		res.AccidentScore = 0.85
		res.SequenceRiskScore = 0.40
		res.MatchedPattern = PatternAccidentalTransfer
		res.TransitionAnomalies = append(res.TransitionAnomalies,
			fmt.Sprintf("Accidental magnitude anomaly: transfer to habitual contact is exactly %.0fx typical amount", fs.AmountDeviation))
		return res
	}

	// Legitimate Flow Motif
	if fs.DeviceKnown && !hasDeviceChange && !hasCredChange && fs.AmountDeviation <= 2.0 {
		res.LegitPatternScore = 0.90
		res.SequenceRiskScore = 0.05
		res.MatchedPattern = PatternLegitimateFlow
		return res
	}

	// Default scoring if ambiguous
	if res.MatchedPattern == PatternAmbiguousIncomplete {
		res.LegitPatternScore = 0.40
		res.SocEngPatternScore = 0.20
		res.ATOPatternScore = 0.20
		res.AccidentScore = 0.20
		res.SequenceRiskScore = 0.30
	}

	return res
}
