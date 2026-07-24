package ingress

import (
	"context"
	"testing"
)

func TestIntentPayloadDefaults(t *testing.T) {
	p := IntentPayload{Text: "build an app"}
	if p.Text != "build an app" {
		t.Errorf("text=%s", p.Text)
	}
	if p.UserID != "" {
		t.Errorf("user should be empty, got=%s", p.UserID)
	}
	if p.OrgID != "" {
		t.Errorf("org should be empty, got=%s", p.OrgID)
	}
}

func TestIntentPayloadWithAttachments(t *testing.T) {
	p := IntentPayload{
		Text: "deploy",
		Attachments: []Attachment{
			{Name: "screenshot.png", MIMEType: "image/png", URI: "https://example.com/img.png"},
		},
	}
	if len(p.Attachments) != 1 {
		t.Fatalf("attachments=%d", len(p.Attachments))
	}
	if p.Attachments[0].Name != "screenshot.png" {
		t.Errorf("name=%s", p.Attachments[0].Name)
	}
	if p.Attachments[0].MIMEType != "image/png" {
		t.Errorf("mime=%s", p.Attachments[0].MIMEType)
	}
}

func TestAttachmentNoRawData(t *testing.T) {
	// Verify Attachment has no Data/Content/[]byte payload field.
	a := Attachment{Name: "f", MIMEType: "text", URI: "https://example.com/f.txt"}
	if a.URI != "https://example.com/f.txt" {
		t.Errorf("uri=%s", a.URI)
	}
}

func TestIntentResult(t *testing.T) {
	r := IntentResult{IntentID: "intent-abc", Status: "accepted", TraceID: "trace-xyz"}
	if r.IntentID != "intent-abc" {
		t.Errorf("id=%s", r.IntentID)
	}
	if r.Status != "accepted" {
		t.Errorf("status=%s", r.Status)
	}
	if r.TraceID != "trace-xyz" {
		t.Errorf("trace=%s", r.TraceID)
	}
}

func TestSentinelErrors(t *testing.T) {
	if ErrInvalidRequest == nil {
		t.Fatal("ErrInvalidRequest is nil")
	}
	if ErrInvalidRequest.Error() != "ingress: invalid request" {
		t.Errorf("msg=%s", ErrInvalidRequest.Error())
	}
}

// ---------------------------------------------------------------------------
// FakeIngress tests
// ---------------------------------------------------------------------------

func TestFakeIngressSubmit(t *testing.T) {
	fi := NewFakeIngress()

	result, err := fi.SubmitIntent(nil, IntentPayload{Text: "hello"})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if result.IntentID != "intent-fake-1" {
		t.Errorf("id=%s", result.IntentID)
	}
	if fi.SubmitCount.Load() != 1 {
		t.Errorf("count=%d", fi.SubmitCount.Load())
	}
}

func TestFakeIngressRecordsPayloads(t *testing.T) {
	fi := NewFakeIngress()

	fi.SubmitIntent(nil, IntentPayload{Text: "first", UserID: "user-1"})
	fi.SubmitIntent(nil, IntentPayload{Text: "second", UserID: "user-2"})

	if len(fi.ReceivedPayloads) != 2 {
		t.Fatalf("payloads=%d", len(fi.ReceivedPayloads))
	}
	if fi.ReceivedPayloads[0].Text != "first" {
		t.Errorf("text0=%s", fi.ReceivedPayloads[0].Text)
	}
	if fi.ReceivedPayloads[1].UserID != "user-2" {
		t.Errorf("user1=%s", fi.ReceivedPayloads[1].UserID)
	}
}

func TestFakeIngressCustomFunc(t *testing.T) {
	fi := NewFakeIngress()
	fi.SubmitFunc = func(_ context.Context, payload IntentPayload) (IntentResult, error) {
		return IntentResult{IntentID: "custom", Status: "accepted"}, nil
	}

	result, err := fi.SubmitIntent(nil, IntentPayload{Text: "custom"})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if result.IntentID != "custom" {
		t.Errorf("id=%s", result.IntentID)
	}
}
