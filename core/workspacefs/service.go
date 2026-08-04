package workspacefs

import (
	"context"
	"path"
	"strings"
)

// Service is the application service for workspace file operations.
// It enforces path normalization, duplicate checks, parent existence,
// and safe delete rules, then delegates to the repository.
type Service struct {
	repo *Repository
}

// NewService creates a workspacefs service backed by the given repository.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateWorkspace creates a new workspace.
func (s *Service) CreateWorkspace(ctx context.Context, name string) (*Workspace, error) {
	if strings.TrimSpace(name) == "" {
		return nil, ErrInvalidInput
	}
	return s.repo.CreateWorkspace(ctx, name)
}

// GetWorkspace returns a workspace by ID.
func (s *Service) GetWorkspace(ctx context.Context, id string) (*Workspace, error) {
	return s.repo.GetWorkspace(ctx, id)
}

// ListWorkspaces returns all workspaces.
func (s *Service) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	return s.repo.ListWorkspaces(ctx)
}

// CreateFile creates a file or folder, enforcing duplicate and parent checks.
func (s *Service) CreateFile(ctx context.Context, req CreateFileRequest) (*WorkspaceFile, error) {
	normalized := normalizePath(req.Path)
	if normalized == "/" {
		return nil, ErrInvalidInput
	}

	// Duplicate check.
	if _, err := s.repo.ReadFile(ctx, req.WorkspaceID, normalized); err == nil {
		return nil, ErrAlreadyExists
	}

	// Parent folder existence check (for non-root paths).
	parent := path.Dir(normalized)
	if parent != "/" {
		if _, err := s.repo.ReadFile(ctx, req.WorkspaceID, parent); err != nil {
			return nil, ErrParentNotFound
		}
	}

	return s.repo.CreateFile(ctx, req.WorkspaceID, normalized, req.Content, req.IsFolder)
}

// ReadFile returns a file by workspace and path.
func (s *Service) ReadFile(ctx context.Context, workspaceID, filePath string) (*WorkspaceFile, error) {
	return s.repo.ReadFile(ctx, workspaceID, filePath)
}

// UpdateFile updates file content. The file must exist.
func (s *Service) UpdateFile(ctx context.Context, req UpdateFileRequest) (*WorkspaceFile, error) {
	normalized := normalizePath(req.Path)
	existing, err := s.repo.ReadFile(ctx, req.WorkspaceID, normalized)
	if err != nil {
		return nil, ErrNotFound
	}
	if existing.IsFolder {
		return nil, ErrInvalidInput
	}
	if err := s.repo.UpdateFile(ctx, req.WorkspaceID, normalized, req.Content); err != nil {
		return nil, err
	}
	return s.repo.ReadFile(ctx, req.WorkspaceID, normalized)
}

// RenameFile renames a file or folder (and its descendants).
func (s *Service) RenameFile(ctx context.Context, req RenameFileRequest) error {
	oldNorm := normalizePath(req.OldPath)
	newNorm := normalizePath(req.NewPath)
	if _, err := s.repo.ReadFile(ctx, req.WorkspaceID, oldNorm); err != nil {
		return ErrNotFound
	}
	if _, err := s.repo.ReadFile(ctx, req.WorkspaceID, newNorm); err == nil {
		return ErrAlreadyExists
	}

	// Rename the node and any descendants.
	files, err := s.repo.ListFiles(ctx, req.WorkspaceID)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.Path == oldNorm {
			if err := s.repo.RenameFile(ctx, req.WorkspaceID, f.Path, newNorm); err != nil {
				return err
			}
		} else if strings.HasPrefix(f.Path, oldNorm+"/") {
			childNew := newNorm + strings.TrimPrefix(f.Path, oldNorm)
			if err := s.repo.RenameFile(ctx, req.WorkspaceID, f.Path, childNew); err != nil {
				return err
			}
		}
	}
	return nil
}

// DeleteFile deletes a file or folder. Non-empty folders are protected.
func (s *Service) DeleteFile(ctx context.Context, req DeleteFileRequest) error {
	normalized := normalizePath(req.Path)
	if normalized == "/" {
		return ErrInvalidInput
	}
	existing, err := s.repo.ReadFile(ctx, req.WorkspaceID, normalized)
	if err != nil {
		return ErrNotFound
	}

	// Safe delete: prevent deleting non-empty folders.
	if existing.IsFolder {
		files, err := s.repo.ListFiles(ctx, req.WorkspaceID)
		if err != nil {
			return err
		}
		for _, f := range files {
			if f.Path != normalized && strings.HasPrefix(f.Path, normalized+"/") {
				return ErrNotEmpty
			}
		}
	}

	return s.repo.DeleteFile(ctx, req.WorkspaceID, normalized)
}

// ListFiles returns a nested folder tree rooted at the given directory.
func (s *Service) ListFiles(ctx context.Context, workspaceID, dirPath string) (*FolderNode, error) {
	files, err := s.repo.ListFiles(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	dir := normalizePath(dirPath)
	root := &FolderNode{Name: path.Base(dir), Path: dir, Type: "folder", Children: []*FileNode{}}

	// Build a map from path → node for files under dir.
	nodeMap := map[string]*FileNode{}
	for _, f := range files {
		if !strings.HasPrefix(f.Path, dir) {
			continue
		}
		rel := strings.TrimPrefix(f.Path, dir)
		if rel == "" {
			continue
		}
		rel = strings.TrimPrefix(rel, "/")
		nodeMap[f.Path] = &FileNode{
			Name: f.Name, Path: f.Path,
			Type: typeFor(f.IsFolder), Size: f.Size,
			Children: []*FileNode{},
		}
	}

	// Assemble the tree.
	for p, node := range nodeMap {
		parent := path.Dir(p)
		if parentNode, ok := nodeMap[parent]; ok {
			parentNode.Children = append(parentNode.Children, node)
		} else if parent == dir || (dir == "/" && parent == "/") {
			root.Children = append(root.Children, node)
		} else {
			// Insert intermediate folders that were not explicitly created.
			root.Children = append(root.Children, node)
		}
	}

	return root, nil
}

func typeFor(isFolder bool) string {
	if isFolder {
		return "folder"
	}
	return "file"
}
