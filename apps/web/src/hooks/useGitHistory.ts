import { useCallback, useEffect, useState } from "react";
import type { GitCommit } from "../types/git";
import { getHistory } from "../lib/git/gitClient";

interface UseGitHistoryResult {
  readonly commits: GitCommit[];
  readonly isLoading: boolean;
  readonly refresh: () => void;
}

export function useGitHistory(limit = 50): UseGitHistoryResult {
  const [commits, setCommits] = useState<GitCommit[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  const refresh = useCallback(() => {
    setIsLoading(true);
    getHistory(undefined, limit)
      .then(setCommits)
      .catch(() => {/* ignore */})
      .finally(() => { setIsLoading(false); });
  }, [limit]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { commits, isLoading, refresh };
}
