package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/apps/gateway/middleware"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/auth"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/intents"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/workspacefs"
)

// gwWorkspaceRow scans a workspace (5 fields).
type gwWorkspaceRow struct {
	id string
}

func (r *gwWorkspaceRow) Scan(dest ...any) error {
	now := time.Now()
	*(dest[0].(*string)) = r.id
	*(dest[1].(*string)) = "default"
	*(dest[2].(*string)) = "/"
	*(dest[3].(*time.Time)) = now
	*(dest[4].(*time.Time)) = now
	return nil
}

// gwFileRow scans a workspace file (9 fields).
type gwFileRow struct {
	path   string
	folder bool
}

func (r *gwFileRow) Scan(dest ...any) error {
	now := time.Now()
	*(dest[0].(*string)) = "file-id"
	*(dest[1].(*string)) = "ws-1"
	*(dest[2].(*string)) = "name"
	*(dest[3].(*string)) = r.path
	*(dest[4].(*int)) = 0
	*(dest[5].(*string)) = ""
	*(dest[6].(*bool)) = r.folder
	*(dest[7].(*time.Time)) = now
	*(dest[8].(*time.Time)) = now
	return nil
}

// memoryStore is an in-memory Store that persists workspace files in a map,
// giving handler tests realistic read/not-found semantics.
type memoryStore struct {
	workspaces map[string]workspacefs.Workspace
	files      map[string]workspacefs.WorkspaceFile // key: path
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		workspaces: map[string]workspacefs.Workspace{},
		files:      map[string]workspacefs.WorkspaceFile{},
	}
}

func (m *memoryStore) Exec(ctx context.Context, sql string, args ...any) (int64, error) {
	switch {
	case strings.Contains(sql, "INSERT INTO workspaces"):
		m.workspaces[args[0].(string)] = workspacefs.Workspace{ID: args[0].(string), Name: args[1].(string)}
		return 1, nil
	case strings.Contains(sql, "INSERT INTO workspace_files"):
		path := args[3].(string)
		m.files[path] = workspacefs.WorkspaceFile{
			ID: args[0].(string), WorkspaceID: args[1].(string),
			Name: args[2].(string), Path: path, Size: args[4].(int),
			Content: args[5].(string), IsFolder: args[6].(bool),
		}
		return 1, nil
	case strings.Contains(sql, "DELETE FROM workspace_files"):
		delete(m.files, args[1].(string))
		return 1, nil
	case strings.Contains(sql, "UPDATE workspace_files"):
		oldPath := args[3].(string)
		if f, ok := m.files[oldPath]; ok {
			f.Name = args[0].(string)
			f.Path = args[1].(string)
			delete(m.files, oldPath)
			m.files[f.Path] = f
		}
		return 1, nil
	}
	return 0, nil
}

func (m *memoryStore) QueryRow(ctx context.Context, sql string, args ...any) store.Row {
	if strings.Contains(sql, "FROM workspaces") {
		if w, ok := m.workspaces[args[0].(string)]; ok {
			return &gwWorkspaceRow{id: w.ID}
		}
		return &gwWorkspaceRow{id: "ws-1"}
	}
	// File query.
	path := args[1].(string)
	if f, ok := m.files[path]; ok {
		return &gwFileRow{path: f.Path, folder: f.IsFolder}
	}
	return &notFoundFileRow{}
}

func (m *memoryStore) Query(ctx context.Context, sql string, args ...any) (store.Rows, error) {
	return &storeEmptyRows{}, nil
}

func (m *memoryStore) Begin(ctx context.Context) (store.Tx, error) {
	return &gwTx{}, nil
}
func (m *memoryStore) Ping(ctx context.Context) error  { return nil }
func (m *memoryStore) Close(ctx context.Context) error { return nil }

// gwTx is a minimal transaction stub.
type gwTx struct{}

func (t *gwTx) Query(ctx context.Context, sql string, args ...any) (store.Rows, error) {
	return &storeEmptyRows{}, nil
}
func (t *gwTx) QueryRow(ctx context.Context, sql string, args ...any) store.Row {
	return &notFoundFileRow{}
}
func (t *gwTx) Exec(ctx context.Context, sql string, args ...any) (int64, error) { return 0, nil }
func (t *gwTx) Commit(ctx context.Context) error                                 { return nil }
func (t *gwTx) Rollback(ctx context.Context) error                               { return nil }

// notFoundFileRow scans with an error to simulate a missing file.
type notFoundFileRow struct{}

