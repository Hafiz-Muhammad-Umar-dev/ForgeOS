package workspacefs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"
)

// Named SQL statements for workspaces and files.
const (
	sqlCreateWorkspace = `INSERT INTO workspaces (id, name, root) VALUES ($1, $2, $3)`

	sqlGetWorkspace = `SELECT id, name, root, created_at, updated_at FROM workspaces WHERE id = $1`

	sqlListWorkspaces = `SELECT id, name, root, created_at, updated_at FROM workspaces ORDER BY created_at DESC`

	sqlCreateFile = `INSERT INTO workspace_files
		(id, workspace_id, name, path, size, content, is_folder, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		ON CONFLICT (workspace_id, path) DO UPDATE SET
			content = EXCLUDED.content,
			size = EXCLUDED.size,
			updated_at = NOW()`

	sqlGetFile = `SELECT id, workspace_id, name, path, size, content, is_folder, created_at, updated_at
		FROM workspace_files WHERE workspace_id = $1 AND path = $2`

	sqlListFiles = `SELECT id, workspace_id, name, path, size, content, is_folder, created_at, updated_at
		FROM workspace_files WHERE workspace_id = $1 ORDER BY path`

	sqlDeleteFile = `DELETE FROM workspace_files WHERE workspace_id = $1 AND path = $2`

	sqlRenameFile = `UPDATE workspace_files SET name = $1, path = $2, updated_at = NOW()
		WHERE workspace_id = $3 AND path = $4`
)

// Repository provides database operations for workspaces and files.
type Repository struct {
	store store.Store
}

// NewRepository creates a workspacefs repository backed by the given store.
func NewRepository(s store.Store) *Repository {
	return &Repository{store: s}
}

// CreateWorkspace inserts a new workspace.
func (r *Repository) CreateWorkspace(ctx context.Context, name string) (*Workspace, error) {
	id := newID()
	if _, err := r.store.Exec(ctx, sqlCreateWorkspace, id, name, "/"); err != nil {
		return nil, fmt.Errorf("workspace: create: %w", err)
	}
	return r.GetWorkspace(ctx, id)
}

// GetWorkspace returns a workspace by ID.
func (r *Repository) GetWorkspace(ctx context.Context, id string) (*Workspace, error) {
	row := r.store.QueryRow(ctx, sqlGetWorkspace, id)
	var w Workspace
	if err := row.Scan(&w.ID, &w.Name, &w.Root, &w.CreatedAt, &w.UpdatedAt); err != nil {
		return nil, fmt.Errorf("workspace: get %s: %w", id, err)
	}
	return &w, nil
}

// ListWorkspaces returns all workspaces.
func (r *Repository) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	rows, err := r.store.Query(ctx, sqlListWorkspaces)
	if err != nil {
		return nil, fmt.Errorf("workspace: list: %w", err)
	}
	defer rows.Close()

	results := make([]Workspace, 0)
	for rows.Next() {
		var w Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.Root, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, fmt.Errorf("workspace: scan: %w", err)
		}
		results = append(results, w)
	}
	return results, nil
}

// CreateFile inserts or upserts a file record.
func (r *Repository) CreateFile(ctx context.Context, workspaceID, path, content string, isFolder bool) (*WorkspaceFile, error) {
	name := path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		name = path[idx+1:]
	}
	normalized := normalizePath(path)
	id := newID()
	now := time.Now()
	if _, err := r.store.Exec(ctx, sqlCreateFile,
		id, workspaceID, name, normalized, len(content), content, isFolder,
	); err != nil {
		return nil, fmt.Errorf("workspace: create file: %w", err)
	}
	return &WorkspaceFile{
		ID: id, WorkspaceID: workspaceID, Name: name, Path: normalized,
		Size: len(content), Content: content, IsFolder: isFolder,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

// ReadFile returns a file by workspace and path.
func (r *Repository) ReadFile(ctx context.Context, workspaceID, path string) (*WorkspaceFile, error) {
	row := r.store.QueryRow(ctx, sqlGetFile, workspaceID, normalizePath(path))
	var f WorkspaceFile
	if err := row.Scan(&f.ID, &f.WorkspaceID, &f.Name, &f.Path, &f.Size, &f.Content, &f.IsFolder, &f.CreatedAt, &f.UpdatedAt); err != nil {
		return nil, fmt.Errorf("workspace: read %s: %w", path, err)
	}
	return &f, nil
}

// UpdateFile updates file content and metadata.
func (r *Repository) UpdateFile(ctx context.Context, workspaceID, path, content string) error {
	name := path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		name = path[idx+1:]
	}
	if _, err := r.store.Exec(ctx, sqlCreateFile,
		newID(), workspaceID, name, normalizePath(path), len(content), content, false,
	); err != nil {
		return fmt.Errorf("workspace: update file: %w", err)
	}
	return nil
}

// DeleteFile removes a file record.
func (r *Repository) DeleteFile(ctx context.Context, workspaceID, path string) error {
	if _, err := r.store.Exec(ctx, sqlDeleteFile, workspaceID, normalizePath(path)); err != nil {
		return fmt.Errorf("workspace: delete file: %w", err)
	}
	return nil
}

// RenameFile updates a file's name and path.
func (r *Repository) RenameFile(ctx context.Context, workspaceID, oldPath, newPath string) error {
	name := newPath
	if idx := strings.LastIndex(newPath, "/"); idx >= 0 {
		name = newPath[idx+1:]
	}
	if _, err := r.store.Exec(ctx, sqlRenameFile,
		name, normalizePath(newPath), workspaceID, normalizePath(oldPath),
	); err != nil {
		return fmt.Errorf("workspace: rename file: %w", err)
	}
	return nil
}

// ListFiles returns all file records for a workspace as a flat list.
func (r *Repository) ListFiles(ctx context.Context, workspaceID string) ([]WorkspaceFile, error) {
	rows, err := r.store.Query(ctx, sqlListFiles, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("workspace: list files: %w", err)
	}
	defer rows.Close()

	results := make([]WorkspaceFile, 0)
	for rows.Next() {
		var f WorkspaceFile
		if err := rows.Scan(&f.ID, &f.WorkspaceID, &f.Name, &f.Path, &f.Size, &f.Content, &f.IsFolder, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("workspace: scan file: %w", err)
		}
		results = append(results, f)
	}
	return results, nil
}

// normalizePath ensures a consistent leading-slash path.
func normalizePath(p string) string {
	if p == "" || p[0] != '/' {
		return "/" + p
	}
	return p
}
