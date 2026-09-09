package features

import (
	"testing"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/internal/baseline"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

func TestExtractNormalFeatures(t *testing.T) {
	b := baseline.DefaultBaseline("user_001")
	extractor := NewExtractor()

	now := time.Now().UTC()
	trajectory := []contracts.Event{
		{
			SessionID:       "sess_1",
			UserID:          "user_001",
			Type:            contracts.EventLogin,
			Timestamp:       now.Unix(),
			IngestedAtNanos: now.UnixNano(),
			Metadata:        map[string]any{"device_id": "device_primary"},
		},
		{
			SessionID:       "sess_1",
			UserID:          "user_001",
			Type:            contracts.EventAmountEntered,
			Timestamp:       now.Add(15 * time.Second).Unix(),
			IngestedAtNanos: now.Add(15 * time.Second).UnixNano(),
			Metadata: map[string]any{
				"amount":         35000.0,
				"beneficiary_id": "ben_mother",
			},
		},
	}

	fs := extractor.Extract(trajectory, b)

	if !fs.DeviceKnown {
		t.Errorf("expected device to be known")
	}
	if fs.IsNewBeneficiary {
		t.Errorf("expected beneficiary to be known")
	}
	if fs.AmountDeviation != 1.0 {
		t.Errorf("expected amount deviation 1.0, got %f", fs.AmountDeviation)
	}
}

func TestExtractAccountTakeoverFeatures(t *testing.T) {
	b := baseline.DefaultBaseline("user_001")
	extractor := NewExtractor()

	now := time.Now().UTC()
	trajectory := []contracts.Event{
		{
			SessionID:       "sess_ato",
			UserID:          "user_001",
			Type:            contracts.EventDeviceChanged,
			Timestamp:       now.Unix(),
			IngestedAtNanos: now.UnixNano(),
			Metadata:        map[string]any{"device_id": "device_unknown_hacker"},
		},
		{
			SessionID:       "sess_ato",
			UserID:          "user_001",
			Type:            contracts.EventPasswordChanged,
			Timestamp:       now.Add(5 * time.Second).Unix(),
			IngestedAtNanos: now.Add(5 * time.Second).UnixNano(),
		},
		{
			SessionID:       "sess_ato",
			UserID:          "user_001",
			Type:            contracts.EventBeneficiaryCreated,
			Timestamp:       now.Add(10 * time.Second).Unix(),
			IngestedAtNanos: now.Add(10 * time.Second).UnixNano(),
			Metadata:        map[string]any{"beneficiary_id": "ben_attacker_mule"},
		},
		{
			SessionID:       "sess_ato",
			UserID:          "user_001",
			Type:            contracts.EventAmountEntered,
			Timestamp:       now.Add(15 * time.Second).Unix(),
			IngestedAtNanos: now.Add(15 * time.Second).UnixNano(),
			Metadata: map[string]any{
				"amount":         250000.0,
				"beneficiary_id": "ben_attacker_mule",
			},
		},
	}

	fs := extractor.Extract(trajectory, b)

	if fs.DeviceKnown {
		t.Errorf("expected unknown device")
	}
	if !fs.IsNewBeneficiary {
		t.Errorf("expected new beneficiary")
	}
	if !fs.CredentialChangeRecent {
		t.Errorf("expected recent credential change")
	}
	if !fs.RapidSequence {
		t.Errorf("expected rapid sequence flag")
	}
	if fs.AmountDeviation < 5.0 {
		t.Errorf("expected high amount deviation, got %f", fs.AmountDeviation)
	}

	evidence := GenerateEvidenceItems(fs)
	if len(evidence) < 3 {
		t.Errorf("expected multiple evidence items, got %d", len(evidence))
	}
}

func TestExtractProbeResponseFeatures(t *testing.T) {
	b := baseline.DefaultBaseline("user_001")
	extractor := NewExtractor()

	now := time.Now().UTC()
	trajectory := []contracts.Event{
		{
			SessionID:       "sess_probe",
			UserID:          "user_001",
			Type:            contracts.EventLogin,
			Timestamp:       now.Unix(),
			IngestedAtNanos: now.UnixNano(),
			Metadata:        map[string]any{"device_id": "device_primary"},
		},
		{
			SessionID:       "sess_probe",
			UserID:          "user_001",
			Type:            contracts.EventIntentProbeResponse,
			Timestamp:       now.Add(10 * time.Second).Unix(),
			IngestedAtNanos: now.Add(10 * time.Second).UnixNano(),
			Metadata: map[string]any{
				"response": "The bank security team told me to reverse my account immediately",
			},
		},
	}

	fs := extractor.Extract(trajectory, b)
	if !fs.ProbeResponded {
		t.Errorf("expected probe responded true")
	}
	if !fs.ProbeImpersonationSignal {
		t.Errorf("expected impersonation signal")
	}
	if !fs.ProbeUrgencySignal {
		t.Errorf("expected urgency signal")
	}
}
