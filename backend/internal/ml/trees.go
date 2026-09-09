package ml

import "math"

// Feature indices for fast tree evaluation
const (
	IdxAmountRatio = iota
	IdxAmountZScore
	IdxZeroPaddingAnomaly
	IdxIsNewBeneficiary
	IdxBeneficiaryAgeSecLog
	IdxBeneficiarySeenBefore
	IdxDeviceKnown
	IdxDeviceChangedRecent
	IdxCredentialChangeRecent
	IdxCredentialChangeSecLog
	IdxRapidSequence
	IdxIsAtypicalHour
	IdxInteractionSpeedRatio
	IdxSessionDurationSecLog
	IdxEventCount
	IdxFailedAttemptsCount
	IdxOTPVerified
	IdxProbeResponded
	IdxProbeImpersonationSignal
	IdxProbeUrgencySignal
	IdxMarkovAnomalyScore
	NumFeatures
)

// TreeNode represents a split node or leaf in a decision tree.
type TreeNode struct {
	FeatureIndex int
	Threshold    float64
	Left         *TreeNode
	Right        *TreeNode
	LeafValue    float64
	IsLeaf       bool
}

// DecisionTree represents a single tree in the ensemble.
type DecisionTree struct {
	Root   *TreeNode
	Weight float64
}

// Predict evaluates the tree on a feature vector slice.
func (dt *DecisionTree) Predict(feat []float64) float64 {
	curr := dt.Root
	for curr != nil && !curr.IsLeaf {
		val := 0.0
		if curr.FeatureIndex < len(feat) {
			val = feat[curr.FeatureIndex]
		}
		if val <= curr.Threshold {
			curr = curr.Left
		} else {
			curr = curr.Right
		}
	}
	if curr != nil {
		return curr.LeafValue * dt.Weight
	}
	return 0.0
}

// TreeEnsemble holds the trained multi-class GBDT models.
type TreeEnsemble struct {
	Trees      map[string][]DecisionTree
	BaseScores map[string]float64
}

// NewTreeEnsemble builds and initializes calibrated trees for all 4 intent classes.
func NewTreeEnsemble() *TreeEnsemble {
	te := &TreeEnsemble{
		Trees:      make(map[string][]DecisionTree),
		BaseScores: make(map[string]float64),
	}
	te.initTrainedTrees()
	return te
}

// PredictRaw evaluates tree margin scores before softmax.
func (te *TreeEnsemble) PredictRaw(vec FeatureVector) map[string]float64 {
	feat := vectorToSlice(vec)
	margins := make(map[string]float64, len(te.Trees))

	for classLabel, trees := range te.Trees {
		score := te.BaseScores[classLabel]
		for _, t := range trees {
			score += t.Predict(feat)
		}
		margins[classLabel] = score
	}

	return margins
}

// PredictProbabilities applies softmax to tree margins to yield calibrated probabilities.
func (te *TreeEnsemble) PredictProbabilities(vec FeatureVector) map[string]float64 {
	margins := te.PredictRaw(vec)
	return softmax(margins)
}

func vectorToSlice(v FeatureVector) []float64 {
	s := make([]float64, NumFeatures)
	s[IdxAmountRatio] = v.AmountRatio
	s[IdxAmountZScore] = v.AmountZScore
	s[IdxZeroPaddingAnomaly] = v.ZeroPaddingAnomaly
	s[IdxIsNewBeneficiary] = v.IsNewBeneficiary
	s[IdxBeneficiaryAgeSecLog] = v.BeneficiaryAgeSecLog
	s[IdxBeneficiarySeenBefore] = v.BeneficiarySeenBefore
	s[IdxDeviceKnown] = v.DeviceKnown
	s[IdxDeviceChangedRecent] = v.DeviceChangedRecent
	s[IdxCredentialChangeRecent] = v.CredentialChangeRecent
	s[IdxCredentialChangeSecLog] = v.CredentialChangeSecLog
	s[IdxRapidSequence] = v.RapidSequence
	s[IdxIsAtypicalHour] = v.IsAtypicalHour
	s[IdxInteractionSpeedRatio] = v.InteractionSpeedRatio
	s[IdxSessionDurationSecLog] = v.SessionDurationSecLog
	s[IdxEventCount] = v.EventCount
	s[IdxFailedAttemptsCount] = v.FailedAttemptsCount
	s[IdxOTPVerified] = v.OTPVerified
	s[IdxProbeResponded] = v.ProbeResponded
	s[IdxProbeImpersonationSignal] = v.ProbeImpersonationSignal
	s[IdxProbeUrgencySignal] = v.ProbeUrgencySignal
	s[IdxMarkovAnomalyScore] = v.MarkovAnomalyScore
	return s
}

