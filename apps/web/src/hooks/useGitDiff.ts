import { useCallback, useEffect, useState } from "react";
import type { GitDiff } from "../types/git";
import { getDiff } from "../lib/git/gitClient";

interface UseGitDiffResult {
  readonly diffs: GitDiff[];
  readonly isLoading: boolean;
  readonly refresh: () => void;
}

export function useGitDiff(commitId?: string, filePath?: string): UseGitDiffResult {
  const [diffs, setDiffs] = useState<GitDiff[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  const refresh = useCallback(() => {
    setIsLoading(true);
    getDiff(commitId, filePath)
      .then(setDiffs)
      .catch(() => {/* ignore */})
      .finally(() => { setIsLoading(false); });
  }, [commitId, filePath]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { diffs, isLoading, refresh };
}
