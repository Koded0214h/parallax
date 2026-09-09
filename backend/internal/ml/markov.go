package ml

import (
	"math"

	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

// MarkovSequenceModel scores the likelihood of event transition sequences.
type MarkovSequenceModel struct {
	// transitionMatrix[from][to] = probability
	matrix map[contracts.EventType]map[contracts.EventType]float64
}

// NewMarkovSequenceModel initializes the sequence transition likelihood model.
func NewMarkovSequenceModel() *MarkovSequenceModel {
	m := &MarkovSequenceModel{
		matrix: make(map[contracts.EventType]map[contracts.EventType]float64),
	}
	m.seedEmpiricalTransitions()
	return m
}

// ScoreTrajectory returns an anomaly score in [0.0, 1.0] where 1.0 indicates severe transition anomaly.
func (m *MarkovSequenceModel) ScoreTrajectory(trajectory []contracts.Event) float64 {
	if len(trajectory) < 2 {
		return 0.1
	}

	var totalNLL float64
	var transitions int

	for i := 0; i < len(trajectory)-1; i++ {
		from := trajectory[i].Type
		to := trajectory[i+1].Type

		prob := m.getTransitionProb(from, to)
		// Negative log-likelihood with epsilon floor
		nll := -math.Log(math.Max(prob, 1e-4))
		totalNLL += nll
		transitions++
	}

	if transitions == 0 {
		return 0.1
	}

	avgNLL := totalNLL / float64(transitions)
	// Normal banking transitions average NLL around 0.5 - 1.8.
	// Anomalous attack transitions jump to 4.0 - 9.0.
	// Sigmoid transform to normalize between 0.0 and 1.0
	score := 1.0 / (1.0 + math.Exp(-(avgNLL - 3.2)))
	return math.Round(score*10000) / 10000
}

func (m *MarkovSequenceModel) getTransitionProb(from, to contracts.EventType) float64 {
	if row, ok := m.matrix[from]; ok {
		if p, exists := row[to]; exists {
			return p
		}
	}
	// Default smoothing probability for unseen transitions
	return 0.005
}

func (m *MarkovSequenceModel) seedEmpiricalTransitions() {
	// Initialize default transitions based on normal legitimate banking flows
	m.matrix[contracts.EventLogin] = map[contracts.EventType]float64{
		contracts.EventAmountEntered:       0.35,
		contracts.EventTransferStarted:     0.35,
		contracts.EventBeneficiaryCreated:  0.10,
		contracts.EventDeviceSeen:          0.12,
		contracts.EventLogout:              0.05,
		contracts.EventDeviceChanged:       0.01,
		contracts.EventPasswordChanged:     0.005,
		contracts.EventPINChanged:          0.005,
	}

	m.matrix[contracts.EventTransferStarted] = map[contracts.EventType]float64{
		contracts.EventAmountEntered:       0.80,
		contracts.EventBeneficiaryCreated:  0.10,
		contracts.EventBeneficiaryModified: 0.05,
		contracts.EventTransferFailed:      0.05,
	}

	m.matrix[contracts.EventAmountEntered] = map[contracts.EventType]float64{
		contracts.EventOTPRequested:        0.70,
		contracts.EventOTPVerified:         0.15,
		contracts.EventTransferCompleted:   0.10,
		contracts.EventTransferFailed:      0.05,
	}

	m.matrix[contracts.EventOTPRequested] = map[contracts.EventType]float64{
		contracts.EventOTPVerified:         0.90,
		contracts.EventTransferFailed:      0.08,
		contracts.EventOTPRequested:        0.02,
	}

	m.matrix[contracts.EventOTPVerified] = map[contracts.EventType]float64{
		contracts.EventTransferCompleted:   0.85,
		contracts.EventTransferFailed:      0.10,
		contracts.EventIntentProbeStarted:  0.05,
	}

	// Normal beneficiary addition followed by routine transfer
	m.matrix[contracts.EventBeneficiaryCreated] = map[contracts.EventType]float64{
		contracts.EventTransferStarted:     0.45,
		contracts.EventAmountEntered:       0.40,
		contracts.EventBeneficiaryModified: 0.05,
		contracts.EventLogout:              0.05,
	}

	// High-anomaly transitions (attack patterns)
	m.matrix[contracts.EventDeviceChanged] = map[contracts.EventType]float64{
		contracts.EventPasswordChanged:     0.002, // Extremely rare in legitimate sessions
		contracts.EventPINChanged:          0.002,
		contracts.EventBeneficiaryCreated:  0.01,
		contracts.EventLogin:               0.60,
		contracts.EventDeviceSeen:          0.30,
	}

	m.matrix[contracts.EventPasswordChanged] = map[contracts.EventType]float64{
		contracts.EventBeneficiaryCreated:  0.001, // Signature ATO step
		contracts.EventTransferStarted:     0.002,
		contracts.EventLogin:               0.50,
		contracts.EventLogout:              0.45,
	}
}