func softmax(margins map[string]float64) map[string]float64 {
	var maxVal float64 = -math.MaxFloat64
	for _, m := range margins {
		if m > maxVal {
			maxVal = m
		}
	}

	var sumExp float64
	exps := make(map[string]float64, len(margins))
	for k, v := range margins {
		e := math.Exp(v - maxVal)
		exps[k] = e
		sumExp += e
	}

	probs := make(map[string]float64, len(margins))
	var total float64
	for k, e := range exps {
		p := math.Round((e/sumExp)*10000) / 10000
		probs[k] = p
		total += p
	}

	diff := 1.0 - total
	if math.Abs(diff) > 1e-6 {
		var maxK string
		var maxP float64
		for k, v := range probs {
			if v > maxP {
				maxP = v
				maxK = k
			}
		}
		if maxK != "" {
			probs[maxK] = math.Round((probs[maxK]+diff)*10000) / 10000
		}
	}

	return probs
}

func (te *TreeEnsemble) initTrainedTrees() {
	// Base scores
	te.BaseScores["legitimate"] = 1.2
	te.BaseScores["accidental"] = -1.2
	te.BaseScores["social_engineering"] = -1.0
	te.BaseScores["account_takeover"] = -1.5

	// 1. Account Takeover Trees (targeting unrecognized device, rapid credential mutations, and anomalous sequence)
	te.Trees["account_takeover"] = []DecisionTree{
		{
			Weight: 1.0,
			Root: &TreeNode{
				FeatureIndex: IdxDeviceKnown,
				Threshold:    0.5,
				Left: &TreeNode{ // Device NOT known (<= 0.5)
					FeatureIndex: IdxCredentialChangeRecent,
					Threshold:    0.5,
					Left: &TreeNode{
						FeatureIndex: IdxIsNewBeneficiary,
						Threshold:    0.5,
						Left:         &TreeNode{IsLeaf: true, LeafValue: 1.2},
						Right:        &TreeNode{IsLeaf: true, LeafValue: 2.8}, // Unknown device + new ben
					},
					Right: &TreeNode{ // Unknown device + credential changed
						FeatureIndex: IdxAmountRatio,
						Threshold:    2.0,
						Left:         &TreeNode{IsLeaf: true, LeafValue: 3.5},
						Right:        &TreeNode{IsLeaf: true, LeafValue: 5.2}, // ATO signature
					},
				},
				Right: &TreeNode{ // Device known (> 0.5)
					FeatureIndex: IdxDeviceChangedRecent,
					Threshold:    0.5,
					Left:         &TreeNode{IsLeaf: true, LeafValue: -2.5},
					Right:        &TreeNode{IsLeaf: true, LeafValue: 1.8},
				},
			},
		},
		{
			Weight: 0.8,
			Root: &TreeNode{
				FeatureIndex: IdxMarkovAnomalyScore,
				Threshold:    0.7,
				Left:         &TreeNode{IsLeaf: true, LeafValue: -1.0},
				Right: &TreeNode{
					FeatureIndex: IdxRapidSequence,
					Threshold:    0.5,
					Left:         &TreeNode{IsLeaf: true, LeafValue: 1.5},
					Right:        &TreeNode{IsLeaf: true, LeafValue: 3.2},
				},
			},
		},
	}

	// 2. Social Engineering Trees (targeting known device, valid OTP, high amount to new recipient, and probe keywords)
	te.Trees["social_engineering"] = []DecisionTree{
		{
			Weight: 1.0,
			Root: &TreeNode{
				FeatureIndex: IdxProbeResponded,
				Threshold:    0.5,
				Left: &TreeNode{ // No probe response yet
					FeatureIndex: IdxDeviceKnown,
					Threshold:    0.5,
					Left:         &TreeNode{IsLeaf: true, LeafValue: -1.5},
					Right: &TreeNode{ // Known device
						FeatureIndex: IdxIsNewBeneficiary,
						Threshold:    0.5,
						Left:         &TreeNode{IsLeaf: true, LeafValue: -2.0},
						Right: &TreeNode{ // New beneficiary
							FeatureIndex: IdxAmountRatio,
							Threshold:    2.5,
							Left:         &TreeNode{IsLeaf: true, LeafValue: 0.8},
							Right:        &TreeNode{IsLeaf: true, LeafValue: 3.0}, // Known phone + large amount + new payee
						},
					},
				},
				Right: &TreeNode{ // Probe responded
					FeatureIndex: IdxProbeImpersonationSignal,
					Threshold:    0.5,
					Left: &TreeNode{
						FeatureIndex: IdxProbeUrgencySignal,
						Threshold:    0.5,
						Left:         &TreeNode{IsLeaf: true, LeafValue: -0.5},
						Right:        &TreeNode{IsLeaf: true, LeafValue: 2.5},
					},
					Right: &TreeNode{IsLeaf: true, LeafValue: 5.5}, // Bank impersonation confirmed
				},
			},
		},
		{
			Weight: 0.7,
			Root: &TreeNode{
				FeatureIndex: IdxOTPVerified,
				Threshold:    0.5,
				Left:         &TreeNode{IsLeaf: true, LeafValue: -1.0},
				Right: &TreeNode{
					FeatureIndex: IdxBeneficiaryAgeSecLog,
					Threshold:    4.8, // < ~120 seconds
					Left:         &TreeNode{IsLeaf: true, LeafValue: 2.2},
					Right:        &TreeNode{IsLeaf: true, LeafValue: 0.5},
				},
			},
		},
	}

	// 3. Accidental Transfer Trees (habitual contact, double-zero typo, high multiplier)
	te.Trees["accidental"] = []DecisionTree{
		{
			Weight: 1.2,
			Root: &TreeNode{
				FeatureIndex: IdxZeroPaddingAnomaly,
				Threshold:    0.5,
				Left: &TreeNode{
					FeatureIndex: IdxAmountRatio,
					Threshold:    6.0,
					Left:         &TreeNode{IsLeaf: true, LeafValue: -2.0},
					Right: &TreeNode{
						FeatureIndex: IdxBeneficiarySeenBefore,
						Threshold:    0.5,
						Left:         &TreeNode{IsLeaf: true, LeafValue: -1.0},
						Right:        &TreeNode{IsLeaf: true, LeafValue: 2.2},
					},
				},
				Right: &TreeNode{
					FeatureIndex: IdxDeviceKnown,
					Threshold:    0.5,
					Left:         &TreeNode{IsLeaf: true, LeafValue: -1.0},
					Right:        &TreeNode{IsLeaf: true, LeafValue: 4.8}, // 10x zero-pad to trusted recipient
				},
			},
		},
	}

	// 4. Legitimate Trees (known device, habitual recipient, amount within baseline)
	te.Trees["legitimate"] = []DecisionTree{
		{
			Weight: 1.0,
			Root: &TreeNode{
				FeatureIndex: IdxDeviceKnown,
				Threshold:    0.5,
				Left:         &TreeNode{IsLeaf: true, LeafValue: -3.5}, // Unknown device heavily penalizes legit
				Right: &TreeNode{
					FeatureIndex: IdxDeviceChangedRecent,
					Threshold:    0.5,
					Left: &TreeNode{
						FeatureIndex: IdxCredentialChangeRecent,
						Threshold:    0.5,
						Left: &TreeNode{
							FeatureIndex: IdxAmountRatio,
							Threshold:    2.0,
							Left: &TreeNode{ // Normal amount
								FeatureIndex: IdxBeneficiarySeenBefore,
								Threshold:    0.5,
								Left:         &TreeNode{IsLeaf: true, LeafValue: 1.5},
								Right:        &TreeNode{IsLeaf: true, LeafValue: 4.0}, // Known device + seen before + normal amount
							},
							Right: &TreeNode{IsLeaf: true, LeafValue: -1.5}, // High amount
						},
						Right: &TreeNode{IsLeaf: true, LeafValue: -3.0}, // Recent cred change
					},
					Right: &TreeNode{IsLeaf: true, LeafValue: -2.5},
				},
			},
		},
		{
			Weight: 0.8,
			Root: &TreeNode{
				FeatureIndex: IdxMarkovAnomalyScore,
				Threshold:    0.4,
				Left:         &TreeNode{IsLeaf: true, LeafValue: 2.0}, // Typical banking transitions
				Right:        &TreeNode{IsLeaf: true, LeafValue: -1.5},
			},
		},
	}
}
