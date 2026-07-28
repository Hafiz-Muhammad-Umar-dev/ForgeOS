import { useCallback, useEffect, useState } from "react";
import type { Deployment, DeploymentLog, DeploymentMetric, TimelineStage, HealthCheck, EnvVariable } from "../types/deployment";
import {
  getDeployment, getDeploymentLogs, getDeploymentMetrics, getDeploymentTimeline,
  getHealthChecks, getEnvVariables, startDeployment, stopDeployment, restartDeployment,
  pauseDeployment, resumeDeployment, deleteDeployment,
} from "../lib/deploy/deploymentClient";

interface UseDeploymentResult {
  readonly deployment: Deployment | null;
  readonly logs: DeploymentLog[];
  readonly metrics: DeploymentMetric[];
  readonly timeline: TimelineStage[];
  readonly healthChecks: HealthCheck[];
  readonly envVars: EnvVariable[];
  readonly isLoading: boolean;
  readonly error: string | null;
  readonly start: () => void;
  readonly stop: () => void;
  readonly restart: () => void;
  readonly pause: () => void;
  readonly resume: () => void;
  readonly del: () => void;
  readonly refresh: () => void;
}

export function useDeployment(id: string | undefined): UseDeploymentResult {
  const [deployment, setDeployment] = useState<Deployment | null>(null);
  const [logs, setLogs] = useState<DeploymentLog[]>([]);
  const [metrics, setMetrics] = useState<DeploymentMetric[]>([]);
  const [timeline, setTimeline] = useState<TimelineStage[]>([]);
  const [healthChecks, setHealthChecks] = useState<HealthCheck[]>([]);
  const [envVars, setEnvVars] = useState<EnvVariable[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(() => {
    if (id === undefined) return;
    setIsLoading(true);
    setError(null);
    Promise.all([
      getDeployment(id).catch(() => null),
      getDeploymentLogs(id).catch(() => []),
      getDeploymentMetrics(id).catch(() => []),
      getDeploymentTimeline(id).catch(() => []),
      getHealthChecks(id).catch(() => []),
      getEnvVariables(id).catch(() => []),
    ])
      .then(([dep, l, m, t, h, e]) => {
        setDeployment(dep);
        setLogs(l);
        setMetrics(m);
        setTimeline(t);
        setHealthChecks(h);
        setEnvVars(e);
      })
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : String(err));
      })
      .finally(() => {
        setIsLoading(false);
      });
  }, [id]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const act = useCallback((fn: () => Promise<void>) => {
    fn().then(refresh).catch(() => undefined);
  }, [refresh]);

  return {
    deployment, logs, metrics, timeline, healthChecks, envVars, isLoading, error, refresh,
    start: useCallback(() => { act(() => id !== undefined ? startDeployment(id) : Promise.resolve()); }, [id, act]),
    stop: useCallback(() => { act(() => id !== undefined ? stopDeployment(id) : Promise.resolve()); }, [id, act]),
    restart: useCallback(() => { act(() => id !== undefined ? restartDeployment(id) : Promise.resolve()); }, [id, act]),
    pause: useCallback(() => { act(() => id !== undefined ? pauseDeployment(id) : Promise.resolve()); }, [id, act]),
    resume: useCallback(() => { act(() => id !== undefined ? resumeDeployment(id) : Promise.resolve()); }, [id, act]),
    del: useCallback(() => { act(() => id !== undefined ? deleteDeployment(id) : Promise.resolve()); }, [id, act]),
  };
}
