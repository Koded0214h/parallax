package policy

import (
	"github.com/holiday-heartbreaks/parallax/backend/internal/features"
	"github.com/holiday-heartbreaks/parallax/backend/internal/intent"
	"github.com/holiday-heartbreaks/parallax/backend/internal/uncertainty"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

// Decision represents the actionable enforcement output from the policy engine (prd.md §16, §21.8).
type Decision struct {
	Action          contracts.Action `json:"action"`
	Reason          string           `json:"reason"`
	RiskLevel       string           `json:"risk_level"`
	ConfidenceLevel string           `json:"confidence_level"`
	RequiresProbe   bool             `json:"requires_probe"`
	EscalateToOps   bool             `json:"escalate_to_ops"`
}

// Config allows tuning policy thresholds independently from models.
type Config struct {
	AllowMaxRisk       float64
	AllowMaxUncert     float64
	ProbeMinRisk       float64
	ProbeMinUncert     float64
	BlockMinRisk       float64
	BlockMinConfidence float64
}

// DefaultConfig provides balanced thresholds for real-time payments.
func DefaultConfig() Config {
	return Config{
		AllowMaxRisk:       0.25,
		AllowMaxUncert:     0.35,
		ProbeMinRisk:       0.35,
		ProbeMinUncert:     0.40,
		BlockMinRisk:       0.70,
		BlockMinConfidence: 0.70,
	}
}

// Engine maps intent distributions and uncertainty metrics to proportional actions.
type Engine struct {
	cfg Config
}

// NewEngine constructs a policy engine with configuration.
func NewEngine(cfg Config) *Engine {
	return &Engine{cfg: cfg}
}

// Decide determines the enforcement action for a transaction.
func (e *Engine) Decide(inf intent.InferenceResult, u uncertainty.Result, fs features.FeatureSet) Decision {
	var (
		riskLevel string
		confLevel string
	)

	// Determine qualitative risk level
	switch {
	case inf.RiskScore >= 0.75:
		riskLevel = "CRITICAL"
	case inf.RiskScore >= 0.50:
		riskLevel = "HIGH"
	case inf.RiskScore >= 0.25:
		riskLevel = "MODERATE"
	default:
		riskLevel = "LOW"
	}

	// Determine qualitative confidence level
	switch {
	case u.Confidence >= 0.75:
		confLevel = "HIGH"
	case u.Confidence >= 0.50:
		confLevel = "MODERATE"
	default:
		confLevel = "LOW"
	}

	// Rule 1: High Risk + Strong Account Takeover Evidence => BLOCK
	if inf.DominantIntent == contracts.IntentAccountTakeover && inf.DominantScore >= e.cfg.BlockMinConfidence {
		return Decision{
			Action:          contracts.ActionBlock,
			Reason:          "Unambiguous account takeover sequence and unverified device mutation",
			RiskLevel:       riskLevel,
			ConfidenceLevel: confLevel,
			EscalateToOps:   true,
		}
	}

	// Rule 2: Confirmed Social Engineering via Probe => BLOCK / INTERVENE
	if inf.DominantIntent == contracts.IntentSocialEngineering && fs.ProbeResponded && (fs.ProbeImpersonationSignal || fs.ProbeUrgencySignal) {
		return Decision{
			Action:          contracts.ActionBlock,
			Reason:          "Payment blocked: external impersonation and coercive fraud confirmed by context probe",
			RiskLevel:       riskLevel,
			ConfidenceLevel: confLevel,
			EscalateToOps:   true,
		}
	}

	// Rule 3: Moderate/High Risk + High Uncertainty + Unprobed => INTENT PROBE
	if inf.RiskScore >= e.cfg.ProbeMinRisk && u.Uncertainty >= e.cfg.ProbeMinUncert && !fs.ProbeResponded {
		return Decision{
			Action:          contracts.ActionProbe,
			Reason:          "High intent ambiguity: authorized credentials accompanied by anomalous recipient and amount",
			RiskLevel:       riskLevel,
			ConfidenceLevel: confLevel,
			RequiresProbe:   true,
		}
	}

	// Rule 4: Accidental magnitude anomaly (e.g. 10x extra zeroes to habitual beneficiary) => VERIFY
	if inf.DominantIntent == contracts.IntentAccidental || fs.ZeroPaddingAnomaly {
		return Decision{
			Action:          contracts.ActionVerify,
			Reason:          "Potential accidental magnitude deviation: step-up confirmation required for unusual amount to trusted contact",
			RiskLevel:       riskLevel,
			ConfidenceLevel: confLevel,
		}
	}

	// Rule 5: Conflicting Evidence or High Risk + High Uncertainty => ESCALATE
	if inf.RiskScore >= 0.60 && u.Uncertainty >= 0.50 {
		return Decision{
			Action:          contracts.ActionEscalate,
			Reason:          "Severely contradictory risk indicators requiring manual fraud operations review",
			RiskLevel:       riskLevel,
			ConfidenceLevel: confLevel,
			EscalateToOps:   true,
		}
	}

	// Rule 6: Low Risk + High Confidence => ALLOW
	if inf.RiskScore <= e.cfg.AllowMaxRisk && u.Uncertainty <= e.cfg.AllowMaxUncert {
		return Decision{
			Action:          contracts.ActionAllow,
			Reason:          "Telemetry, beneficiary, and transaction parameters align with customer historical baseline",
			RiskLevel:       riskLevel,
			ConfidenceLevel: confLevel,
		}
	}

	// Rule 7: Moderate risk with moderate confidence but no probe match => VERIFY
	if inf.RiskScore > e.cfg.AllowMaxRisk {
		return Decision{
			Action:          contracts.ActionVerify,
			Reason:          "Moderate anomaly detected: supplemental secondary authorization required",
			RiskLevel:       riskLevel,
			ConfidenceLevel: confLevel,
		}
	}

	// Default fallback: ALLOW
	return Decision{
		Action:          contracts.ActionAllow,
		Reason:          "No high-risk indicators observed in active session",
		RiskLevel:       riskLevel,
		ConfidenceLevel: confLevel,
	}
}
