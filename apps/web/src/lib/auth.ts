import { ApiError } from "./api";

const BASE_URL: string =
  (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "/v1";

export interface AuthUser {
  readonly sub: string;
  readonly org_id: string;
  readonly scopes?: string[];
}

export interface LoginRequest {
  readonly token: string;
}

export interface LoginResponse {
  readonly token: string;
  readonly user: AuthUser;
}

// ---------------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------------

export async function login(req: LoginRequest): Promise<LoginResponse> {
  const response = await fetch(`${BASE_URL}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ token: req.token }),
  });

  if (!response.ok) {
    const body = await parseSafe<{ error?: string }>(response);
    throw new ApiError(
      response.status,
      body?.error ?? `Login failed: HTTP ${String(response.status)}`,
    );
  }

  const data: LoginResponse = (await response.json()) as LoginResponse;
  return data;
}

// ---------------------------------------------------------------------------
// Verify /me
// ---------------------------------------------------------------------------

export async function fetchMe(token: string): Promise<AuthUser> {
  const response = await fetch(`${BASE_URL}/auth/me`, {
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

  const data: AuthUser = (await response.json()) as AuthUser;
  return data;
}

// ---------------------------------------------------------------------------
// Logout
// ---------------------------------------------------------------------------

export async function logout(token: string): Promise<void> {
  await fetch(`${BASE_URL}/auth/logout`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
  });
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
