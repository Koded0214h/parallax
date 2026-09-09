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

// ProbeOption is an option in a contextual probe presented to the user.
type ProbeOption struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Category string `json:"category,omitempty"`
}

// ProbePayload represents an active or completed intent probe.
type ProbePayload struct {
	ProbeID       string        `json:"probe_id"`
	Prompt        string        `json:"prompt"`
	Options       []ProbeOption `json:"options"`
	AllowFreeform bool          `json:"allow_freeform"`
	Completed     bool          `json:"completed"`
	Response      string        `json:"response,omitempty"`
}

// IntentResponse is the current inference for a session, returned by
// GET /v1/sessions/{id}/intent (prd.md §33).
//
// This is the contract between the intelligence engine and the frontend.
type IntentResponse struct {
	SessionID      string             `json:"session_id"`
	Hypotheses     map[string]float64 `json:"hypotheses"`
	Uncertainty    float64            `json:"uncertainty"`
	Action         Action             `json:"action"`
	Evidence       []string           `json:"evidence"`
	EventsSeen     int                `json:"events_seen"`
	Explanation    string             `json:"explanation,omitempty"`
	Confidence     string             `json:"confidence,omitempty"`
	RiskScore      float64            `json:"risk_score,omitempty"`
	DominantIntent string             `json:"dominant_intent,omitempty"`
	Probe          *ProbePayload      `json:"probe,omitempty"`
}
