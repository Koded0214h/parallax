package explanation

import (
	"fmt"
	"strings"

	"github.com/holiday-heartbreaks/parallax/backend/internal/features"
	"github.com/holiday-heartbreaks/parallax/backend/internal/intent"
	"github.com/holiday-heartbreaks/parallax/backend/internal/policy"
	"github.com/holiday-heartbreaks/parallax/backend/internal/uncertainty"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

// Summary encapsulates human-readable explanations and structured justifications (prd.md §17, §21.9).
type Summary struct {
	Narrative         string           `json:"narrative"`
	EvidenceBullets   []string         `json:"evidence_bullets"`
	Action            contracts.Action `json:"action"`
	ActionSummary     string           `json:"action_summary"`
	PrimaryHypothesis string           `json:"primary_hypothesis"`
	Confidence        string           `json:"confidence"`
}

// Engine synthesizes structured evidence into factual, transparent explanations.
type Engine struct{}

// NewEngine constructs an explanation engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Explain produces an end-to-end factual explanation for the system's decision.
func (e *Engine) Explain(
	dec policy.Decision,
	inf intent.InferenceResult,
	u uncertainty.Result,
	fs features.FeatureSet,
) Summary {
	evidenceBullets := inf.Evidence
	if len(evidenceBullets) == 0 {
		evidenceBullets = features.GenerateEvidenceItems(fs)
	}

	primaryHypothesis := fmt.Sprintf("%s (%.0f%%)",
		humanizeIntent(inf.DominantIntent),
		inf.DominantScore*100,
	)

	var (
		actionSummary string
		narrativeParts []string
	)

	switch dec.Action {
	case contracts.ActionAllow:
		actionSummary = "Transaction approved"
		narrativeParts = append(narrativeParts,
			"This transaction was approved because the session interaction, payee, and device align with normal customer history.",
		)

	case contracts.ActionBlock:
		actionSummary = "Transaction blocked"
		if inf.DominantIntent == contracts.IntentAccountTakeover {
			narrativeParts = append(narrativeParts,
				"This transaction was blocked because a critical account takeover pattern was detected: an unverified device made rapid security credential alterations followed immediately by a high-value withdrawal request.",
			)
		} else if inf.DominantIntent == contracts.IntentSocialEngineering {
			narrativeParts = append(narrativeParts,
				"This transaction was blocked to protect account funds: contextual verification confirmed that this transfer was initiated under external authority impersonation or social engineering pressure.",
			)
		} else {
			narrativeParts = append(narrativeParts,
				"This transaction was blocked due to critical risk indicators exceeding safety tolerances.",
			)
		}

	case contracts.ActionProbe:
		actionSummary = "Context verification required"
		narrativeParts = append(narrativeParts,
			"This transaction has been paused for an intent probe: while authentic credentials were used, the transfer details deviate significantly from normal activity, creating high uncertainty between legitimate intent and social engineering.",
		)

	case contracts.ActionVerify:
		actionSummary = "Step-up verification required"
		if fs.ZeroPaddingAnomaly {
			narrativeParts = append(narrativeParts,
				"Step-up authorization requested: the transfer amount is roughly 10x higher than typical transfers to this contact, indicating a probable accidental typo.",
			)
		} else {
			narrativeParts = append(narrativeParts,
				"Step-up verification requested due to moderate anomaly indicators in current session parameters.",
			)
		}

	case contracts.ActionEscalate:
		actionSummary = "Escalated for fraud operations review"
		narrativeParts = append(narrativeParts,
			"This session has been routed for manual inspection by fraud operations due to conflicting behavioral signals and elevated risk.",
		)
	}

	narrative := strings.Join(narrativeParts, " ")

	return Summary{
		Narrative:         narrative,
		EvidenceBullets:   evidenceBullets,
		Action:            dec.Action,
		ActionSummary:     actionSummary,
		PrimaryHypothesis: primaryHypothesis,
		Confidence:        dec.ConfidenceLevel,
	}
}

func humanizeIntent(intentLabel string) string {
	switch intentLabel {
	case contracts.IntentLegitimate:
		return "Legitimate"
	case contracts.IntentAccountTakeover:
		return "Account Takeover"
	case contracts.IntentSocialEngineering:
		return "Social Engineering"
	case contracts.IntentAccidental:
		return "Accidental"
	default:
		return intentLabel
	}
}