func (r *notFoundFileRow) Scan(dest ...any) error { return store.ErrNoRows }

// newWorkspaceTestGateway returns a gateway with an in-memory workspace store.
func newWorkspaceTestGateway() *Gateway {
	ms := newMemoryStore()
	wsSvc := workspacefs.NewService(workspacefs.NewRepository(ms))
	return NewGateway(DefaultGatewayConfig(), auth.NewFakeAuthProvider(),
		&testIngress{}, intents.NewService(intents.NewRepository(store.NewFakeStore())), wsSvc)
}

func TestWorkspaceListFiles(t *testing.T) {
	g := newWorkspaceTestGateway()
	req := httptest.NewRequest(http.MethodGet, "/v1/workspace/files?path=/", nil)
	rr := httptest.NewRecorder()

	g.handleAuthenticated(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got=%d want=200", rr.Code)
	}
	var tree workspacefs.FolderNode
	if err := json.NewDecoder(rr.Body).Decode(&tree); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if tree.Type != "folder" {
		t.Errorf("type: got=%s", tree.Type)
	}
}

func TestWorkspaceCreateFile(t *testing.T) {
	g := newWorkspaceTestGateway()
	req := httptest.NewRequest(http.MethodPost, "/v1/workspace/files/hello.txt", nil)
	rr := httptest.NewRecorder()

	g.handleAuthenticated(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got=%d want=200", rr.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["path"] != "/hello.txt" {
		t.Errorf("path: got=%v", body["path"])
	}
}

func TestWorkspaceWriteFile(t *testing.T) {
	g := newWorkspaceTestGateway()
	// Create first, then write.
	g.handleAuthenticated(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodPost, "/v1/workspace/files/test.txt", nil))

	payload := `{"content":"hello world"}`
	req := httptest.NewRequest(http.MethodPut, "/v1/workspace/files/test.txt", bytes.NewReader([]byte(payload)))
	rr := httptest.NewRecorder()

	g.handleAuthenticated(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got=%d want=200", rr.Code)
	}
}

func TestWorkspaceRenameFile(t *testing.T) {
	g := newWorkspaceTestGateway()
	g.handleAuthenticated(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodPost, "/v1/workspace/files/old.txt", nil))

	payload := `{"new_path":"/new.txt"}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/workspace/files/old.txt", bytes.NewReader([]byte(payload)))
	rr := httptest.NewRecorder()

	g.handleAuthenticated(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got=%d want=200", rr.Code)
	}
}

func TestWorkspaceDeleteFile(t *testing.T) {
	g := newWorkspaceTestGateway()
	g.handleAuthenticated(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodPost, "/v1/workspace/files/del.txt", nil))

	req := httptest.NewRequest(http.MethodDelete, "/v1/workspace/files/del.txt", nil)
	rr := httptest.NewRecorder()

	g.handleAuthenticated(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got=%d want=200", rr.Code)
	}
}

func TestWorkspaceReadFile(t *testing.T) {
	g := newWorkspaceTestGateway()
	g.handleAuthenticated(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodPost, "/v1/workspace/files/read.txt", nil))

	req := httptest.NewRequest(http.MethodGet, "/v1/workspace/files/read.txt", nil)
	rr := httptest.NewRecorder()

	g.handleAuthenticated(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got=%d want=200", rr.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["path"] != "/read.txt" {
		t.Errorf("path: got=%v", body["path"])
	}
}

func TestWorkspaceUnauthenticated(t *testing.T) {
	g := newWorkspaceTestGateway()
	// Without an Authorization header, the middleware rejects the request.
	handler := middleware.Authenticate(g.provider, http.HandlerFunc(g.handleAuthenticated))
	req := httptest.NewRequest(http.MethodGet, "/v1/workspace/files?path=/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got=%d want=401", rr.Code)
	}
}

func TestWorkspaceEnsureDefault(t *testing.T) {
	g := newWorkspaceTestGateway()
	req := httptest.NewRequest(http.MethodGet, "/v1/workspace/files?path=/", nil)
	wsID, err := g.ensureDefaultWorkspace(req)
	if err != nil {
		t.Fatalf("ensure default: %v", err)
	}
	if wsID == "" {
		t.Error("expected a workspace id")
	}
	// Second call should reuse the same workspace.
	wsID2, err := g.ensureDefaultWorkspace(req)
	if err != nil {
		t.Fatalf("ensure default 2: %v", err)
	}
	if wsID2 != wsID {
		t.Errorf("expected same workspace, got %s then %s", wsID, wsID2)
	}
}
