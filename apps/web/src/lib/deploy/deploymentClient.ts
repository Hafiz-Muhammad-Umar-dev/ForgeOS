import type { Deployment, DeploymentLog, DeploymentMetric, EnvVariable, HealthCheck, TimelineStage } from "../../types/deployment";

const BASE_URL: string =
  (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "/v1";

const TOKEN_KEY = "forge_token";

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  const token = localStorage.getItem(TOKEN_KEY);
  if (token !== null) {
    headers["authorization"] = `Bearer ${token}`;
  }
  const url = `${BASE_URL}${path}`;
  const response = await globalThis.fetch(url, {
    ...init,
    headers: { ...headers, ...(init?.headers as Record<string, string>) },
  });
  if (!response.ok) throw new Error(`API error: ${String(response.status)}`);
  return (await response.json()) as T;
}

export async function listDeployments(project?: string, status?: string, page = 1, limit = 20): Promise<Deployment[]> {
  const params = new URLSearchParams({ page: String(page), limit: String(limit) });
  if (project !== undefined) params.set("project", project);
  if (status !== undefined) params.set("status", status);
  return apiFetch<Deployment[]>(`/deployments?${params.toString()}`);
}

export async function getDeployment(id: string): Promise<Deployment> {
  return apiFetch<Deployment>(`/deployments/${encodeURIComponent(id)}`);
}

export async function getDeploymentLogs(id: string, tail?: number): Promise<DeploymentLog[]> {
  const params = new URLSearchParams();
  if (tail !== undefined) params.set("tail", String(tail));
  return apiFetch<DeploymentLog[]>(`/deployments/${encodeURIComponent(id)}/logs?${params.toString()}`);
}

export async function getDeploymentMetrics(id: string): Promise<DeploymentMetric[]> {
  return apiFetch<DeploymentMetric[]>(`/deployments/${encodeURIComponent(id)}/metrics`);
}

export async function getDeploymentTimeline(id: string): Promise<TimelineStage[]> {
  return apiFetch<TimelineStage[]>(`/deployments/${encodeURIComponent(id)}/timeline`);
}

export async function getHealthChecks(id: string): Promise<HealthCheck[]> {
  return apiFetch<HealthCheck[]>(`/deployments/${encodeURIComponent(id)}/health`);
}

export async function getEnvVariables(project: string): Promise<EnvVariable[]> {
  return apiFetch<EnvVariable[]>(`/deployments/${encodeURIComponent(project)}/env`);
}

export async function setEnvVariable(project: string, key: string, value: string, isSecret: boolean): Promise<void> {
  await apiFetch(`/deployments/${encodeURIComponent(project)}/env`, {
    method: "POST",
    body: JSON.stringify({ key, value, is_secret: isSecret }),
  });
}

export async function deleteEnvVariable(project: string, key: string): Promise<void> {
  await apiFetch(`/deployments/${encodeURIComponent(project)}/env/${encodeURIComponent(key)}`, {
    method: "DELETE",
  });
}

export async function startDeployment(id: string): Promise<void> {
  await apiFetch(`/deployments/${encodeURIComponent(id)}/start`, { method: "POST" });
}

export async function stopDeployment(id: string): Promise<void> {
  await apiFetch(`/deployments/${encodeURIComponent(id)}/stop`, { method: "POST" });
}

export async function restartDeployment(id: string): Promise<void> {
  await apiFetch(`/deployments/${encodeURIComponent(id)}/restart`, { method: "POST" });
}

export async function pauseDeployment(id: string): Promise<void> {
  await apiFetch(`/deployments/${encodeURIComponent(id)}/pause`, { method: "POST" });
}

export async function resumeDeployment(id: string): Promise<void> {
  await apiFetch(`/deployments/${encodeURIComponent(id)}/resume`, { method: "POST" });
}

export async function deleteDeployment(id: string): Promise<void> {
  await apiFetch(`/deployments/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export async function rollbackDeployment(id: string, targetId: string): Promise<void> {
  await apiFetch(`/deployments/${encodeURIComponent(id)}/rollback`, {
    method: "POST",
    body: JSON.stringify({ target_id: targetId }),
  });
}
