const BASE_URL: string =
  (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "/v1";

// ---------------------------------------------------------------------------
// Auth token management (set by AuthContext on login/logout)
// ---------------------------------------------------------------------------

let _token: string | null = null;
let _onUnauthorized: (() => void) | null = null;

export function setAuthToken(token: string | null): void {
  _token = token;
}

export function getAuthToken(): string | null {
  return _token;
}

export function onUnauthorized(handler: () => void): void {
  _onUnauthorized = handler;
}

// ---------------------------------------------------------------------------
// Error types
// ---------------------------------------------------------------------------

export class ApiError extends Error {
  public readonly status: number;
  public readonly code: string | undefined;

  public constructor(status: number, message: string, code?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

export interface ApiOptions {
  readonly signal?: AbortSignal;
  readonly timeout?: number;
}

// ---------------------------------------------------------------------------
// Request helpers
// ---------------------------------------------------------------------------

function buildHeaders(): Record<string, string> {
  const headers: Record<string, string> = {
    accept: "application/json",
  };
  if (_token !== null) {
    headers["authorization"] = `Bearer ${_token}`;
  }
  return headers;
}

async function request<T>(
  path: string,
  options: ApiOptions = {},
): Promise<T> {
  const { signal, timeout = 15_000 } = options;

  const controller = new AbortController();
  const timeoutId = setTimeout(() => {
    controller.abort();
  }, timeout);

  const combinedSignal: AbortSignal =
    signal !== undefined
      ? combineAbortSignals(signal, controller.signal)
      : controller.signal;

  try {
    const response = await fetch(`${BASE_URL}${path}`, {
      signal: combinedSignal,
      headers: buildHeaders(),
    });

    if (!response.ok) {
      // Signal session expiry to the auth layer.
      if (response.status === 401 || response.status === 403) {
        _onUnauthorized?.();
      }

      const body = await parseJsonSafe<{ error?: string; code?: string }>(
        response,
      );
      const errMsg: string = body?.error ?? `HTTP ${String(response.status)}`;
      throw new ApiError(response.status, errMsg, body?.code);
    }

    const data: T = (await response.json()) as T;
    return data;
  } finally {
    clearTimeout(timeoutId);
  }
}

async function parseJsonSafe<T>(
  response: Response,
): Promise<T | undefined> {
  try {
    const text = await response.text();
    if (text.length === 0) return undefined;
    return JSON.parse(text) as T;
  } catch {
    return undefined;
  }
}

function combineAbortSignals(...signals: AbortSignal[]): AbortSignal {
  const controller = new AbortController();
  for (const sig of signals) {
    if (sig.aborted) {
      controller.abort(sig.reason);
      return controller.signal;
    }
    sig.addEventListener(
      "abort",
      () => {
        controller.abort(sig.reason);
      },
      { once: true },
    );
  }
  return controller.signal;
}

// ---------------------------------------------------------------------------
// Public helpers
// ---------------------------------------------------------------------------

export function get<T>(path: string, options?: ApiOptions): Promise<T> {
  return request<T>(path, options);
}
