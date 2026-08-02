import type { AgentInfo } from "../../types/agent";
import type { ExecutionPlan, ExecutionMetrics, ExecutionEvent } from "../../types/execution";

const BASE_URL: string =
  (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "/v1";

const TOKEN_KEY = "forge_token";

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  const token = localStorage.getItem(TOKEN_KEY);
  if (token !== null) {
    headers["authorization"] = `Bearer ${token}`;
  }
  const response = await fetch(`${BASE_URL}${path}`, {
    ...init,
    headers: { ...headers, ...(init?.headers as Record<string, string>) },
  });
  if (!response.ok) throw new Error(`API error: ${String(response.status)}`);
  return (await response.json()) as T;
}

export async function listAgents(): Promise<AgentInfo[]> {
  return apiFetch<AgentInfo[]>("/agents");
}

export async function getAgent(id: string): Promise<AgentInfo> {
  return apiFetch<AgentInfo>(`/agents/${encodeURIComponent(id)}`);
}

export async function getExecutionPlan(intentId: string): Promise<ExecutionPlan> {
  return apiFetch<ExecutionPlan>(`/plans/${encodeURIComponent(intentId)}`);
}

export async function runExecution(intentId: string): Promise<ExecutionPlan> {
  return apiFetch<ExecutionPlan>(`/executions/${encodeURIComponent(intentId)}/run`, {
    method: "POST",
  });
}

export async function pauseExecution(intentId: string): Promise<void> {
  await apiFetch(`/executions/${encodeURIComponent(intentId)}/pause`, { method: "POST" });
}

export async function resumeExecution(intentId: string): Promise<void> {
  await apiFetch(`/executions/${encodeURIComponent(intentId)}/resume`, { method: "POST" });
}

export async function stopExecution(intentId: string): Promise<void> {
  await apiFetch(`/executions/${encodeURIComponent(intentId)}/stop`, { method: "POST" });
}

export async function getMetrics(intentId: string): Promise<ExecutionMetrics> {
  return apiFetch<ExecutionMetrics>(`/executions/${encodeURIComponent(intentId)}/metrics`);
}

export async function getEvents(intentId: string): Promise<ExecutionEvent[]> {
  return apiFetch<ExecutionEvent[]>(`/executions/${encodeURIComponent(intentId)}/events`);
}
