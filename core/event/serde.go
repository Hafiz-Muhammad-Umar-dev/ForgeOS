// Package event defines the DevOS event envelope and the canonical event
// catalog shared across all services.
//
// This file adds type-erased serialization/deserialization for transport over
// the message bus (core/bus). Full type safety is preserved at the
// producer/consumer boundary; RawEnvelope bridges the generic EventEnvelope[T]
// and the byte wire format.
//
// See specs/02-specification/04-message-bus.md §4 (Event Envelope).
package event

import (
	"encoding/json"
	"fmt"
)

// RawEnvelope is the JSON-serializable, type-erased form of EventEnvelope.
// The Payload field holds the raw JSON bytes of the original typed payload;
// callers use UnmarshalPayload to recover the concrete type.
type RawEnvelope struct {
	ID            string          `json:"id"`
	Type          EventType       `json:"type"`
	SchemaVersion SchemaVersion   `json:"schemaVersion"`
	TraceID       string          `json:"traceId"`
	OrgID         string          `json:"orgId"`
	ProjectID     string          `json:"projectId,omitempty"`
	ProducedBy    string          `json:"producedBy"`
	ProducedAt    int64           `json:"producedAt"` // UnixNano for precision
	Payload       json.RawMessage `json:"payload"`
}

// Serialize marshals an EventEnvelope[T] to JSON bytes for transport.
// The generic payload is inlined into the Payload field as raw JSON.
func Serialize[T any](e EventEnvelope[T]) ([]byte, error) {
	raw, err := json.Marshal(e.Payload)
	if err != nil {
		return nil, fmt.Errorf("event: serialize payload: %w", err)
	}
	env := RawEnvelope{
		ID:            e.ID,
		Type:          e.Type,
		SchemaVersion: e.SchemaVersion,
		TraceID:       e.TraceID,
		OrgID:         e.OrgID,
		ProjectID:     e.ProjectID,
		ProducedBy:    e.ProducedBy,
		ProducedAt:    e.ProducedAt.UnixNano(),
		Payload:       raw,
	}
	b, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("event: serialize envelope: %w", err)
	}
	return b, nil
}

// Deserialize unmarshals JSON bytes into a RawEnvelope. Callers use
// UnmarshalPayload or the Payload field directly to recover the typed value.
func Deserialize(data []byte) (RawEnvelope, error) {
	var env RawEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return RawEnvelope{}, fmt.Errorf("event: deserialize: %w", err)
	}
	return env, nil
}

// UnmarshalPayload decodes the payload of a RawEnvelope into the target value.
// target must be a pointer to the expected payload type.
func UnmarshalPayload[T any](env RawEnvelope) (T, error) {
	var zero T
	if len(env.Payload) == 0 {
		return zero, fmt.Errorf("event: unmarshal %s: empty payload", env.Type)
	}
	var val T
	if err := json.Unmarshal(env.Payload, &val); err != nil {
		return zero, fmt.Errorf("event: unmarshal %s payload: %w", env.Type, err)
	}
	return val, nil
}

// ToRawEnvelope converts a generic EventEnvelope[T] to its type-erased form
// without marshalling to JSON bytes.
func ToRawEnvelope[T any](e EventEnvelope[T]) (RawEnvelope, error) {
	raw, err := json.Marshal(e.Payload)
	if err != nil {
		return RawEnvelope{}, fmt.Errorf("event: marshal payload: %w", err)
	}
	return RawEnvelope{
		ID:            e.ID,
		Type:          e.Type,
		SchemaVersion: e.SchemaVersion,
		TraceID:       e.TraceID,
		OrgID:         e.OrgID,
		ProjectID:     e.ProjectID,
		ProducedBy:    e.ProducedBy,
		ProducedAt:    e.ProducedAt.UnixNano(),
		Payload:       raw,
	}, nil
}

// Subject returns the NATS subject for this envelope.
func (e RawEnvelope) Subject() string {
	return e.Type.Subject().String()
}
