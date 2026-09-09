package scenarios

import (
	"fmt"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

// ScenarioInfo describes a scripted demonstration scenario (prd.md §23, §24, §35).
type ScenarioInfo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Expected    string `json:"expected_action"`
}

// List returns the catalogue of available demonstration scenarios.
func List() []ScenarioInfo {
	return []ScenarioInfo{
		{
			ID:          "legitimate",
			Title:       "Scenario A — Legitimate Transfer",
			Description: "Habitual payee, normal amount, recognized device, standard hours. Allowed with high confidence.",
			Expected:    "ALLOW",
		},
		{
			ID:          "account_takeover",
			Title:       "Scenario B — Account Takeover",
			Description: "Unrecognized device, rapid password change, mule beneficiary addition, high-value transfer. Blocked immediately.",
			Expected:    "BLOCK",
		},
		{
			ID:          "social_engineering",
			Title:       "Scenario C — Social Engineering (Authorized Push Payment)",
			Description: "Customer's real phone and valid OTP, but urgent transfer to new recipient under external impersonation. Triggers Intent Probe.",
			Expected:    "PROBE -> BLOCK",
		},
		{
			ID:          "accidental",
			Title:       "Scenario D — Accidental Transfer (Zero-Padding Error)",
			Description: "Habitual beneficiary with 10x normal amount (extra zero typo). Requires step-up confirmation.",
			Expected:    "VERIFY",
		},
		{
			ID:          "ambiguous",
			Title:       "Scenario E — Ambiguous / Incomplete Session",
			Description: "Early session state with incomplete telemetry. System admits uncertainty rather than guessing.",
			Expected:    "UNKNOWN / NEED MORE EVIDENCE",
		},
	}
}

// BuildEvents returns the ordered event sequence for a given scenario ID.
func BuildEvents(scenarioID string) (string, []contracts.Event, error) {
	now := time.Now().UTC()
	sessID := fmt.Sprintf("sess_%s_%d", scenarioID, now.Unix())
	userID := "user_001"

	switch scenarioID {
	case "legitimate", "scenario_a":
		return sessID, []contracts.Event{
			{
				EventID:   fmt.Sprintf("ev_%s_1", sessID),
				SessionID: sessID,
				UserID:    userID,
				Type:      contracts.EventLogin,
				Timestamp: now.Unix(),
				Metadata:  map[string]any{"device_id": "device_primary"},
			},
			{
				EventID:   fmt.Sprintf("ev_%s_2", sessID),
				SessionID: sessID,
				UserID:    userID,
				Type:      contracts.EventAmountEntered,
				Timestamp: now.Add(5 * time.Second).Unix(),
				Metadata: map[string]any{
					"amount":         35000.0,
					"beneficiary_id": "ben_mother",
				},
			},
			{
				EventID:   fmt.Sprintf("ev_%s_3", sessID),
				SessionID: sessID,
				UserID:    userID,
				Type:      contracts.EventOTPRequested,
				Timestamp: now.Add(8 * time.Second).Unix(),
			},
			{
				EventID:   fmt.Sprintf("ev_%s_4", sessID),
				SessionID: sessID,
				UserID:    userID,
				Type:      contracts.EventOTPVerified,
				Timestamp: now.Add(12 * time.Second).Unix(),
			},
			{
				EventID:   fmt.Sprintf("ev_%s_5", sessID),
				SessionID: sessID,
				UserID:    userID,
				Type:      contracts.EventTransferCompleted,
				Timestamp: now.Add(15 * time.Second).Unix(),
			},
		}, nil

	case "account_takeover", "scenario_b", "ato":
		return sessID, []contracts.Event{
			{
				EventID:   fmt.Sprintf("ev_%s_1", sessID),
				SessionID: sessID,
				UserID:    userID,
				Type:      contracts.EventDeviceChanged,
				Timestamp: now.Unix(),
				Metadata:  map[string]any{"device_id": "device_unknown_attacker"},
			},
			{
				EventID:   fmt.Sprintf("ev_%s_2", sessID),
				SessionID: sessID,
				UserID:    userID,
				Type:      contracts.EventPasswordChanged,
				Timestamp: now.Add(2 * time.Second).Unix(),
			},
			{
				EventID:   fmt.Sprintf("ev_%s_3", sessID),
				SessionID: sessID,
				UserID:    userID,
				Type:      contracts.EventBeneficiaryCreated,
				Timestamp: now.Add(4 * time.Second).Unix(),
				Metadata:  map[string]any{"beneficiary_id": "ben_attacker_mule_account"},
			},
			{
				EventID:   fmt.Sprintf("ev_%s_4", sessID),
				SessionID: sessID,
				UserID:    userID,
				Type:      contracts.EventAmountEntered,
				Timestamp: now.Add(6 * time.Second).Unix(),
				Metadata: map[string]any{
					"amount":         250000.0,
					"beneficiary_id": "ben_attacker_mule_account",
				},
			},
		}, nil

	case "social_engineering", "scenario_c", "soceng":
		return sessID, []contracts.Event{
			{
				EventID:   fmt.Sprintf("ev_%s_1", sessID),
				SessionID: sessID,
				UserID:    userID,
				Type:      contracts.EventLogin,
				Timestamp: now.Unix(),
				Metadata:  map[string]any{"device_id": "device_primary"},
			},
			{
				EventID:   fmt.Sprintf("ev_%s_2", sessID),
				SessionID: sessID,
				UserID:    userID,
				Type:      contracts.EventBeneficiaryCreated,
				Timestamp: now.Add(5 * time.Second).Unix(),
				Metadata:  map[string]any{"beneficiary_id": "ben_urgent_investigator"},
			},
			{
				EventID:   fmt.Sprintf("ev_%s_3", sessID),
				SessionID: sessID,
				UserID:    userID,
				Type:      contracts.EventAmountEntered,
				Timestamp: now.Add(12 * time.Second).Unix(),
				Metadata: map[string]any{
					"amount":         220000.0,
					"beneficiary_id": "ben_urgent_investigator",
				},
			},
			{
				EventID:   fmt.Sprintf("ev_%s_4", sessID),
				SessionID: sessID,
				UserID:    userID,
				Type:      contracts.EventOTPVerified,
				Timestamp: now.Add(18 * time.Second).Unix(),
			},
		}, nil

	case "accidental", "scenario_d":
		return sessID, []contracts.Event{
			{
				EventID:   fmt.Sprintf("ev_%s_1", sessID),
				SessionID: sessID,
				UserID:    userID,
				Type:      contracts.EventLogin,
				Timestamp: now.Unix(),
				Metadata:  map[string]any{"device_id": "device_primary"},
			},
			{
				EventID:   fmt.Sprintf("ev_%s_2", sessID),
				SessionID: sessID,
				UserID:    userID,
				Type:      contracts.EventAmountEntered,
				Timestamp: now.Add(8 * time.Second).Unix(),
				Metadata: map[string]any{
					"amount":         350000.0, // 10x typo to habitual contact
					"beneficiary_id": "ben_mother",
				},
			},
		}, nil

	case "ambiguous", "scenario_e":
		return sessID, []contracts.Event{
			{
				EventID:   fmt.Sprintf("ev_%s_1", sessID),
				SessionID: sessID,
				UserID:    userID,
				Type:      contracts.EventLogin,
				Timestamp: now.Unix(),
				Metadata:  map[string]any{"device_id": "device_primary"},
			},
		}, nil

	default:
		return "", nil, fmt.Errorf("unknown scenario ID: %s", scenarioID)
	}
}
