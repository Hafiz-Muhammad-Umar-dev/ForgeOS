//go:build integration

package ingress

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/bus"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
)

// testIntegrationBus implements a minimal bus for integration tests.
type testIntegrationBus struct {
	PublishedSubject string
	PublishedData    []byte
}

func (b *testIntegrationBus) Connect(ctx context.Context) error { return nil }
func (b *testIntegrationBus) IsConnected() bool                 { return true }
func (b *testIntegrationBus) Close(ctx context.Context) error    { return nil }
func (b *testIntegrationBus) Publish(_ context.Context, subject string, data []byte) error {
	b.PublishedSubject = subject
	b.PublishedData = data
	return nil
}
func (b *testIntegrationBus) Subscribe(_ context.Context, subject string, handler bus.MessageHandler) (bus.Subscription, error) {
	return nil, nil
}

// TestIntegrationIntentCreatedEvent verifies the full flow:
// HTTP POST → RESTAdapter → bus.Publish(devos.intent.created).
func TestIntegrationIntentCreatedEvent(t *testing.T) {
	tb := &testIntegrationBus{}
	a := NewRESTAdapter(WithIngressBus(tb))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/intents":
			a.handleSubmitIntent(w, r)
		}
	}))
	defer srv.Close()

	body := `{"text":"build a blog","user_id":"user-42","org_id":"org-acme"}`
	resp, err := http.Post(srv.URL+"/v1/intents", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status: got=%d want=201", resp.StatusCode)
	}

	// Verify the event was published on the correct subject
	if tb.PublishedSubject != "devos.intent.created" {
		t.Errorf("subject: got=%s want=devos.intent.created", tb.PublishedSubject)
	}
	if tb.PublishedData == nil {
		t.Fatal("no event published")
	}

	// Deserialize and verify the event envelope
	env, err := event.Deserialize(tb.PublishedData)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}
	if env.Type != event.TypeIntentCreated {
		t.Errorf("type: got=%s want=%s", env.Type, event.TypeIntentCreated)
	}
	if env.OrgID != "org-acme" {
		t.Errorf("orgId: got=%s", env.OrgID)
	}
	if env.ProducedBy != "ingress" {
		t.Errorf("producedBy: got=%s", env.ProducedBy)
	}

	// Verify the serialized event contains the payload
	payloadJSON := string(tb.PublishedData)
	if !strings.Contains(payloadJSON, "build a blog") {
		t.Error("published event missing 'build a blog'")
	}
	if !strings.Contains(payloadJSON, "user-42") {
		t.Error("published event missing user_id")
	}
}

// TestIntegrationIntentCreatedWithGeneratedIDs verifies ID generation.
func TestIntegrationIntentCreatedWithGeneratedIDs(t *testing.T) {
	tb := &testIntegrationBus{}
	a := NewRESTAdapter(WithIngressBus(tb))
	srv := httptest.NewServer(http.HandlerFunc(a.handleSubmitIntent))
	defer srv.Close()

	// Submit with no IDs — adapter should generate them
	body := `{"text":"hello world"}`
	resp, err := http.Post(srv.URL+"/", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status: got=%d want=201", resp.StatusCode)
	}

	var result IntentResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.IntentID == "" {
		t.Error("intent_id should be generated")
	}
	if result.TraceID == "" {
		t.Error("trace_id should be generated")
	}
	if result.Status != "accepted" {
		t.Errorf("status=%s", result.Status)
	}
}

// TestIntegrationIntentCreatedWithTraceID verifies trace ID passthrough.
func TestIntegrationIntentCreatedWithTraceID(t *testing.T) {
	tb := &testIntegrationBus{}
	a := NewRESTAdapter(WithIngressBus(tb))
	srv := httptest.NewServer(http.HandlerFunc(a.handleSubmitIntent))
	defer srv.Close()

	body := `{"text":"test","trace_id":"trace-custom-123"}`
	resp, err := http.Post(srv.URL+"/", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	var result IntentResult
	json.NewDecoder(resp.Body).Decode(&result)
	if result.TraceID != "trace-custom-123" {
		t.Errorf("trace: got=%s want=trace-custom-123", result.TraceID)
	}

	// Also verify on the bus event
	env, _ := event.Deserialize(tb.PublishedData)
	if env.TraceID != "trace-custom-123" {
		t.Errorf("envelope traceId: got=%s", env.TraceID)
	}
}
