package uncertainty

import (
	"math"
	"sort"

	"github.com/holiday-heartbreaks/parallax/backend/internal/features"
	"github.com/holiday-heartbreaks/parallax/backend/internal/intent"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

// Result encapsulates uncertainty metrics (prd.md §8, §21.6).
type Result struct {
	Uncertainty float64  `json:"uncertainty"`
	Confidence  float64  `json:"confidence"`
	Entropy     float64  `json:"entropy"`
	Reasons     []string `json:"reasons"`
}

// Engine evaluates evidence ambiguity, entropy, and conflicting signals.
type Engine struct{}

// NewEngine creates an uncertainty engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Evaluate computes the uncertainty score given the inference result and feature set.
func (e *Engine) Evaluate(inf intent.InferenceResult, fs features.FeatureSet) Result {
	var reasons []string

	// 1. Normalized Shannon Entropy across the 4 intent classes
	var entropySum float64
	var probs []float64
	for _, p := range inf.Hypotheses {
		probs = append(probs, p)
		if p > 1e-6 {
			entropySum -= p * math.Log2(p)
		}
	}
	// Max entropy for 4 classes is log2(4) = 2.0
	normalizedEntropy := entropySum / 2.0
	if normalizedEntropy > 1.0 {
		normalizedEntropy = 1.0
	} else if normalizedEntropy < 0.0 {
		normalizedEntropy = 0.0
	}

	// 2. Margin between top 1 and top 2 hypotheses
	sort.Float64s(probs)
	var margin float64 = 1.0
	if len(probs) >= 2 {
		margin = probs[len(probs)-1] - probs[len(probs)-2]
	}

	// 3. Detect conflicting evidence
	// Conflict: Valid device & OTP vs new beneficiary & large amount (unprobed)
	hasConflictingSignals := false
	if fs.DeviceKnown && fs.OTPVerified && fs.IsNewBeneficiary && fs.AmountDeviation >= 2.0 && !fs.ProbeResponded {
		hasConflictingSignals = true
		reasons = append(reasons, "Conflicting evidence: known device and valid credentials conflict with anomalous recipient and amount")
	}

	// Conflict: High amount deviation on habitual beneficiary without double-zero pattern
	if fs.BeneficiarySeenBefore && fs.AmountDeviation >= 4.0 && !fs.ZeroPaddingAnomaly && !fs.ProbeResponded {
		hasConflictingSignals = true
		reasons = append(reasons, "Ambiguous amount deviation to habitual recipient without verified context")
	}

	// 4. Insufficient evidence
	isInsufficient := false
	if fs.EventCount <= 2 && fs.Amount == 0 {
		isInsufficient = true
		reasons = append(reasons, "Insufficient session evidence: session in preliminary state without financial authorization")
	}

	// 5. Combine into final uncertainty score
	var u float64
	if isInsufficient {
		u = 0.85
	} else if hasConflictingSignals {
		// Blend entropy and small margin
		u = math.Max(0.50, normalizedEntropy*0.7+(1.0-margin)*0.3)
	} else if fs.ProbeResponded {
		// Additional user context directly reduces uncertainty (prd.md §15)
		u = math.Min(0.25, normalizedEntropy*0.4)
	} else {
		u = normalizedEntropy*0.6 + (1.0-margin)*0.4
	}

	if u > 1.0 {
		u = 1.0
	} else if u < 0.02 {
		u = 0.02
	}

	u = round4(u)
	conf := round4(1.0 - u)

	if len(reasons) == 0 {
		if conf >= 0.75 {
			reasons = append(reasons, "High classification confidence with definitive behavioural alignment")
		} else {
			reasons = append(reasons, "Moderate distribution variance across competing intent classes")
		}
	}

	return Result{
		Uncertainty: u,
		Confidence:  conf,
		Entropy:     round4(normalizedEntropy),
		Reasons:     reasons,
	}
}

// IsProbeRequired reports whether uncertainty warrants triggering an intent probe (prd.md §15, §16).
func (e *Engine) IsProbeRequired(u Result, inf intent.InferenceResult, fs features.FeatureSet) bool {
	if fs.ProbeResponded {
		return false
	}
	// Moderate to high risk + moderate to high uncertainty
	if inf.RiskScore >= 0.35 && u.Uncertainty >= 0.40 {
		// Specifically when social engineering or accidental is suspected but unconfirmed
		if inf.DominantIntent == contracts.IntentSocialEngineering ||
			inf.DominantIntent == contracts.IntentAccidental ||
			(inf.Hypotheses[contracts.IntentSocialEngineering] >= 0.30) {
			return true
		}
	}
	return false
}

func round4(val float64) float64 {
	return math.Round(val*10000) / 10000
}
