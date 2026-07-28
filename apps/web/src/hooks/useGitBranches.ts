import { useCallback, useEffect, useState } from "react";
import type { GitBranch } from "../types/git";
import { listBranches, createBranch, checkoutBranch, deleteBranch } from "../lib/git/gitClient";

interface UseGitBranchesResult {
  readonly branches: GitBranch[];
  readonly currentBranch: string | null;
  readonly isLoading: boolean;
  readonly refresh: () => void;
  readonly create: (name: string, base?: string) => void;
  readonly checkout: (name: string) => void;
  readonly delete: (name: string) => void;
}

export function useGitBranches(): UseGitBranchesResult {
  const [branches, setBranches] = useState<GitBranch[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  const refresh = useCallback(() => {
    setIsLoading(true);
    listBranches()
      .then(setBranches)
      .catch(() => {/* ignore */})
      .finally(() => { setIsLoading(false); });
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const currentBranch = branches.find((b) => b.isCurrent)?.name ?? null;

  const create = useCallback((name: string, base?: string) => {
    createBranch(name, base)
      .then(refresh)
      .catch(() => {/* ignore */});
  }, [refresh]);

  const checkout = useCallback((name: string) => {
    checkoutBranch(name)
      .then(refresh)
      .catch(() => {/* ignore */});
  }, [refresh]);

  const del = useCallback((name: string) => {
    deleteBranch(name)
      .then(refresh)
      .catch(() => {/* ignore */});
  }, [refresh]);

  return { branches, currentBranch, isLoading, refresh, create, checkout, delete: del };
}
