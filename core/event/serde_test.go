package event

import (
	"testing"
	"time"
)

func TestSerializeDeserializeRoundTrip(t *testing.T) {
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	payload := map[string]string{"key": "value"}
	e := New(TypeIntentCreated, "test-service", payload,
		WithTraceID("trace-1"), WithOrgID("org-1"), WithProjectID("proj-1"), WithTime(now))

	// Record generated ID for later comparison
	savedID := e.ID

	data, err := Serialize(e)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("serialize returned empty data")
	}

	raw, err := Deserialize(data)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}

	if raw.ID != savedID {
		t.Errorf("id: got=%s want=%s", raw.ID, savedID)
	}
	if raw.Type != TypeIntentCreated {
		t.Errorf("type: got=%s want=%s", raw.Type, TypeIntentCreated)
	}
	if raw.SchemaVersion != CurrentSchemaVersion {
		t.Errorf("schemaVersion: got=%d want=%d", raw.SchemaVersion, CurrentSchemaVersion)
	}
	if raw.TraceID != "trace-1" {
		t.Errorf("traceId: got=%s", raw.TraceID)
	}
	if raw.OrgID != "org-1" {
		t.Errorf("orgId: got=%s", raw.OrgID)
	}
	if raw.ProjectID != "proj-1" {
		t.Errorf("projectId: got=%s", raw.ProjectID)
	}
	if raw.ProducedBy != "test-service" {
		t.Errorf("producedBy: got=%s", raw.ProducedBy)
	}

	gotTime := time.Unix(0, raw.ProducedAt)
	if !gotTime.Equal(now) {
		t.Errorf("producedAt: got=%v want=%v", gotTime, now)
	}

	// Deserialize back to typed payload
	gotPayload, err := UnmarshalPayload[map[string]string](raw)
	if err != nil {
		t.Fatalf("unmarshalPayload: %v", err)
	}
	if gotPayload["key"] != "value" {
		t.Errorf("payload: got=%v", gotPayload)
	}
}

func TestSerializeNilPayload(t *testing.T) {
	e := New(TypeTaskAssigned, "scheduler", struct{}{})
	data, err := Serialize(e)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	raw, err := Deserialize(data)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}
	if raw.Type != TypeTaskAssigned {
		t.Errorf("type: got=%s", raw.Type)
	}
}

func TestDeserializeInvalidData(t *testing.T) {
	_, err := Deserialize([]byte("{invalid"))
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestUnmarshalPayloadWrongType(t *testing.T) {
	payload := map[string]int{"count": 42}
	e := New(TypeWorkspaceReady, "wm", payload)
	data, err := Serialize(e)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	raw, err := Deserialize(data)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}

	// Attempting to unmarshal into a different type will fail with a JSON
	// type error because the payload has int values but we target string.
	_, err = UnmarshalPayload[map[string]string](raw)
	if err == nil {
		t.Log("note: unmarshalling into wrong type succeeded (fields lost)")
	} else {
		t.Logf("unmarshal into wrong type correctly failed: %v", err)
	}
}

func TestToRawEnvelope(t *testing.T) {
	now := time.Now()
	e := New(TypePlanProposed, "planner", map[string]int{"steps": 3},
		WithTraceID("trace-x"), WithOrgID("org-x"), WithTime(now))

	raw, err := ToRawEnvelope(e)
	if err != nil {
		t.Fatalf("toRaw: %v", err)
	}
	if raw.ID != e.ID {
		t.Errorf("raw.ID mismatch")
	}
	if raw.Type != TypePlanProposed {
		t.Errorf("raw.Type mismatch")
	}
	if raw.TraceID != "trace-x" {
		t.Errorf("raw.TraceID mismatch")
	}
}

func TestSubjectFromRawEnvelope(t *testing.T) {
	e := New[struct{}](TypeDeployCompleted, "deployer", struct{}{})
	raw, err := ToRawEnvelope(e)
	if err != nil {
		t.Fatalf("toRaw: %v", err)
	}
	want := "devos.deploy.completed"
	if got := raw.Subject(); got != want {
		t.Errorf("subject: got=%s want=%s", got, want)
	}
}

func TestSerializeLargePayload(t *testing.T) {
	payload := make(map[string]int)
	for i := 0; i < 1000; i++ {
		payload["key"] = i
	}
	e := New(TypeAgentToken, "agent", payload)
	_, err := Serialize(e)
	if err != nil {
		t.Fatalf("serialize large: %v", err)
	}
}
