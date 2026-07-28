import { getAuthToken } from "./api";
import type { ConnectionState, StreamEvent, StreamEventType } from "../types/stream";

export interface SSEClientOptions {
  readonly url: string;
  readonly onEvent?: (event: StreamEvent) => void;
  readonly onStateChange?: (state: ConnectionState) => void;
  readonly onError?: (error: Error) => void;
  readonly lastEventId?: string;
  readonly maxRetries?: number;
}

const BASE_DELAY_MS = 1_000;
const MAX_DELAY_MS = 30_000;
const JITTER_MAX_MS = 5_000;
const MAX_RETRIES = 10;

export class SSEClient {
  private readonly url: string;
  private readonly onEvent?: (event: StreamEvent) => void;
  private readonly onStateChange?: (state: ConnectionState) => void;
  private readonly onError?: (error: Error) => void;
  private readonly maxRetries: number;

  private reader: ReadableStreamDefaultReader<Uint8Array> | null = null;
  private abortController: AbortController | null = null;
  private retryCount = 0;
  private lastEventId: string | undefined;
  private buffer = "";
  private isStopped = false;
  private currentState: ConnectionState = "disconnected";

  public constructor(options: SSEClientOptions) {
    this.url = options.url;
    this.onEvent = options.onEvent;
    this.onStateChange = options.onStateChange;
    this.onError = options.onError;
    this.maxRetries = options.maxRetries ?? MAX_RETRIES;
    this.lastEventId = options.lastEventId;
  }

  // -------------------------------------------------------------------------
  // Connection
  // -------------------------------------------------------------------------

  public connect(): void {
    if (this.isStopped) return;
    this.retryCount = 0;
    this.isStopped = false;
    this.startConnection();
  }

  public disconnect(): void {
    this.isStopped = true;
    this.retryCount = 0;
    this.cleanup();
    this.setState("disconnected");
  }

  // -------------------------------------------------------------------------
  // Internal
  // -------------------------------------------------------------------------

  private startConnection(): void {
    if (this.isStopped) return;

    this.cleanup();
    this.abortController = new AbortController();
    this.buffer = "";
    this.setState(this.retryCount > 0 ? "reconnecting" : "connecting");

    const token = getAuthToken();
    const headers: Record<string, string> = { accept: "text/event-stream" };
    if (token !== null) {
      headers["authorization"] = `Bearer ${token}`;
    }
    if (this.lastEventId !== undefined) {
      headers["last-event-id"] = this.lastEventId;
    }

    void this.performFetch(headers);
  }

  private async performFetch(headers: Record<string, string>): Promise<void> {
    const abortSignal = this.abortController?.signal;
    if (abortSignal === undefined) return;

    try {
      const response = await fetch(this.url, {
        headers,
        signal: abortSignal,
      });

      if (!response.ok) {
        throw new Error(`SSE request failed: HTTP ${String(response.status)}`);
      }

      const body = response.body;
      if (body === null) {
        throw new Error("SSE response body is null");
      }

      this.setState("connected");
      this.retryCount = 0;
      this.reader = body.getReader();

      await this.readLoop();
    } catch (err: unknown) {
      if (this.isStopped) return;
      if (err instanceof DOMException && err.name === "AbortError") return;

      const error = err instanceof Error ? err : new Error(String(err));
      this.onError?.(error);
      this.scheduleReconnect();
    }
  }

  private async readLoop(): Promise<void> {
    const decoder = new TextDecoder();

    while (!this.isStopped && this.reader !== null) {
      const { done, value } = await this.reader.read();
      if (done) break;

      this.buffer += decoder.decode(value, { stream: true });
      this.processBuffer();
    }

    // Stream ended naturally — reconnect.
    if (!this.isStopped) {
      this.scheduleReconnect();
    }
  }

  private processBuffer(): void {
    const lines = this.buffer.split("\n");
    // Keep the last incomplete line in the buffer.
    this.buffer = lines.pop() ?? "";

    let currentEvent: StreamEventType | null = null;
    let currentData = "";
    let currentId: string | undefined;
    let currentTimestamp = Date.now();

    for (const line of lines) {
      if (line.startsWith("event: ")) {
        currentEvent = line.slice(7).trim() as StreamEventType;
      } else if (line.startsWith("data: ")) {
        currentData = line.slice(6);
      } else if (line.startsWith("id: ")) {
        currentId = line.slice(4).trim();
        currentTimestamp = Date.now();
      } else if (line === "" && currentEvent !== null) {
        // Empty line signals the end of an event.
        {
          const parsed = tryParseJson(currentData);
          const event: StreamEvent = {
            id: currentId ?? String(currentTimestamp),
            type: currentEvent,
            data: currentData,
            parsed,
            timestamp: currentTimestamp,
          };
          this.lastEventId = event.id;
          this.onEvent?.(event);
        }
        currentEvent = null;
        currentData = "";
        currentId = undefined;
      }
      // Lines starting with ": " are comments (heartbeats) — ignored.
    }
  }

  private scheduleReconnect(): void {
    if (this.isStopped) return;
    if (this.retryCount >= this.maxRetries) {
      this.setState("error");
      this.onError?.(new Error(`SSE max retries (${String(this.maxRetries)}) exceeded`));
      return;
    }

    this.retryCount++;
    this.setState("reconnecting");

    const delay = calculateBackoff(this.retryCount);
    setTimeout(() => {
      this.startConnection();
    }, delay);
  }

  private cleanup(): void {
    if (this.reader !== null) {
      this.reader.cancel().catch(() => undefined);
      this.reader = null;
    }
    if (this.abortController !== null) {
      this.abortController.abort();
      this.abortController = null;
    }
  }

  private setState(state: ConnectionState): void {
    if (state !== this.currentState) {
      this.currentState = state;
      this.onStateChange?.(state);
    }
  }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function tryParseJson(text: string): Record<string, unknown> | null {
  try {
    const parsed = JSON.parse(text) as Record<string, unknown>;
    return parsed;
  } catch {
    return null;
  }
}

function calculateBackoff(retryCount: number): number {
  const exponential = Math.min(
    BASE_DELAY_MS * Math.pow(2, retryCount - 1),
    MAX_DELAY_MS,
  );
  const jitter = Math.random() * JITTER_MAX_MS;
  return exponential + jitter;
}
