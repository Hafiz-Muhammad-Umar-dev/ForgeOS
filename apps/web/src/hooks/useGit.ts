import { useCallback, useEffect, useState } from "react";
import type { GitStatus, GitFile, GitMetrics } from "../types/git";
import { getStatus, listFiles, getMetrics, stageFile, unstageFile, discardFile } from "../lib/git/gitClient";

interface UseGitResult {
  readonly status: GitStatus | null;
  readonly files: GitFile[];
  readonly metrics: GitMetrics | null;
  readonly isLoading: boolean;
  readonly error: string | null;
  readonly backendAvailable: boolean;
  readonly refresh: () => void;
  readonly stage: (path: string) => void;
  readonly unstage: (path: string) => void;
  readonly discard: (path: string) => void;
}

export function useGit(): UseGitResult {
  const [status, setStatus] = useState<GitStatus | null>(null);
  const [files, setFiles] = useState<GitFile[]>([]);
  const [metrics, setMetrics] = useState<GitMetrics | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [backendAvailable, setBackendAvailable] = useState(true);

  const refresh = useCallback(() => {
    setIsLoading(true);
    setError(null);
    Promise.all([
      getStatus().catch((err: unknown) => {
        setBackendAvailable(false);
        throw err;
      }),
      listFiles().catch((err: unknown) => {
        setBackendAvailable(false);
        throw err;
      }),
      getMetrics().catch(() => null as GitMetrics | null),
    ])
      .then(([s, f, m]) => {
        setStatus(s);
        setFiles(f);
        setMetrics(m);
        setBackendAvailable(true);
      })
      .catch((err: unknown) => {
        setError(String(err));
      })
      .finally(() => {
        setIsLoading(false);
      });
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const stage = useCallback((path: string) => {
    stageFile(path)
      .then(() => { refresh(); })
      .catch(() => {/* ignore */});
  }, [refresh]);

  const unstage = useCallback((path: string) => {
    unstageFile(path)
      .then(() => { refresh(); })
      .catch(() => {/* ignore */});
  }, [refresh]);

  const discard = useCallback((path: string) => {
    discardFile(path)
      .then(() => { refresh(); })
      .catch(() => {/* ignore */});
  }, [refresh]);

  return { status, files, metrics, isLoading, error, backendAvailable, refresh, stage, unstage, discard };
}
