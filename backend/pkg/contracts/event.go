// Package contracts defines the stable wire and internal schema shared across
// the Parallax backend: event types, the normalized Event, and the decision /
// intent response. The runtime team and the intelligence team both code
// against this package, so changes here are deliberate and reviewed.
package contracts

import (
	"errors"
	"fmt"
	"strings"
)

// EventType enumerates every session or transaction event Parallax observes
// (prd.md §10). Values are upper-snake-case on the wire.
type EventType string

const (
	EventLogin               EventType = "LOGIN"
	EventLogout              EventType = "LOGOUT"
	EventDeviceSeen          EventType = "DEVICE_SEEN"
	EventDeviceChanged       EventType = "DEVICE_CHANGED"
	EventPasswordChanged     EventType = "PASSWORD_CHANGED"
	EventPINChanged          EventType = "PIN_CHANGED"
	EventBeneficiaryCreated  EventType = "BENEFICIARY_CREATED"
	EventBeneficiaryModified EventType = "BENEFICIARY_MODIFIED"
	EventTransferStarted     EventType = "TRANSFER_STARTED"
	EventAmountEntered       EventType = "AMOUNT_ENTERED"
	EventOTPRequested        EventType = "OTP_REQUESTED"
	EventOTPVerified         EventType = "OTP_VERIFIED"
	EventTransferCompleted   EventType = "TRANSFER_COMPLETED"
	EventTransferFailed      EventType = "TRANSFER_FAILED"
	EventIntentProbeStarted  EventType = "INTENT_PROBE_STARTED"
	EventIntentProbeResponse EventType = "INTENT_PROBE_RESPONSE"
)

var knownEventTypes = map[EventType]struct{}{
	EventLogin: {}, EventLogout: {}, EventDeviceSeen: {}, EventDeviceChanged: {},
	EventPasswordChanged: {}, EventPINChanged: {}, EventBeneficiaryCreated: {},
	EventBeneficiaryModified: {}, EventTransferStarted: {}, EventAmountEntered: {},
	EventOTPRequested: {}, EventOTPVerified: {}, EventTransferCompleted: {},
	EventTransferFailed: {}, EventIntentProbeStarted: {}, EventIntentProbeResponse: {},
}

// Valid reports whether t is a recognized event type.
func (t EventType) Valid() bool {
	_, ok := knownEventTypes[t]
	return ok
}

// Event is a single observed event.
//
// Client-supplied fields (EventID, SessionID, UserID, Type, Timestamp,
// Metadata) arrive on POST /v1/events. The ingestion pipeline fills the
// server-owned fields (Seq, IngestedAtNanos) and guarantees EventID and
// Timestamp are set before the event reaches any processor.
type Event struct {
	// EventID is a stable unique id. Generated on ingest if the client omits it.
	EventID string `json:"event_id,omitempty"`
	// SessionID groups events into one session trajectory. Required.
	SessionID string `json:"session_id"`
	// UserID is the synthetic customer id. Required.
	UserID string `json:"user_id"`
	// Type is the event type. Required, must be Valid.
	Type EventType `json:"type"`
	// Timestamp is the client-reported event time, unix seconds. Filled with
	// the server receive time on ingest when zero.
	Timestamp int64 `json:"timestamp"`
	// Metadata carries type-specific fields (amount, beneficiary_id, ...).
	Metadata map[string]any `json:"metadata,omitempty"`

	// Seq is the per-session monotonic position, assigned by the session store.
	Seq uint64 `json:"seq,omitempty"`
	// IngestedAtNanos is the server receive time in unix nanoseconds, used for
	// sub-second sequence and latency analysis.
	IngestedAtNanos int64 `json:"ingested_at_nanos,omitempty"`
}

// ErrInvalidEvent is the sentinel for all client-side validation failures.
var ErrInvalidEvent = errors.New("invalid event")

// ValidateForIngest checks the client-supplied fields. It does not require the
// server-owned fields (Seq, IngestedAtNanos), which the pipeline fills in.
func (e Event) ValidateForIngest() error {
	switch {
	case strings.TrimSpace(e.SessionID) == "":
		return fmt.Errorf("%w: session_id is required", ErrInvalidEvent)
	case strings.TrimSpace(e.UserID) == "":
		return fmt.Errorf("%w: user_id is required", ErrInvalidEvent)
	case e.Type == "":
		return fmt.Errorf("%w: type is required", ErrInvalidEvent)
	case !e.Type.Valid():
		return fmt.Errorf("%w: unknown type %q", ErrInvalidEvent, e.Type)
	case e.Timestamp < 0:
		return fmt.Errorf("%w: timestamp must not be negative", ErrInvalidEvent)
	}
	return nil
}
