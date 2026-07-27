package query

import (
	"testing"
)

func TestIntentViewDefaults(t *testing.T) {
	v := IntentView{ID: "intent-1", OrgID: "org-1", Status: "completed"}
	if v.ID != "intent-1" {
		t.Errorf("id=%s", v.ID)
	}
	if v.OrgID != "org-1" {
		t.Errorf("orgId=%s", v.OrgID)
	}
	if v.Status != "completed" {
		t.Errorf("status=%s", v.Status)
	}
}

func TestTaskViewDefaults(t *testing.T) {
	v := TaskView{ID: "task-1", IntentID: "intent-1", Status: "done"}
	if v.IntentID != "intent-1" {
		t.Errorf("intentId=%s", v.IntentID)
	}
	if v.InputTokens != 0 {
		t.Errorf("inputTokens=%d", v.InputTokens)
	}
}

func TestSentinelErrors(t *testing.T) {
	if ErrIntentNotFound == nil {
		t.Fatal("ErrIntentNotFound is nil")
	}
	if ErrTaskNotFound == nil {
		t.Fatal("ErrTaskNotFound is nil")
	}
}

func TestMetricConstants(t *testing.T) {
	if MetricProjectionDuration == "" {
		t.Error("empty metric name")
	}
	if MetricProjectionSuccessTotal == "" {
		t.Error("empty metric name")
	}
	if MetricProjectionFailureTotal == "" {
		t.Error("empty metric name")
	}
}

// ---------------------------------------------------------------------------
// FakeQueryService tests
// ---------------------------------------------------------------------------

func TestFakeQueryServiceGetIntent(t *testing.T) {
	f := NewFakeQueryService()
	f.AddIntent(IntentView{ID: "i1", OrgID: "org-1", Status: "completed"})

	v, err := f.GetIntent(nil, "i1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if v.ID != "i1" {
		t.Errorf("id=%s", v.ID)
	}
	if f.GetCount.Load() != 1 {
		t.Errorf("count=%d", f.GetCount.Load())
	}
}

func TestFakeQueryServiceGetIntentNotFound(t *testing.T) {
	f := NewFakeQueryService()
	_, err := f.GetIntent(nil, "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFakeQueryServiceListIntents(t *testing.T) {
	f := NewFakeQueryService()
	f.AddIntent(IntentView{ID: "i1", OrgID: "org-1"})
	f.AddIntent(IntentView{ID: "i2", OrgID: "org-1"})
	f.AddIntent(IntentView{ID: "i3", OrgID: "org-2"})

	results, err := f.ListIntents(nil, "org-1", 10, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("results=%d want=2", len(results))
	}
}

func TestFakeQueryServiceListIntentsPagination(t *testing.T) {
	f := NewFakeQueryService()
	for i := 0; i < 5; i++ {
		id := string(rune('a' + i))
		f.AddIntent(IntentView{ID: "i" + string(id), OrgID: "org-1"})
	}

	results, err := f.ListIntents(nil, "org-1", 2, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("results=%d want=2", len(results))
	}
}

func TestFakeQueryServiceGetTask(t *testing.T) {
	f := NewFakeQueryService()
	f.AddTask(TaskView{ID: "t1", IntentID: "i1", Status: "done"})

	v, err := f.GetTask(nil, "t1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if v.IntentID != "i1" {
		t.Errorf("intentId=%s", v.IntentID)
	}
}

func TestFakeQueryServiceGetTaskNotFound(t *testing.T) {
	f := NewFakeQueryService()
	_, err := f.GetTask(nil, "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFakeQueryServiceListTasks(t *testing.T) {
	f := NewFakeQueryService()
	f.AddTask(TaskView{ID: "t1", IntentID: "i1"})
	f.AddTask(TaskView{ID: "t2", IntentID: "i1"})
	f.AddTask(TaskView{ID: "t3", IntentID: "i2"})

	results, err := f.ListTasks(nil, "i1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("results=%d want=2", len(results))
	}
}

func TestFakeQueryServicePing(t *testing.T) {
	f := NewFakeQueryService()
	if err := f.Ping(nil); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestFakeQueryServiceListTasksEmpty(t *testing.T) {
	f := NewFakeQueryService()
	results, err := f.ListTasks(nil, "unknown")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if results == nil {
		t.Error("should return empty slice, not nil")
	}
}
