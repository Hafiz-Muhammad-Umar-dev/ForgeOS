import type { GitStatus, GitBranch, GitCommit, GitDiff, GitFile, GitMetrics } from "../../types/git";

const BASE_URL: string =
  (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "/v1";

let backendAvailable = true;

export function isGitBackendAvailable(): boolean {
  return backendAvailable;
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  try {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };
    const token = sessionStorage.getItem("auth_token");
    if (token !== null) {
      headers["authorization"] = `Bearer ${token}`;
    }
    const url = `${BASE_URL}${path}`;
    const response = await globalThis.fetch(url, {
      ...init,
      headers: { ...headers, ...(init?.headers as Record<string, string>) },
    });
    if (!response.ok) throw new Error(`HTTP ${String(response.status)}`);
    return (await response.json()) as T;
  } catch {
    backendAvailable = false;
    throw new Error("Git backend unavailable");
  }
}

export async function getStatus(): Promise<GitStatus> {
  return apiFetch<GitStatus>("/git/status");
}

export async function listFiles(): Promise<GitFile[]> {
  return apiFetch<GitFile[]>("/git/files");
}

export async function stageFile(path: string): Promise<void> {
  await apiFetch("/git/stage", {
    method: "POST",
    body: JSON.stringify({ path }),
  });
}

export async function unstageFile(path: string): Promise<void> {
  await apiFetch("/git/unstage", {
    method: "POST",
    body: JSON.stringify({ path }),
  });
}

export async function discardFile(path: string): Promise<void> {
  await apiFetch(`/git/discard/${encodeURIComponent(path)}`, {
    method: "POST",
  });
}

export async function commit(message: string, description?: string, signOff?: boolean): Promise<string> {
  const result = await apiFetch<{ oid: string }>("/git/commit", {
    method: "POST",
    body: JSON.stringify({ message, description, sign_off: signOff }),
  });
  return result.oid;
}

export async function push(remote?: string, branch?: string): Promise<void> {
  await apiFetch("/git/push", {
    method: "POST",
    body: JSON.stringify({ remote, branch }),
  });
}

export async function pull(remote?: string, branch?: string): Promise<void> {
  await apiFetch("/git/pull", {
    method: "POST",
    body: JSON.stringify({ remote, branch }),
  });
}

export async function fetch(remote?: string): Promise<void> {
  await apiFetch("/git/fetch", {
    method: "POST",
    body: JSON.stringify({ remote }),
  });
}

export async function listBranches(): Promise<GitBranch[]> {
  return apiFetch<GitBranch[]>("/git/branches");
}

export async function createBranch(name: string, base?: string): Promise<void> {
  await apiFetch("/git/branches", {
    method: "POST",
    body: JSON.stringify({ name, base }),
  });
}

export async function checkoutBranch(name: string): Promise<void> {
  await apiFetch("/git/checkout", {
    method: "POST",
    body: JSON.stringify({ branch: name }),
  });
}

export async function deleteBranch(name: string): Promise<void> {
  await apiFetch("/git/branches", {
    method: "DELETE",
    body: JSON.stringify({ name }),
  });
}

export async function getHistory(path?: string, limit = 50): Promise<GitCommit[]> {
  const params = new URLSearchParams();
  if (path !== undefined) params.set("path", path);
  params.set("limit", String(limit));
  return apiFetch<GitCommit[]>(`/git/history?${params.toString()}`);
}

export async function getDiff(commitId?: string, path?: string): Promise<GitDiff[]> {
  const params = new URLSearchParams();
  if (commitId !== undefined) params.set("commit", commitId);
  if (path !== undefined) params.set("path", path);
  return apiFetch<GitDiff[]>(`/git/diff?${params.toString()}`);
}

export async function stash(): Promise<void> {
  await apiFetch("/git/stash", { method: "POST" });
}

export async function stashPop(): Promise<void> {
  await apiFetch("/git/stash-pop", { method: "POST" });
}

export async function getMetrics(): Promise<GitMetrics> {
  return apiFetch<GitMetrics>("/git/metrics");
}
