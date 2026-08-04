// Package workspacefs provides PostgreSQL-backed persistence for workspace
// file trees. It follows the Handler → Service → Repository → Store (database)
// architecture.
//
// This replaces the previous mock file responses with a durable storage layer.
// The file tree is stored as flat records in the workspace_files table and
// assembled into a nested structure for API responses.
package workspacefs

import "time"

// Workspace is a workspace container.
type Workspace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Root      string    `json:"root"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WorkspaceFile is a stored file or folder record.
type WorkspaceFile struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Size        int       `json:"size"`
	Content     string    `json:"content,omitempty"`
	IsFolder    bool      `json:"is_folder"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// FileNode is a node in the nested file tree response.
type FileNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	Type     string      `json:"type"` // "file" or "folder"
	Size     int         `json:"size,omitempty"`
	Children []*FileNode `json:"children,omitempty"`
}

// FolderNode represents the directory listing response (WorkspaceFolder).
type FolderNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	Type     string      `json:"type"`
	Children []*FileNode `json:"children"`
}

// CreateFileRequest is the input for creating a file.
type CreateFileRequest struct {
	WorkspaceID string
	Path        string
	Content     string
	IsFolder    bool
}

// ReadFileRequest identifies a file to read.
type ReadFileRequest struct {
	WorkspaceID string
	Path        string
}

// UpdateFileRequest is the input for updating file content.
type UpdateFileRequest struct {
	WorkspaceID string
	Path        string
	Content     string
}

// RenameFileRequest is the input for renaming a file or folder.
type RenameFileRequest struct {
	WorkspaceID string
	OldPath     string
	NewPath     string
}

// DeleteFileRequest identifies a file to delete.
type DeleteFileRequest struct {
	WorkspaceID string
	Path        string
}
