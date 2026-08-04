package workspacefs

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"
)

// workspaceRow is a store.Row that scans a workspace.
type workspaceRow struct {
	id string
}

func (r *workspaceRow) Scan(dest ...any) error {
	now := time.Now()
	*(dest[0].(*string)) = r.id
	*(dest[1].(*string)) = "workspace-" + r.id
	*(dest[2].(*string)) = "/"
	*(dest[3].(*time.Time)) = now
	*(dest[4].(*time.Time)) = now
	return nil
}

// fileRow is a store.Row that scans a workspace file.
type fileRow struct {
	path   string
	folder bool
}

func (r *fileRow) Scan(dest ...any) error {
	now := time.Now()
	*(dest[0].(*string)) = "file-id"
	*(dest[1].(*string)) = "ws-1"
	name := r.path
	if idx := strings.LastIndex(r.path, "/"); idx >= 0 {
		name = r.path[idx+1:]
	}
	*(dest[2].(*string)) = name
	*(dest[3].(*string)) = r.path
	*(dest[4].(*int)) = 0
	*(dest[5].(*string)) = ""
	*(dest[6].(*bool)) = r.folder
	*(dest[7].(*time.Time)) = now
	*(dest[8].(*time.Time)) = now
	return nil
}

// emptyRows is a store.Rows that yields no rows.
type emptyRows struct{}

func (r *emptyRows) Next() bool             { return false }
func (r *emptyRows) Scan(dest ...any) error { return nil }
func (r *emptyRows) Close()                 {}

// newTestService returns a service backed by a FakeStore configured to return
// a workspace on workspace queries, a file on file queries, and an empty file
// list on list queries.
func newTestService() *Service {
	fs := store.NewFakeStore()
	fs.QueryRowFunc = func(ctx context.Context, sql string, args ...any) store.Row {
		if containsSqlWorkspace(sql) {
			return &workspaceRow{id: "ws-1"}
		}
		p := "/"
		if len(args) > 1 {
			p = args[1].(string)
		}
		return &fileRow{path: p}
	}
	fs.QueryFunc = func(ctx context.Context, sql string, args ...any) (store.Rows, error) {
		return &emptyRows{}, nil
	}
	return NewService(NewRepository(fs))
}

func containsSqlWorkspace(sql string) bool {
	return strings.Contains(sql, "FROM workspaces")
}

func TestCreateWorkspace_Valid(t *testing.T) {
	s := newTestService()
	w, err := s.CreateWorkspace(context.Background(), "my-workspace")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if w.ID == "" {
		t.Error("expected generated id")
	}
}

func TestCreateWorkspace_EmptyName(t *testing.T) {
	s := newTestService()
	_, err := s.CreateWorkspace(context.Background(), "   ")
	if err != ErrInvalidInput {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetWorkspace(t *testing.T) {
	s := newTestService()
	w, err := s.GetWorkspace(context.Background(), "ws-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if w.ID != "ws-1" {
		t.Errorf("id: got=%s", w.ID)
	}
}

func TestListWorkspaces(t *testing.T) {
	s := newTestService()
	ws, err := s.ListWorkspaces(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if ws == nil {
		t.Error("expected non-nil slice")
	}
}

func TestCreateFile_NormalizesPath(t *testing.T) {
	// Use a store where reads return not-found (duplicate check passes),
	// and exec records the normalized path.
	fs := store.NewFakeStore()
	fs.QueryRowFunc = func(ctx context.Context, sql string, args ...any) store.Row {
		return &notFoundRow{}
	}
	fs.QueryFunc = func(ctx context.Context, sql string, args ...any) (store.Rows, error) {
		return &emptyRows{}, nil
	}
	s := NewService(NewRepository(fs))

	f, err := s.CreateFile(context.Background(), CreateFileRequest{
		WorkspaceID: "ws-1", Path: "main.go", Content: "package main",
	})
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	if f == nil {
		t.Error("expected a file")
	}
	if f.Path != "/main.go" {
		t.Errorf("path: got=%s want=/main.go", f.Path)
	}
	if f.Content != "package main" {
		t.Errorf("content: got=%s", f.Content)
	}
}

// notFoundRow is a store.Row whose Scan returns an error.
type notFoundRow struct{}

func (r *notFoundRow) Scan(dest ...any) error {
	return store.ErrNoRows
}

func TestCreateFile_InvalidRoot(t *testing.T) {
	s := newTestService()
	_, err := s.CreateFile(context.Background(), CreateFileRequest{
		WorkspaceID: "ws-1", Path: "/",
	})
	if err != ErrInvalidInput {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestReadFile(t *testing.T) {
	fs := store.NewFakeStore()
	fs.QueryRowFunc = func(ctx context.Context, sql string, args ...any) store.Row {
		return &fileRow{path: args[1].(string)}
	}
	s := NewService(NewRepository(fs))

	f, err := s.ReadFile(context.Background(), "ws-1", "main.go")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if f == nil {
		t.Error("expected a file")
	}
}

func TestListFiles_BuildsTree(t *testing.T) {
	fs := store.NewFakeStore()
	fs.QueryFunc = func(ctx context.Context, sql string, args ...any) (store.Rows, error) {
		return &fakeFileRows{
			files: []WorkspaceFile{
				{Path: "/src", Name: "src", IsFolder: true},
				{Path: "/src/main.go", Name: "main.go"},
			},
		}, nil
	}
	s := NewService(NewRepository(fs))

	root, err := s.ListFiles(context.Background(), "ws-1", "/")
	if err != nil {
		t.Fatalf("list files: %v", err)
	}
	if root.Type != "folder" {
		t.Errorf("root type: got=%s", root.Type)
	}
	if len(root.Children) == 0 {
		t.Error("expected children")
	}
}

// fakeFileRows is a store.Rows that yields the given files.
type fakeFileRows struct {
	files []WorkspaceFile
	index int
}

func (r *fakeFileRows) Next() bool {
	return r.index < len(r.files)
}
func (r *fakeFileRows) Scan(dest ...any) error {
	f := r.files[r.index]
	r.index++
	now := time.Now()
	*(dest[0].(*string)) = f.ID
	*(dest[1].(*string)) = f.WorkspaceID
	*(dest[2].(*string)) = f.Name
	*(dest[3].(*string)) = f.Path
	*(dest[4].(*int)) = f.Size
	*(dest[5].(*string)) = f.Content
	*(dest[6].(*bool)) = f.IsFolder
	*(dest[7].(*time.Time)) = now
	*(dest[8].(*time.Time)) = now
	return nil
}
func (r *fakeFileRows) Close() {}
