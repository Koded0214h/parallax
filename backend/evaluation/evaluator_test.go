package evaluation

import (
	"testing"

	"github.com/holiday-heartbreaks/parallax/backend/simulator"
)

func TestEvaluationMetricsAndFalseAlarmAudit(t *testing.T) {
	gen := simulator.NewGenerator(42)
	// Generate a representative batch: 200 legit, 50 ATO, 50 SocEng, 50 Accidental
	dataset := gen.GenerateDataset(200, 50, 50, 50)

	evaluator := NewEvaluator()
	summary := evaluator.Evaluate(dataset)

	t.Logf("\n%s", summary.PrintReport())

	// Proof assertion 1: Overall accuracy should exceed 95%
	if summary.OverallAccuracy < 0.95 {
		t.Errorf("expected overall accuracy >= 0.95, got %f", summary.OverallAccuracy)
	}

	// Proof assertion 2: ATO catch rate (recall) should be >= 95%
	if summary.ATOCatchRate < 0.95 {
		t.Errorf("expected ATO catch rate >= 0.95, got %f", summary.ATOCatchRate)
	}

	// Proof assertion 3: Social Engineering catch rate (recall) should be >= 95%
	if summary.SocEngCatchRate < 0.95 {
		t.Errorf("expected SocEng catch rate >= 0.95, got %f", summary.SocEngCatchRate)
	}

	// Proof assertion 4: Bank false alarm safety — Legitimate FPR must be < 2%
	if summary.LegitimateFPR > 0.02 {
		t.Errorf("false alarm rate too high on legitimate customers: got %f (must be < 0.02)", summary.LegitimateFPR)
	}
}
