import { useCallback, useEffect, useState } from "react";
import type { ExecutionPlan, ExecutionMetrics, ExecutionEvent } from "../types/execution";
import {
  getExecutionPlan,
  runExecution,
  pauseExecution,
  resumeExecution,
  stopExecution,
  getMetrics,
  getEvents,
} from "../lib/agents/agentClient";

interface UseExecutionResult {
  readonly plan: ExecutionPlan | null;
  readonly metrics: ExecutionMetrics | null;
  readonly events: ExecutionEvent[];
  readonly isRunning: boolean;
  readonly isLoading: boolean;
  readonly error: string | null;
  readonly run: () => void;
  readonly pause: () => void;
  readonly resume: () => void;
  readonly stop: () => void;
}

export function useExecution(intentId: string | undefined): UseExecutionResult {
  const [plan, setPlan] = useState<ExecutionPlan | null>(null);
  const [metrics, setMetrics] = useState<ExecutionMetrics | null>(null);
  const [events, setEvents] = useState<ExecutionEvent[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isRunning = plan?.status === "running";

  const refresh = useCallback(() => {
    if (intentId === undefined) return;
    getExecutionPlan(intentId)
      .then(setPlan)
      .catch(() => {});
    getMetrics(intentId)
      .then(setMetrics)
      .catch(() => {});
    getEvents(intentId)
      .then(setEvents)
      .catch(() => {});
  }, [intentId]);

  useEffect(() => {
    refresh();
    if (intentId === undefined) return;
    const interval = setInterval(refresh, 2000);
    return () => {
      clearInterval(interval);
    };
  }, [refresh, intentId]);

  const run = useCallback(() => {
    if (intentId === undefined) return;
    setIsLoading(true);
    setError(null);
    runExecution(intentId)
      .then(setPlan)
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : "Failed to run");
      })
      .finally(() => { setIsLoading(false); });
  }, [intentId]);

  const pause = useCallback(() => {
    if (intentId === undefined) return;
    pauseExecution(intentId).catch(() => {});
  }, [intentId]);

  const resume = useCallback(() => {
    if (intentId === undefined) return;
    resumeExecution(intentId).catch(() => {});
  }, [intentId]);

  const stop = useCallback(() => {
    if (intentId === undefined) return;
    stopExecution(intentId).catch(() => {});
  }, [intentId]);

  return { plan, metrics, events, isRunning, isLoading, error, run, pause, resume, stop };
}
