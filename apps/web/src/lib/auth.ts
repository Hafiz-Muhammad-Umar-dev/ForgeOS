import { ApiError } from "./api";

// Auth service runs on a separate port during development.
const AUTH_BASE_URL: string =
  (import.meta.env.VITE_AUTH_BASE_URL as string | undefined) ?? "http://localhost:8081";

export interface AuthUser {
  readonly id: string;
  readonly name: string;
  readonly role: string;
}

export interface LoginRequest {
  readonly username: string;
  readonly password: string;
}

export interface LoginResponse {
  readonly access_token: string;
  readonly token_type: string;
  readonly expires_in: number;
  readonly user: AuthUser;
}

// ---------------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------------

export async function login(req: LoginRequest): Promise<LoginResponse> {
  const response = await fetch(`${AUTH_BASE_URL}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username: req.username, password: req.password }),
  });

  if (!response.ok) {
    const body = await parseSafe<{ error?: string }>(response);
    throw new ApiError(
      response.status,
      body?.error ?? `Login failed: HTTP ${String(response.status)}`,
    );
  }

  return (await response.json()) as LoginResponse;
}

// ---------------------------------------------------------------------------
// Verify /me
// ---------------------------------------------------------------------------

export async function fetchMe(token: string): Promise<AuthUser> {
  const response = await fetch(`${AUTH_BASE_URL}/auth/me`, {
    headers: {
      Authorization: `Bearer ${token}`,
      accept: "application/json",
    },
  });

  if (!response.ok) {
    const body = await parseSafe<{ error?: string }>(response);
    throw new ApiError(
      response.status,
      body?.error ?? `Auth failed: HTTP ${String(response.status)}`,
    );
  }

  return (await response.json()) as AuthUser;
}

// ---------------------------------------------------------------------------
// Logout
// ---------------------------------------------------------------------------

export async function logout(token: string): Promise<void> {
  try {
    await fetch(`${AUTH_BASE_URL}/auth/logout`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
    });
  } catch {
    // Best-effort.
  }
}

// ---------------------------------------------------------------------------
// Token storage
// ---------------------------------------------------------------------------

const TOKEN_KEY = "forge_token";

export function getStoredToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function storeToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY);
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

async function parseSafe<T>(response: Response): Promise<T | undefined> {
  try {
    const text = await response.text();
    if (text.length === 0) return undefined;
    return JSON.parse(text) as T;
  } catch {
    return undefined;
  }
}
