package ml

import (
	"math"

	"github.com/holiday-heartbreaks/parallax/backend/internal/baseline"
	"github.com/holiday-heartbreaks/parallax/backend/internal/features"
	"github.com/holiday-heartbreaks/parallax/backend/internal/sequence"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

// Prediction contains machine-learning inference probabilities and scores.
type Prediction struct {
	Probabilities      map[string]float64 `json:"probabilities"`
	DominantIntent     string             `json:"dominant_intent"`
	Confidence         float64            `json:"confidence"`
	MarkovAnomalyScore float64            `json:"markov_anomaly_score"`
	MarginScores       map[string]float64 `json:"margin_scores"`
}

// Classifier encapsulates the hybrid GBDT and Markov sequence detection architecture.
type Classifier struct {
	trees  *TreeEnsemble
	markov *MarkovSequenceModel
}

// NewClassifier initializes the production machine-learning model.
func NewClassifier() *Classifier {
	return &Classifier{
		trees:  NewTreeEnsemble(),
		markov: NewMarkovSequenceModel(),
	}
}

// Predict processes a session trajectory, feature set, and user baseline into probabilistic predictions.
func (c *Classifier) Predict(
	trajectory []contracts.Event,
	fs features.FeatureSet,
	seq sequence.SequenceResult,
	b baseline.UserBaseline,
) Prediction {
	markovScore := c.markov.ScoreTrajectory(trajectory)
	vec := ToVector(fs, seq, markovScore)

	margins := c.trees.PredictRaw(vec)
	probs := c.trees.PredictProbabilities(vec)

	var dominant string
	var maxP float64

	for k, p := range probs {
		if p > maxP {
			maxP = p
			dominant = k
		}
	}

	return Prediction{
		Probabilities:      probs,
		DominantIntent:     dominant,
		Confidence:         math.Round(maxP*10000) / 10000,
		MarkovAnomalyScore: markovScore,
		MarginScores:       margins,
	}
}
