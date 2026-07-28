import { useCallback, useEffect, useState } from "react";
import type { Deployment } from "../types/deployment";
import { listDeployments } from "../lib/deploy/deploymentClient";

interface UseDeploymentsOptions {
  readonly project?: string;
  readonly status?: string;
  readonly page?: number;
  readonly limit?: number;
}

interface UseDeploymentsResult {
  readonly deployments: Deployment[];
  readonly isLoading: boolean;
  readonly error: string | null;
  readonly refresh: () => void;
}

export function useDeployments(options?: UseDeploymentsOptions): UseDeploymentsResult {
  const { project, status, page = 1, limit = 20 } = options ?? {};
  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(() => {
    setIsLoading(true);
    setError(null);
    listDeployments(project, status, page, limit)
      .then(setDeployments)
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : String(err));
      })
      .finally(() => {
        setIsLoading(false);
      });
  }, [project, status, page, limit]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { deployments, isLoading, error, refresh };
}
