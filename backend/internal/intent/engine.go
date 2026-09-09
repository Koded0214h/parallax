package intent

import (
	"math"

	"github.com/holiday-heartbreaks/parallax/backend/internal/baseline"
	"github.com/holiday-heartbreaks/parallax/backend/internal/features"
	"github.com/holiday-heartbreaks/parallax/backend/internal/sequence"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

// InferenceResult represents the synthesized intent distribution and risk metrics (prd.md §7, §21.5).
type InferenceResult struct {
	Hypotheses     map[string]float64 `json:"hypotheses"`
	DominantIntent string             `json:"dominant_intent"`
	DominantScore  float64            `json:"dominant_score"`
	RiskScore      float64            `json:"risk_score"`
	Evidence       []string           `json:"evidence"`
}

// Engine combines behavioural, sequence, and contextual evidence into calibrated intent hypotheses.
type Engine struct{}

// NewEngine constructs a new Intent Engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Infer generates calibrated intent hypotheses from extracted features and sequence analysis.
func (e *Engine) Infer(fs features.FeatureSet, seq sequence.SequenceResult, b baseline.UserBaseline) InferenceResult {
	// Raw unnormalized logits
	var (
		scoreLegit   = 1.0
		scoreAccid   = 0.2
		scoreSocEng  = 0.2
		scoreATO     = 0.2
	)

	evidence := features.GenerateEvidenceItems(fs)
	for _, anomaly := range seq.TransitionAnomalies {
		evidence = append(evidence, anomaly)
	}

	// 1. Evaluate Account Takeover indicators
	if !fs.DeviceKnown && fs.DeviceID != "" {
		scoreATO += 4.0
		scoreLegit -= 1.0
	}
	if fs.DeviceChangedRecent {
		scoreATO += 3.5
		scoreLegit -= 1.0
	}
	if fs.CredentialChangeRecent {
		scoreATO += 4.5
		scoreLegit -= 1.5
	}
	if fs.RapidSequence && fs.AmountDeviation > 2.0 {
		scoreATO += 3.0
	}
	if seq.MatchedPattern == sequence.PatternAccountTakeover {
		scoreATO += 8.0
		scoreLegit = 0.05
		scoreAccid = 0.05
	}

	// 2. Evaluate Social Engineering indicators
	if fs.DeviceKnown && fs.IsNewBeneficiary && fs.AmountDeviation >= 2.0 {
		scoreSocEng += 3.0
		// Because device is known and credentials worked, legitimate is still plausible
		scoreLegit += 1.0
	}
	if fs.ProbeResponded {
		if fs.ProbeImpersonationSignal {
			scoreSocEng += 8.0
			scoreLegit = 0.1
			scoreATO = 0.1
			scoreAccid = 0.05
		}
		if fs.ProbeUrgencySignal {
			scoreSocEng += 3.0
		}
	} else if seq.MatchedPattern == sequence.PatternSocialEngineering {
		scoreSocEng += 4.0
	}

	// 3. Evaluate Accidental Transfer indicators
	if fs.ZeroPaddingAnomaly && fs.BeneficiarySeenBefore {
		scoreAccid += 6.0
		scoreLegit += 0.5
		scoreATO = 0.1
	} else if fs.AmountDeviation >= 5.0 && fs.BeneficiarySeenBefore && fs.DeviceKnown && !fs.DeviceChangedRecent {
		scoreAccid += 2.5
	}

	// 4. Evaluate Legitimate flow indicators
	if fs.DeviceKnown && !fs.DeviceChangedRecent && !fs.CredentialChangeRecent {
		if fs.BeneficiarySeenBefore && fs.AmountDeviation <= 1.5 && !fs.IsAtypicalHour {
			scoreLegit += 7.0
			scoreATO = 0.05
			scoreSocEng = 0.05
			scoreAccid = 0.05
		} else if fs.AmountDeviation <= 2.0 {
			scoreLegit += 3.0
		}
	}

	// Incomplete / Early session dampening (only when no critical anomalies observed)
	if fs.EventCount <= 2 && fs.Amount == 0 && !fs.DeviceChangedRecent && !fs.CredentialChangeRecent && !fs.RapidSequence && seq.MatchedPattern != sequence.PatternAccountTakeover {
		// Session just started peacefully, keep balanced
		scoreLegit = 2.0
		scoreAccid = 0.5
		scoreSocEng = 0.5
		scoreATO = 0.5
	}

	// Softmax normalization to obtain calibrated probabilities
	hypotheses := softmax(map[string]float64{
		contracts.IntentLegitimate:        math.Max(scoreLegit, 0.01),
		contracts.IntentAccidental:        math.Max(scoreAccid, 0.01),
		contracts.IntentSocialEngineering: math.Max(scoreSocEng, 0.01),
		contracts.IntentAccountTakeover:   math.Max(scoreATO, 0.01),
	})

	dominantIntent := contracts.IntentLegitimate
	dominantScore := 0.0
	for k, v := range hypotheses {
		if v > dominantScore {
			dominantScore = v
			dominantIntent = k
		}
	}

	// Risk score calculation: how dangerous the situation is (separate from uncertainty)
	riskScore := (hypotheses[contracts.IntentAccountTakeover] * 1.0) +
		(hypotheses[contracts.IntentSocialEngineering] * 0.85) +
		(hypotheses[contracts.IntentAccidental] * 0.40) +
		(hypotheses[contracts.IntentLegitimate] * 0.05)

	if riskScore > 1.0 {
		riskScore = 1.0
	} else if riskScore < 0.0 {
		riskScore = 0.0
	}

	return InferenceResult{
		Hypotheses:     hypotheses,
		DominantIntent: dominantIntent,
		DominantScore:  round4(dominantScore),
		RiskScore:      round4(riskScore),
		Evidence:       evidence,
	}
}

func softmax(logits map[string]float64) map[string]float64 {
	var maxLogit float64 = -math.MaxFloat64
	for _, val := range logits {
		if val > maxLogit {
			maxLogit = val
		}
	}

	var sumExp float64
	expValues := make(map[string]float64, len(logits))
	for k, val := range logits {
		e := math.Exp(val - maxLogit)
		expValues[k] = e
		sumExp += e
	}

	result := make(map[string]float64, len(logits))
	var total float64
	for k, e := range expValues {
		p := e / sumExp
		result[k] = round4(p)
		total += result[k]
	}

	// Ensure sum equals 1.0 exactly by adjusting largest
	diff := 1.0 - total
	if math.Abs(diff) > 1e-6 {
		var maxKey string
		var maxP float64
		for k, v := range result {
			if v > maxP {
				maxP = v
				maxKey = k
			}
		}
		if maxKey != "" {
			result[maxKey] = round4(result[maxKey] + diff)
		}
	}

	return result
}

func round4(val float64) float64 {
	return math.Round(val*10000) / 10000
}
