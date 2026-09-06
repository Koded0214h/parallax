package contracts

// Action is a policy decision (prd.md §21.8).
type Action string

const (
	ActionAllow    Action = "ALLOW"
	ActionVerify   Action = "VERIFY"
	ActionProbe    Action = "PROBE"
	ActionBlock    Action = "BLOCK"
	ActionEscalate Action = "ESCALATE"
)

// Intent class labels (prd.md §7). The engine emits a distribution over these.
const (
	IntentLegitimate        = "legitimate"
	IntentAccidental        = "accidental"
	IntentSocialEngineering = "social_engineering"
	IntentAccountTakeover   = "account_takeover"
)

// IntentResponse is the current inference for a session, returned by
// GET /v1/sessions/{id}/intent (prd.md §33).
//
// This is the contract between the intelligence engine and the frontend. The
// runtime layer only transports it.
type IntentResponse struct {
	SessionID   string             `json:"session_id"`
	Hypotheses  map[string]float64 `json:"hypotheses"`
	Uncertainty float64            `json:"uncertainty"`
	Action      Action             `json:"action"`
	Evidence    []string           `json:"evidence"`
	// EventsSeen is how many events the inference is based on. Runtime-owned.
	EventsSeen int `json:"events_seen"`
}
