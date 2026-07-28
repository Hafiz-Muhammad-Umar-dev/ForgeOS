import { useCallback, useEffect, useState } from "react";
import type { WorkspaceFolder } from "../types/workspace";
import { listFiles } from "../lib/workspace/workspaceClient";

interface UseWorkspaceResult {
  readonly root: WorkspaceFolder | null;
  readonly isLoading: boolean;
  readonly error: string | null;
  readonly refresh: () => void;
  readonly expandedFolders: Set<string>;
  readonly toggleFolder: (path: string) => void;
  readonly selectedFile: string | null;
  readonly selectFile: (path: string) => void;
}

export function useWorkspace(basePath = "/"): UseWorkspaceResult {
  const [root, setRoot] = useState<WorkspaceFolder | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedFolders, setExpandedFolders] = useState<Set<string>>(new Set(["/"]));
  const [selectedFile, setSelectedFile] = useState<string | null>(null);

  const refresh = useCallback(() => {
    setIsLoading(true);
    setError(null);
    listFiles(basePath)
      .then((folder) => {
        setRoot(folder);
        setExpandedFolders((prev) => new Set([...prev, "/"]));
      })
      .catch((err: unknown) => {
        const message = err instanceof Error ? err.message : "Failed to load workspace";
        setError(message);
      })
      .finally(() => {
        setIsLoading(false);
      });
  }, [basePath]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const toggleFolder = useCallback((path: string) => {
    setExpandedFolders((prev) => {
      const next = new Set(prev);
      if (next.has(path)) {
        next.delete(path);
      } else {
        next.add(path);
      }
      return next;
    });
  }, []);

  const selectFile = useCallback((path: string) => {
    setSelectedFile(path);
  }, []);

  return {
    root,
    isLoading,
    error,
    refresh,
    expandedFolders,
    toggleFolder,
    selectedFile,
    selectFile,
  };
}
