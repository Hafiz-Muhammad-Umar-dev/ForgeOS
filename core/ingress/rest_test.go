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

// ---------------------------------------------------------------------------
// REST adapter unit tests using httptest.Server
// ---------------------------------------------------------------------------

// testBus is a minimal bus stub for testing the REST adapter.
type testBus struct {
	Published   []byte
	LastSubject string
}

func (b *testBus) Connect(ctx context.Context) error { return nil }
func (b *testBus) IsConnected() bool                 { return true }
func (b *testBus) Close(ctx context.Context) error   { return nil }
func (b *testBus) Publish(_ context.Context, subject string, data []byte) error {
	b.LastSubject = subject
	b.Published = data
	return nil
}
func (b *testBus) Subscribe(_ context.Context, subject string, handler bus.MessageHandler) (bus.Subscription, error) {
	return nil, nil
}

func newTestAdapter(t *testing.T, bus bus.BusPort) *RESTAdapter {
	t.Helper()
	return NewRESTAdapter(
		WithIngressBus(bus),
		WithListenAddr(":0"),
	)
}

func serveTest(t *testing.T, a *RESTAdapter) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Route to the appropriate handler
		switch r.URL.Path {
		case "/v1/intents":
			a.handleSubmitIntent(w, r)
		case "/healthz":
			a.handleHealthz(w, r)
		case "/readyz":
			a.handleReadyz(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestRESTAdapterSubmitIntent(t *testing.T) {
	tb := &testBus{}
	a := newTestAdapter(t, tb)
	srv := serveTest(t, a)
	defer srv.Close()

	body := `{"text":"build an app","user_id":"user-1","org_id":"org-1"}`
	resp, err := http.Post(srv.URL+"/v1/intents", "application/json", bytes.NewReader([]byte(body)))
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
		t.Error("intent_id is empty")
	}
	if result.Status != "accepted" {
		t.Errorf("status=%s", result.Status)
	}
	if result.TraceID == "" {
		t.Error("trace_id is empty")
	}

	// Verify the event was published to the bus
	if tb.LastSubject != "devos.intent.created" {
		t.Errorf("subject: got=%s", tb.LastSubject)
	}
	if tb.Published == nil {
		t.Fatal("no data published")
	}

	// Deserialize and verify
	env, err := event.Deserialize(tb.Published)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}
	if env.Type != event.TypeIntentCreated {
		t.Errorf("type: got=%s", env.Type)
	}
	if env.ProducedBy != "ingress" {
		t.Errorf("producedBy: got=%s", env.ProducedBy)
	}
	if env.OrgID != "org-1" {
		t.Errorf("orgId: got=%s", env.OrgID)
	}
}

func TestRESTAdapterSubmitIntentEmptyText(t *testing.T) {
	tb := &testBus{}
	a := newTestAdapter(t, tb)
	srv := serveTest(t, a)
	defer srv.Close()

	body := `{"text":""}`
	resp, err := http.Post(srv.URL+"/v1/intents", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status: got=%d want=400", resp.StatusCode)
	}
}

func TestRESTAdapterSubmitIntentMethodNotAllowed(t *testing.T) {
	tb := &testBus{}
	a := newTestAdapter(t, tb)
	srv := serveTest(t, a)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/v1/intents")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status: got=%d want=405", resp.StatusCode)
	}
}

func TestRESTAdapterSubmitIntentInvalidJSON(t *testing.T) {
	tb := &testBus{}
	a := newTestAdapter(t, tb)
	srv := serveTest(t, a)
	defer srv.Close()

	body := `{invalid json}`
	resp, err := http.Post(srv.URL+"/v1/intents", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status: got=%d want=400", resp.StatusCode)
	}
}

func TestRESTAdapterSubmitIntentGeneratesIDs(t *testing.T) {
	tb := &testBus{}
	a := newTestAdapter(t, tb)
	srv := serveTest(t, a)
	defer srv.Close()

	// No IDs provided — adapter should generate them
	body := `{"text":"hello"}`
	resp, err := http.Post(srv.URL+"/v1/intents", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status: got=%d want=201", resp.StatusCode)
	}

	var result IntentResult
	json.NewDecoder(resp.Body).Decode(&result)
	if result.IntentID == "" {
		t.Error("intent_id should be generated")
	}
	if result.TraceID == "" {
		t.Error("trace_id should be generated")
	}
}

func TestRESTAdapterSubmitIntentLargeText(t *testing.T) {
	tb := &testBus{}
	a := NewRESTAdapter(WithIngressBus(tb), WithMaxTextLength(10))
	srv := serveTest(t, a)
	defer srv.Close()

	body := `{"text":"this is way too long for the limit"}`
	resp, err := http.Post(srv.URL+"/v1/intents", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status: got=%d want=400", resp.StatusCode)
	}
}

func TestRESTAdapterSubmitIntentPreservesTraceID(t *testing.T) {
	tb := &testBus{}
	a := newTestAdapter(t, tb)
	srv := serveTest(t, a)
	defer srv.Close()

	body := `{"text":"hello","trace_id":"trace-provided"}`
	resp, err := http.Post(srv.URL+"/v1/intents", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	var result IntentResult
	json.NewDecoder(resp.Body).Decode(&result)
	if result.TraceID != "trace-provided" {
		t.Errorf("trace: got=%s", result.TraceID)
	}
}

func TestRESTAdapterHealthz(t *testing.T) {
	tb := &testBus{}
	a := newTestAdapter(t, tb)
	srv := serveTest(t, a)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got=%d", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("status=%s", body["status"])
	}
}

func TestRESTAdapterReadyz(t *testing.T) {
	tb := &testBus{}
	a := newTestAdapter(t, tb)
	srv := serveTest(t, a)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got=%d want=200", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ready" {
		t.Errorf("status=%s", body["status"])
	}
}

func TestRESTAdapterLifecycle(t *testing.T) {
	tb := &testBus{}
	a := NewRESTAdapter(WithIngressBus(tb), WithListenAddr(":0"))

	if a.Name() != "ingress" {
		t.Errorf("name=%s", a.Name())
	}

	if err := a.Init(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Health before start
	h := a.Health()
	if h.Status == "UP" {
		t.Log("health before start:", h.Status)
	}

	if err := a.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}

	h = a.Health()
	if h.Status != "UP" {
		t.Errorf("health after start: got=%s", h.Status)
	}

	if err := a.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}

	h = a.Health()
	if h.Status == "UP" {
		t.Errorf("health after stop: got=%s", h.Status)
	}
}

func TestRESTAdapterInitNoBus(t *testing.T) {
	a := NewRESTAdapter()
	err := a.Init(context.Background())
	if err == nil {
		t.Fatal("expected error without bus")
	}
}

func TestRESTAdapterSubmitIntentWithAttachments(t *testing.T) {
	tb := &testBus{}
	a := newTestAdapter(t, tb)
	srv := serveTest(t, a)
	defer srv.Close()

	body := `{"text":"analyze","attachments":[{"name":"report.pdf","mime_type":"application/pdf","uri":"https://example.com/report.pdf"}]}`
	resp, err := http.Post(srv.URL+"/v1/intents", "application/json", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status: got=%d want=201", resp.StatusCode)
	}

	// Deserialize event payload to verify attachments
	if tb.Published != nil {
		env, _ := event.Deserialize(tb.Published)
		if env.OrgID == "" {
			t.Error("orgId should be default")
		}

		// Verify the published event has the correct type
		if !strings.Contains(string(tb.Published), "report.pdf") {
			t.Error("published event should contain attachment name")
		}
	}
}
