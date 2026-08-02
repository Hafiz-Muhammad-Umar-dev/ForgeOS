package intents

import (
	"context"
	"testing"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"
)

func TestCreateIntent_Valid(t *testing.T) {
	s := NewService(NewRepository(newFakeStoreWithCreate()))
	intent, err := s.CreateIntent(context.Background(), NewIntentRequest{
		Text:   "Build auth",
		UserID: "dev-admin",
		OrgID:  "org-1",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if intent.ID == "" {
		t.Error("expected a generated id")
	}
	if intent.Status != "pending" {
		t.Errorf("status: got=%s want=pending", intent.Status)
	}
}

func TestCreateIntent_EmptyText(t *testing.T) {
	s := NewService(NewRepository(newFakeStoreWithCreate()))
	_, err := s.CreateIntent(context.Background(), NewIntentRequest{Text: ""})
	if err != ErrInvalidInput {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateIntent_DefaultOrg(t *testing.T) {
	fs := store.NewFakeStore()
	fs.QueryRowFunc = func(ctx context.Context, sql string, args ...any) store.Row {
		return &intentRow{id: args[0].(string), org: "default"}
	}
	fs.QueryFunc = func(ctx context.Context, sql string, args ...any) (store.Rows, error) {
		return &emptyRows{}, nil
	}
	s := NewService(NewRepository(fs))
	intent, err := s.CreateIntent(context.Background(), NewIntentRequest{Text: "hello", OrgID: ""})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if intent.OrgID != "default" {
		t.Errorf("orgId: got=%s want=default", intent.OrgID)
	}
}

func TestListIntents_DefaultPagination(t *testing.T) {
	fs := newFakeStoreWithCreate()
	s := NewService(NewRepository(fs))
	intents, err := s.ListIntents(context.Background(), "", 0, -1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if intents == nil {
		t.Error("expected non-nil slice")
	}
	if fs.QueryCount.Load() == 0 {
		t.Error("expected a query to be issued")
	}
}

func TestListTasks(t *testing.T) {
	fs := newFakeStoreWithCreate()
	s := NewService(NewRepository(fs))
	tasks, err := s.ListTasks(context.Background(), "intent-1")
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if tasks == nil {
		t.Error("expected non-nil slice")
	}
	if fs.QueryCount.Load() == 0 {
		t.Error("expected a query to be issued")
	}
}

func TestGetIntent(t *testing.T) {
	fs := newFakeStoreWithCreate()
	s := NewService(NewRepository(fs))
	intent, err := s.GetIntent(context.Background(), "intent-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if intent == nil {
		t.Fatal("expected an intent")
	}
	if intent.ID != "intent-1" {
		t.Errorf("id: got=%s", intent.ID)
	}
}
