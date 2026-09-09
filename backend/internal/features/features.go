package features

import (
	"strings"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/internal/baseline"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

// FeatureSet represents structured evidence extracted from a session trajectory
// against a customer baseline (prd.md §9, §12).
type FeatureSet struct {
	// Financial metrics
	Amount                float64 `json:"amount"`
	AmountDeviation       float64 `json:"amount_deviation"`
	AmountZScore          float64 `json:"amount_z_score"`
	ZeroPaddingAnomaly    bool    `json:"zero_padding_anomaly"`

	// Beneficiary metrics
	IsNewBeneficiary      bool    `json:"is_new_beneficiary"`
	BeneficiaryAgeSeconds float64 `json:"beneficiary_age_seconds"`
	BeneficiarySeenBefore bool    `json:"beneficiary_seen_before"`
	BeneficiaryID         string  `json:"beneficiary_id,omitempty"`

	// Device metrics
	DeviceKnown         bool   `json:"device_known"`
	DeviceChangedRecent bool   `json:"device_changed_recent"`
	DeviceID            string `json:"device_id,omitempty"`

	// Credential / Security state
	CredentialChangeRecent  bool    `json:"credential_change_recent"`
	CredentialChangeSeconds float64 `json:"credential_change_seconds"`

	// Temporal & Interaction dynamics
	TimeDeviation             float64 `json:"time_deviation"`
	IsAtypicalHour            bool    `json:"is_atypical_hour"`
	InteractionSpeedDeviation float64 `json:"interaction_speed_deviation"`
	SessionDurationSeconds    float64 `json:"session_duration_seconds"`
	EventCount                int     `json:"event_count"`
	RapidSequence             bool    `json:"rapid_sequence"`

	// Authentication / State transitions
	FailedAttemptsCount int  `json:"failed_attempts_count"`
	OTPRequested        bool `json:"otp_requested"`
	OTPVerified         bool `json:"otp_verified"`

	// Intent Probe observations (prd.md §15)
	ProbeResponded           bool   `json:"probe_responded"`
	ProbeText                string `json:"probe_text,omitempty"`
	ProbeImpersonationSignal bool   `json:"probe_impersonation_signal"`
	ProbeUrgencySignal       bool   `json:"probe_urgency_signal"`
}

// Extractor extracts deterministic, testable behavioural and session features.
type Extractor struct{}

// NewExtractor initializes a feature extractor.
func NewExtractor() *Extractor {
	return &Extractor{}
}

// Extract transforms a session trajectory and customer baseline into a FeatureSet.
func (e *Extractor) Extract(trajectory []contracts.Event, b baseline.UserBaseline) FeatureSet {
	fs := FeatureSet{
		EventCount: len(trajectory),
	}

	if len(trajectory) == 0 {
		return fs
	}

	firstEvent := trajectory[0]
	lastEvent := trajectory[len(trajectory)-1]

	// Session duration calculation
	var firstNanos, lastNanos int64
	if firstEvent.IngestedAtNanos > 0 {
		firstNanos = firstEvent.IngestedAtNanos
		lastNanos = lastEvent.IngestedAtNanos
	} else {
		firstNanos = firstEvent.Timestamp * 1e9
		lastNanos = lastEvent.Timestamp * 1e9
	}
	if lastNanos >= firstNanos {
		fs.SessionDurationSeconds = float64(lastNanos-firstNanos) / 1e9
	}

	// Scan trajectory for contextual markers
	var (
		beneficiaryCreatedNanos int64
		credentialChangedNanos  int64
		latestTransferNanos     int64
		firstDeviceID           string
		currentDeviceID         string
		seenBeneficiaryCreated  bool
		seenCredentialChange    bool
		seenDeviceChange        bool
	)

	for _, ev := range trajectory {
		evNanos := ev.IngestedAtNanos
		if evNanos == 0 {
			evNanos = ev.Timestamp * 1e9
		}

		// Device extraction
		if devVal, ok := ev.Metadata["device_id"].(string); ok && devVal != "" {
			if firstDeviceID == "" {
				firstDeviceID = devVal
			}
			currentDeviceID = devVal
		}

		// Beneficiary extraction
		if benVal, ok := ev.Metadata["beneficiary_id"].(string); ok && benVal != "" {
			fs.BeneficiaryID = benVal
		}

		// Amount extraction
		if amtVal, ok := getFloat64(ev.Metadata, "amount"); ok && amtVal > 0 {
			fs.Amount = amtVal
		}

		switch ev.Type {
		case contracts.EventDeviceChanged:
			seenDeviceChange = true
			fs.DeviceChangedRecent = true

		case contracts.EventPasswordChanged, contracts.EventPINChanged:
			seenCredentialChange = true
			fs.CredentialChangeRecent = true
			credentialChangedNanos = evNanos

		case contracts.EventBeneficiaryCreated:
			seenBeneficiaryCreated = true
			beneficiaryCreatedNanos = evNanos

		case contracts.EventTransferStarted, contracts.EventAmountEntered:
			latestTransferNanos = evNanos

		case contracts.EventTransferFailed:
			fs.FailedAttemptsCount++

		case contracts.EventOTPRequested:
			fs.OTPRequested = true

		case contracts.EventOTPVerified:
			fs.OTPVerified = true

		case contracts.EventIntentProbeResponse:
			fs.ProbeResponded = true
			if textVal, ok := ev.Metadata["response"].(string); ok {
				fs.ProbeText = textVal
				lower := strings.ToLower(textVal)
				if containsAny(lower, "bank", "security", "reverse", "freeze", "investigation", "agent", "support", "official", "safeguard") {
					fs.ProbeImpersonationSignal = true
				}
				if containsAny(lower, "immediately", "urgent", "hurry", "fast", "right now", "emergency", "penalty", "block my account") {
					fs.ProbeUrgencySignal = true
				}
			}
		}
	}

	// Device known evaluation
	if currentDeviceID != "" {
		fs.DeviceID = currentDeviceID
		if _, known := b.KnownDevices[currentDeviceID]; known {
			fs.DeviceKnown = true
		} else {
			fs.DeviceKnown = false
		}
	} else if len(b.KnownDevices) > 0 {
		// If no device specified in event, default to unknown device anomaly
		fs.DeviceKnown = false
	} else {
		fs.DeviceKnown = true
	}

	// Beneficiary age and known checks
	if fs.BeneficiaryID != "" {
		if _, known := b.KnownBeneficiaries[fs.BeneficiaryID]; known {
			fs.BeneficiarySeenBefore = true
			fs.IsNewBeneficiary = false
		} else {
			fs.BeneficiarySeenBefore = false
			fs.IsNewBeneficiary = true
		}
	} else if seenBeneficiaryCreated {
		fs.IsNewBeneficiary = true
	}

	if seenBeneficiaryCreated && latestTransferNanos >= beneficiaryCreatedNanos && beneficiaryCreatedNanos > 0 {
		fs.BeneficiaryAgeSeconds = float64(latestTransferNanos-beneficiaryCreatedNanos) / 1e9
	} else if seenBeneficiaryCreated && fs.SessionDurationSeconds > 0 {
		fs.BeneficiaryAgeSeconds = fs.SessionDurationSeconds
	}

	// Credential change recency
	if seenCredentialChange && latestTransferNanos >= credentialChangedNanos && credentialChangedNanos > 0 {
		fs.CredentialChangeSeconds = float64(latestTransferNanos-credentialChangedNanos) / 1e9
	} else if !b.LastPasswordChange.IsZero() {
		fs.CredentialChangeSeconds = time.Since(b.LastPasswordChange).Seconds()
		if fs.CredentialChangeSeconds < 3600 {
			fs.CredentialChangeRecent = true
		}
	}

	// Amount deviation
	if fs.Amount > 0 {
		ratio, zScore := b.AmountDeviation(fs.Amount)
		fs.AmountDeviation = ratio
		fs.AmountZScore = zScore

		// Accidental double-zero typing anomaly check (e.g., 500,000 instead of 50,000)
		if fs.BeneficiarySeenBefore && ratio >= 8.0 && ratio <= 12.0 {
			fs.ZeroPaddingAnomaly = true
		}
	}

	// Temporal deviation
	eventTime := time.Unix(lastEvent.Timestamp, 0).UTC()
	hour := eventTime.Hour()
	if !b.IsHourTypical(hour) {
		fs.IsAtypicalHour = true
		fs.TimeDeviation = 2.5
	}

	// Rapid sequence check: critical security mutations within short timeframe (< 60s)
	if seenDeviceChange || seenCredentialChange || seenBeneficiaryCreated {
		if fs.SessionDurationSeconds > 0 && fs.SessionDurationSeconds < 60 && fs.Amount > 0 {
			fs.RapidSequence = true
		}
	}

	// Interaction speed deviation
	if fs.SessionDurationSeconds > 0 && fs.EventCount > 1 {
		eventsPerSec := float64(fs.EventCount) / fs.SessionDurationSeconds
		normalPerSec := b.TypicalEventsPerMinute / 60.0
		if normalPerSec > 0 {
			fs.InteractionSpeedDeviation = eventsPerSec / normalPerSec
		}
	}

	return fs
}

// GenerateEvidenceItems transforms raw extracted features into clear, human-readable evidence statements (prd.md §9, §17).
func GenerateEvidenceItems(fs FeatureSet) []string {
	var items []string

	if fs.AmountDeviation >= 2.0 {
		items = append(items, "Transfer amount is significantly higher than user baseline")
	} else if fs.AmountDeviation > 0 && fs.AmountDeviation < 0.2 {
		items = append(items, "Transfer amount is unusually small compared to user baseline")
	}

	if fs.ZeroPaddingAnomaly {
		items = append(items, "Transfer amount appears to have accidental extra zeroes (10x baseline to known contact)")
	}

	if fs.IsNewBeneficiary {
		items = append(items, "Recipient beneficiary has never been sent money before")
	}

	if fs.BeneficiaryAgeSeconds > 0 && fs.BeneficiaryAgeSeconds < 120 {
		items = append(items, "Recipient was added immediately before transfer")
	}

	if !fs.DeviceKnown && fs.DeviceID != "" {
		items = append(items, "Transaction initiated from an unrecognized device")
	}

	if fs.DeviceChangedRecent {
		items = append(items, "Device change observed during current active session")
	}

	if fs.CredentialChangeRecent {
		items = append(items, "Security credentials were modified immediately preceding transaction")
	}

	if fs.RapidSequence {
		items = append(items, "High-velocity succession of critical security modifications detected")
	}

	if fs.IsAtypicalHour {
		items = append(items, "Transaction time deviates significantly from user's active hours")
	}

	if fs.InteractionSpeedDeviation > 3.0 {
		items = append(items, "Interaction speed is abnormally elevated compared to human baseline")
	}

	if fs.FailedAttemptsCount > 0 {
		items = append(items, "Multiple failed transaction or authentication attempts observed")
	}

	if fs.ProbeResponded && fs.ProbeImpersonationSignal {
		items = append(items, "User response to intent probe indicates suspected authority impersonation")
	}

	if fs.ProbeResponded && fs.ProbeUrgencySignal {
		items = append(items, "User reported urgency and coercive pressure from external party")
	}

	if len(items) == 0 {
		items = append(items, "Session telemetry and parameters are consistent with historical baseline")
	}

	return items
}

func getFloat64(m map[string]any, key string) (float64, bool) {
	if m == nil {
		return 0, false
	}
	val, ok := m[key]
	if !ok {
		return 0, false
	}
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint64:
		return float64(v), true
	default:
		return 0, false
	}
}

func containsAny(s string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}
