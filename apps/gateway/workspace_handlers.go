package gateway

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/workspacefs"
)

// ensureDefaultWorkspace lazily creates a default workspace for the file API.
// The frontend does not pass a workspace ID, so a single default workspace is
// used to keep the API contract unchanged.
func (g *Gateway) ensureDefaultWorkspace(r *http.Request) (string, error) {
	if g.defaultWsLoaded {
		return g.defaultWsID, nil
	}
	ws, err := g.workspaceSvc.CreateWorkspace(r.Context(), "default")
	if err != nil {
		// A workspace may already exist; try to use it.
		list, lerr := g.workspaceSvc.ListWorkspaces(r.Context())
		if lerr != nil || len(list) == 0 {
			return "", err
		}
		g.defaultWsID = list[0].ID
		g.defaultWsLoaded = true
		return g.defaultWsID, nil
	}
	g.defaultWsID = ws.ID
	g.defaultWsLoaded = true
	return g.defaultWsID, nil
}

// workspaceFilePath extracts the file path after "/v1/workspace/files/".
func workspaceFilePath(path string) string {
	const prefix = "/v1/workspace/files/"
	return strings.TrimPrefix(path, prefix)
}

// handleWorkspaceGet handles GET /v1/workspace/files[?path=] and
// GET /v1/workspace/files/{path}.
func (g *Gateway) handleWorkspaceGet(w http.ResponseWriter, r *http.Request, path string) {
	wsID, err := g.ensureDefaultWorkspace(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "workspace unavailable")
		return
	}

	// Listing: GET /v1/workspace/files?path=...
	if path == "/v1/workspace/files" || path == "/v1/workspace/files/" {
		dir := r.URL.Query().Get("path")
		if dir == "" {
			dir = "/"
		}
		tree, err := g.workspaceSvc.ListFiles(r.Context(), wsID, dir)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list files")
			return
		}
		writeJSON(w, http.StatusOK, tree)
		return
	}

	// Read file: GET /v1/workspace/files/{path}
	filePath := workspaceFilePath(path)
	if filePath == "" {
		writeError(w, http.StatusBadRequest, "file path is required")
		return
	}
	file, err := g.workspaceSvc.ReadFile(r.Context(), wsID, filePath)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"path":     file.Path,
		"content":  file.Content,
		"language": languageFor(file.Path),
	})
}

// handleWorkspacePost handles POST /v1/workspace/files/{path} (create).
func (g *Gateway) handleWorkspacePost(w http.ResponseWriter, r *http.Request, path string) {
	wsID, err := g.ensureDefaultWorkspace(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "workspace unavailable")
		return
	}
	filePath := workspaceFilePath(path)
	if filePath == "" {
		writeError(w, http.StatusBadRequest, "file path is required")
		return
	}
	file, err := g.workspaceSvc.CreateFile(r.Context(), workspacefs.CreateFileRequest{
		WorkspaceID: wsID, Path: filePath, Content: "", IsFolder: false,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name": file.Name, "path": file.Path, "type": "file", "size": file.Size,
	})
}

// handleWorkspacePut handles PUT /v1/workspace/files/{path} (write content).
func (g *Gateway) handleWorkspacePut(w http.ResponseWriter, r *http.Request, path string) {
	wsID, err := g.ensureDefaultWorkspace(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "workspace unavailable")
		return
	}
	filePath := workspaceFilePath(path)
	if filePath == "" {
		writeError(w, http.StatusBadRequest, "file path is required")
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	file, err := g.workspaceSvc.UpdateFile(r.Context(), workspacefs.UpdateFileRequest{
		WorkspaceID: wsID, Path: filePath, Content: req.Content,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"path": file.Path, "status": "ok"})
}

// handleWorkspaceDelete handles DELETE /v1/workspace/files/{path}.
func (g *Gateway) handleWorkspaceDelete(w http.ResponseWriter, r *http.Request, path string) {
	wsID, err := g.ensureDefaultWorkspace(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "workspace unavailable")
		return
	}
	filePath := workspaceFilePath(path)
	if filePath == "" {
		writeError(w, http.StatusBadRequest, "file path is required")
		return
	}
	if err := g.workspaceSvc.DeleteFile(r.Context(), workspacefs.DeleteFileRequest{
		WorkspaceID: wsID, Path: filePath,
	}); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleWorkspacePatch handles PATCH /v1/workspace/files/{oldPath} (rename).
func (g *Gateway) handleWorkspacePatch(w http.ResponseWriter, r *http.Request, path string) {
	wsID, err := g.ensureDefaultWorkspace(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "workspace unavailable")
		return
	}
	oldPath := workspaceFilePath(path)
	if oldPath == "" {
		writeError(w, http.StatusBadRequest, "old path is required")
		return
	}
	var req struct {
		NewPath string `json:"new_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.NewPath == "" {
		writeError(w, http.StatusBadRequest, "new_path is required")
		return
	}
	if err := g.workspaceSvc.RenameFile(r.Context(), workspacefs.RenameFileRequest{
		WorkspaceID: wsID, OldPath: oldPath, NewPath: req.NewPath,
	}); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "renamed"})
}

func languageFor(p string) string {
	ext := p[strings.LastIndex(p, ".")+1:]
	switch strings.ToLower(ext) {
	case "ts", "tsx":
		return "typescript"
	case "js", "jsx":
		return "javascript"
	case "json":
		return "json"
	case "md":
		return "markdown"
	case "go":
		return "go"
	case "py":
		return "python"
	default:
		return "plaintext"
	}
}
