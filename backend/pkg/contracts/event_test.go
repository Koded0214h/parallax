package contracts

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestEventTypeValid(t *testing.T) {
	all := []EventType{
		EventLogin, EventLogout, EventDeviceSeen, EventDeviceChanged,
		EventPasswordChanged, EventPINChanged, EventBeneficiaryCreated,
		EventBeneficiaryModified, EventTransferStarted, EventAmountEntered,
		EventOTPRequested, EventOTPVerified, EventTransferCompleted,
		EventTransferFailed, EventIntentProbeStarted, EventIntentProbeResponse,
	}
	if len(all) != len(knownEventTypes) {
		t.Fatalf("test list has %d types, knownEventTypes has %d", len(all), len(knownEventTypes))
	}
	for _, tp := range all {
		if !tp.Valid() {
			t.Errorf("%s should be valid", tp)
		}
	}

	for _, bad := range []EventType{"", "login", "LOGIN ", "TRANSFER", "NONSENSE"} {
		if bad.Valid() {
			t.Errorf("%q should not be valid", bad)
		}
	}
}

func TestValidateForIngestAccepts(t *testing.T) {
	ok := []Event{
		{SessionID: "s1", UserID: "u1", Type: EventLogin},
		{SessionID: "s1", UserID: "u1", Type: EventAmountEntered, Timestamp: 1_788_594_121},
		{SessionID: "s1", UserID: "u1", Type: EventTransferCompleted, Timestamp: 0},
	}
	for i, e := range ok {
		if err := e.ValidateForIngest(); err != nil {
			t.Errorf("case %d: valid event rejected: %v", i, err)
		}
	}
}

func TestValidateForIngestRejects(t *testing.T) {
	cases := map[string]Event{
		"missing session":    {UserID: "u1", Type: EventLogin},
		"blank session":      {SessionID: "   ", UserID: "u1", Type: EventLogin},
		"missing user":       {SessionID: "s1", Type: EventLogin},
		"blank user":         {SessionID: "s1", UserID: "\t", Type: EventLogin},
		"missing type":       {SessionID: "s1", UserID: "u1"},
		"lowercase type":     {SessionID: "s1", UserID: "u1", Type: EventType("login")},
		"unknown type":       {SessionID: "s1", UserID: "u1", Type: EventType("WAT")},
		"negative timestamp": {SessionID: "s1", UserID: "u1", Type: EventLogin, Timestamp: -1},
	}
	for name, e := range cases {
		t.Run(name, func(t *testing.T) {
			err := e.ValidateForIngest()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, ErrInvalidEvent) {
				t.Fatalf("error %v does not wrap ErrInvalidEvent", err)
			}
		})
	}
}

func TestEventJSONRoundTrip(t *testing.T) {
	in := Event{
		EventID:         "evt_1",
		SessionID:       "sess_123",
		UserID:          "user_001",
		Type:            EventBeneficiaryCreated,
		Timestamp:       1_788_594_121,
		Metadata:        map[string]any{"beneficiary_id": "ben_93"},
		Seq:             4,
		IngestedAtNanos: 1_788_594_121_000_000_123,
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}

	// Wire field names must match prd.md §33.
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"session_id", "user_id", "type", "timestamp", "metadata", "seq", "ingested_at_nanos", "event_id"} {
		if _, ok := m[k]; !ok {
			t.Errorf("marshaled event missing key %q: %s", k, raw)
		}
	}

	var out Event
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out.SessionID != in.SessionID || out.Type != in.Type || out.Seq != in.Seq ||
		out.Timestamp != in.Timestamp || out.IngestedAtNanos != in.IngestedAtNanos {
		t.Errorf("round trip mismatch:\n in=%+v\nout=%+v", in, out)
	}
	if out.Metadata["beneficiary_id"] != "ben_93" {
		t.Errorf("metadata lost: %+v", out.Metadata)
	}
}

func TestEventJSONOmitsEmptyServerFields(t *testing.T) {
	raw, err := json.Marshal(Event{SessionID: "s1", UserID: "u1", Type: EventLogin})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	for _, k := range []string{"seq", "ingested_at_nanos", "event_id", "metadata"} {
		if _, ok := m[k]; ok {
			t.Errorf("expected %q to be omitted when empty: %s", k, raw)
		}
	}
}

func TestIntentResponseJSON(t *testing.T) {
	r := IntentResponse{
		SessionID: "s1",
		Hypotheses: map[string]float64{
			IntentLegitimate: 0.18, IntentAccidental: 0.07,
			IntentSocialEngineering: 0.68, IntentAccountTakeover: 0.07,
		},
		Uncertainty: 0.31,
		Action:      ActionProbe,
		Evidence:    []string{"new_beneficiary"},
		EventsSeen:  5,
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var out IntentResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out.Action != ActionProbe || out.EventsSeen != 5 || len(out.Hypotheses) != 4 {
		t.Errorf("round trip mismatch: %+v", out)
	}
}
