import { useCallback, useEffect, useState } from "react";
import type { AgentInfo } from "../types/agent";
import { listAgents } from "../lib/agents/agentClient";

interface UseAgentsResult {
  readonly agents: AgentInfo[];
  readonly isLoading: boolean;
  readonly error: string | null;
  readonly refresh: () => void;
}

export function useAgents(refreshInterval = 5000): UseAgentsResult {
  const [agents, setAgents] = useState<AgentInfo[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(() => {
    listAgents()
      .then(setAgents)
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : "Failed to load agents");
      })
      .finally(() => {
        setIsLoading(false);
      });
  }, []);

  useEffect(() => {
    refresh();
    const interval = setInterval(refresh, refreshInterval);
    return () => {
      clearInterval(interval);
    };
  }, [refresh, refreshInterval]);

  return { agents, isLoading, error, refresh };
}
