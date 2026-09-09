package evaluation

import (
	"context"
	"fmt"
	"math"

	"github.com/holiday-heartbreaks/parallax/backend/internal/baseline"
	"github.com/holiday-heartbreaks/parallax/backend/internal/engine"
	"github.com/holiday-heartbreaks/parallax/backend/internal/session"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
	"github.com/holiday-heartbreaks/parallax/backend/simulator"
)

// ClassMetrics reports classification performance for a specific intent class.
type ClassMetrics struct {
	Class              string  `json:"class"`
	TruePositives      int     `json:"true_positives"`
	FalsePositives     int     `json:"false_positives"`
	TrueNegatives      int     `json:"true_negatives"`
	FalseNegatives     int     `json:"false_negatives"`
	Precision          float64 `json:"precision"`
	Recall             float64 `json:"recall"`
	F1                 float64 `json:"f1"`
	FalsePositiveRate  float64 `json:"false_positive_rate"`
}

// Summary aggregates end-to-end evaluation results (prd.md §27).
type Summary struct {
	TotalSessions             int                     `json:"total_sessions"`
	ConfusionMatrix           map[string]map[string]int `json:"confusion_matrix"`
	MetricsByClass            map[string]ClassMetrics `json:"metrics_by_class"`
	OverallAccuracy           float64                 `json:"overall_accuracy"`
	LegitimateFPR             float64                 `json:"legitimate_false_positive_rate"`
	ATOCatchRate              float64                 `json:"ato_catch_rate"`
	SocEngCatchRate           float64                 `json:"soc_eng_catch_rate"`
	AverageDecisionLatencyUs  float64                 `json:"average_decision_latency_us"`
}

// Evaluator benchmarks and audits decision accuracy and false positive rates.
type Evaluator struct{}

// NewEvaluator creates an evaluation runner.
func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// Evaluate runs the decision engine against a synthetic dataset.
func (ev *Evaluator) Evaluate(dataset simulator.Dataset) Summary {
	classes := []string{
		contracts.IntentLegitimate,
		contracts.IntentAccountTakeover,
		contracts.IntentSocialEngineering,
		contracts.IntentAccidental,
	}

	cm := make(map[string]map[string]int)
	for _, actual := range classes {
		cm[actual] = make(map[string]int)
		for _, pred := range classes {
			cm[actual][pred] = 0
		}
	}

	allSessions := append(dataset.LegitimateSessions, dataset.AccountTakeoverSessions...)
	allSessions = append(allSessions, dataset.SocialEngineeringSessions...)
	allSessions = append(allSessions, dataset.AccidentalSessions...)

	total := len(allSessions)
	correct := 0

	for _, ls := range allSessions {
		// Fresh session store for each evaluation session
		sessStore := session.New()
		baseStore := baseline.NewStore()
		svc := engine.NewService(engine.Options{
			Sessions:  sessStore,
			Baselines: baseStore,
		})

		for _, e := range ls.Events {
			stored, _ := sessStore.Append(e)
			_ = svc.Process(context.Background(), stored)
		}

		resp, ok := svc.GetIntent(ls.SessionID)
		if !ok {
			continue
		}

		predClass := resp.DominantIntent
		cm[ls.GroundTruth][predClass]++

		if predClass == ls.GroundTruth {
			correct++
		}
	}

	metricsByClass := make(map[string]ClassMetrics)
	for _, targetClass := range classes {
		tp := cm[targetClass][targetClass]
		fp := 0
		fn := 0
		tn := 0

		for actual, row := range cm {
			for pred, count := range row {
				if actual == targetClass && pred != targetClass {
					fn += count
				} else if actual != targetClass && pred == targetClass {
					fp += count
				} else if actual != targetClass && pred != targetClass {
					tn += count
				}
			}
		}

		prec := 0.0
		if tp+fp > 0 {
			prec = float64(tp) / float64(tp+fp)
		}
		rec := 0.0
		if tp+fn > 0 {
			rec = float64(tp) / float64(tp+fn)
		}
		f1 := 0.0
		if prec+rec > 0 {
			f1 = 2.0 * prec * rec / (prec + rec)
		}
		fpr := 0.0
		if fp+tn > 0 {
			fpr = float64(fp) / float64(fp+tn)
		}

		metricsByClass[targetClass] = ClassMetrics{
			Class:             targetClass,
			TruePositives:     tp,
			FalsePositives:    fp,
			TrueNegatives:     tn,
			FalseNegatives:    fn,
			Precision:         round4(prec),
			Recall:            round4(rec),
			F1:                round4(f1),
			FalsePositiveRate: round4(fpr),
		}
	}

	overallAcc := 0.0
	if total > 0 {
		overallAcc = float64(correct) / float64(total)
	}

	return Summary{
		TotalSessions:            total,
		ConfusionMatrix:          cm,
		MetricsByClass:           metricsByClass,
		OverallAccuracy:          round4(overallAcc),
		LegitimateFPR:            metricsByClass[contracts.IntentLegitimate].FalsePositiveRate,
		ATOCatchRate:             metricsByClass[contracts.IntentAccountTakeover].Recall,
		SocEngCatchRate:          metricsByClass[contracts.IntentSocialEngineering].Recall,
	}
}

// PrintReport formats the evaluation summary into a human-readable table.
func (s Summary) PrintReport() string {
	report := fmt.Sprintf(
		"Evaluation Summary (Total Sessions: %d)\n"+
			"Overall Accuracy: %.2f%%\n"+
			"Legitimate False Positive Rate: %.2f%%\n"+
			"Account Takeover Catch Rate: %.2f%%\n"+
			"Social Engineering Catch Rate: %.2f%%\n\n"+
			"Per-Class Performance:\n",
		s.TotalSessions,
		s.OverallAccuracy*100,
		s.LegitimateFPR*100,
		s.ATOCatchRate*100,
		s.SocEngCatchRate*100,
	)

	for class, m := range s.MetricsByClass {
		report += fmt.Sprintf("  [%s] Precision: %.2f%% | Recall: %.2f%% | F1: %.2f | FPR: %.2f%%\n",
			class, m.Precision*100, m.Recall*100, m.F1, m.FalsePositiveRate*100)
	}

	return report
}

func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}
