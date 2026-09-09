package explanation

import (
	"strings"
	"testing"

	"github.com/holiday-heartbreaks/parallax/backend/internal/features"
	"github.com/holiday-heartbreaks/parallax/backend/internal/intent"
	"github.com/holiday-heartbreaks/parallax/backend/internal/policy"
	"github.com/holiday-heartbreaks/parallax/backend/internal/uncertainty"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

func TestExplainAllow(t *testing.T) {
	eng := NewEngine()

	dec := policy.Decision{
		Action:          contracts.ActionAllow,
		ConfidenceLevel: "HIGH",
	}
	inf := intent.InferenceResult{
		DominantIntent: contracts.IntentLegitimate,
		DominantScore:  0.94,
		Evidence:       []string{"Telemetry matches baseline"},
	}
	u := uncertainty.Result{Confidence: 0.94}
	fs := features.FeatureSet{DeviceKnown: true}

	summary := eng.Explain(dec, inf, u, fs)
	if !strings.Contains(summary.Narrative, "approved") {
		t.Errorf("expected narrative to mention approved")
	}
	if summary.Action != contracts.ActionAllow {
		t.Errorf("expected ActionAllow")
	}
	if len(summary.EvidenceBullets) == 0 {
		t.Errorf("expected evidence bullets")
	}
}

func TestExplainBlockATO(t *testing.T) {
	eng := NewEngine()

	dec := policy.Decision{
		Action:          contracts.ActionBlock,
		ConfidenceLevel: "HIGH",
	}
	inf := intent.InferenceResult{
		DominantIntent: contracts.IntentAccountTakeover,
		DominantScore:  0.96,
		Evidence:       []string{"Unverified device", "Credential reset"},
	}
	u := uncertainty.Result{Confidence: 0.96}
	fs := features.FeatureSet{DeviceKnown: false}

	summary := eng.Explain(dec, inf, u, fs)
	if !strings.Contains(summary.Narrative, "account takeover") {
		t.Errorf("expected narrative to explain account takeover")
	}
	if summary.Action != contracts.ActionBlock {
		t.Errorf("expected ActionBlock")
	}
}
