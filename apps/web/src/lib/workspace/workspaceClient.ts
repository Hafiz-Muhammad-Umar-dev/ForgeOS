import type { WorkspaceFile, WorkspaceFolder, FileContent } from "../../types/workspace";

const BASE_URL: string =
  (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "/v1";

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const token = (window as unknown as Record<string, string>)["__auth_token"];
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (token !== "") {
    headers["authorization"] = `Bearer ${token}`;
  }

  const response = await fetch(`${BASE_URL}${path}`, {
    ...init,
    headers: { ...headers, ...(init?.headers as Record<string, string>) },
  });

  if (!response.ok) {
    throw new Error(`API error: ${String(response.status)}`);
  }

  return (await response.json()) as T;
}

export async function listFiles(path: string): Promise<WorkspaceFolder> {
  return apiFetch<WorkspaceFolder>(`/workspace/files?path=${encodeURIComponent(path)}`);
}

export async function readFile(path: string): Promise<FileContent> {
  return apiFetch<FileContent>(`/workspace/files/${encodeURIComponent(path)}`);
}

export async function writeFile(path: string, content: string): Promise<void> {
  await apiFetch(`/workspace/files/${encodeURIComponent(path)}`, {
    method: "PUT",
    body: JSON.stringify({ content }),
  });
}

export async function createFile(path: string): Promise<WorkspaceFile> {
  return apiFetch<WorkspaceFile>(`/workspace/files/${encodeURIComponent(path)}`, {
    method: "POST",
  });
}

export async function deleteFile(path: string): Promise<void> {
  await apiFetch(`/workspace/files/${encodeURIComponent(path)}`, {
    method: "DELETE",
  });
}

export async function renameFile(oldPath: string, newPath: string): Promise<void> {
  await apiFetch(`/workspace/files/${encodeURIComponent(oldPath)}`, {
    method: "PATCH",
    body: JSON.stringify({ new_path: newPath }),
  });
}
