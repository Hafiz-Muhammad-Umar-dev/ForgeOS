import { getAuthToken } from "../api";
import {
  encodeMessage,
  decodeMessage,
  type WsMessage,
} from "./messageCodec";

export type ConnectionState =
  | "connecting"
  | "connected"
  | "reconnecting"
  | "disconnected"
  | "error";

export interface WsProviderOptions {
  readonly url: string;
  readonly intentId: string;
  readonly onMessage: (msg: WsMessage) => void;
  readonly onStateChange?: (state: ConnectionState) => void;
  readonly onError?: (error: Error) => void;
}

const BASE_DELAY_MS = 1_000;
const MAX_DELAY_MS = 30_000;
const MAX_RETRIES = 15;

export class WsProvider {
  private readonly options: WsProviderOptions;
  private ws: WebSocket | null = null;
  private retryCount = 0;
  private isStopped = false;
  private currentState: ConnectionState = "disconnected";
  private messageQueue: string[] = [];
  private isReady = false;

  public constructor(options: WsProviderOptions) {
    this.options = options;
  }

  // -------------------------------------------------------------------------
  // Public API
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
    this.isReady = false;
    this.messageQueue = [];
    this.closeSocket();
    this.setState("disconnected");
  }

  public send(type: string, data?: string): void {
    const msg = encodeMessage(type, data, this.options.intentId);
    if (this.isReady && this.ws !== null && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(msg);
    } else {
      this.messageQueue.push(msg);
    }
  }

  public getState(): ConnectionState {
    return this.currentState;
  }

  // -------------------------------------------------------------------------
  // Internal
  // -------------------------------------------------------------------------

  private startConnection(): void {
    if (this.isStopped) return;

    this.closeSocket();
    this.setState(this.retryCount > 0 ? "reconnecting" : "connecting");

    const token = getAuthToken();
    if (token === null) {
      this.onError(new Error("No auth token available"));
      this.setState("error");
      return;
    }

    try {
      this.ws = new WebSocket(this.options.url);
    } catch (err: unknown) {
      const error = err instanceof Error ? err : new Error(String(err));
      this.onError(error);
      this.scheduleReconnect();
      return;
    }

    const ws = this.ws;
    ws.onopen = () => {
      // Send auth handshake as the first message.
      ws.send(encodeMessage("auth", token));
    };

    this.ws.onmessage = (event: MessageEvent) => {
      this.handleMessage(event.data as string);
    };

    this.ws.onerror = () => {
      // onclose will fire after this, so we don't reconnect here.
    };

    this.ws.onclose = () => {
      this.isReady = false;
      if (!this.isStopped) {
        this.scheduleReconnect();
      }
    };
  }

  private handleMessage(raw: string): void {
    let msg: WsMessage;
    try {
      msg = decodeMessage(raw);
    } catch {
      return;
    }

    switch (msg.type) {
      case "auth_ok":
        this.isReady = true;
        this.retryCount = 0;
        this.setState("connected");
        // Send join message.
        this.ws?.send(encodeMessage("join", undefined, this.options.intentId));
        // Flush queued messages.
        this.flushQueue();
        break;

      case "auth_failed":
        this.onError(new Error(msg.error ?? "Authentication failed"));
        this.setState("error");
        this.disconnect();
        break;

      case "join_ok":
        // Room joined successfully.
        break;

      case "pong":
        break;

      default:
        // Forward to the consumer.
        this.options.onMessage(msg);
        break;
    }
  }

  private flushQueue(): void {
    const queue = this.messageQueue;
    this.messageQueue = [];
    for (const msg of queue) {
      if (this.isReady && this.ws !== null && this.ws.readyState === WebSocket.OPEN) {
        this.ws.send(msg);
      }
    }
  }

  private scheduleReconnect(): void {
    if (this.isStopped) return;
    if (this.retryCount >= MAX_RETRIES) {
      this.setState("error");
      this.onError(new Error(`Max reconnection attempts (${String(MAX_RETRIES)}) exceeded`));
      return;
    }

    this.retryCount++;
    this.setState("reconnecting");

    const delay = Math.min(
      BASE_DELAY_MS * Math.pow(2, this.retryCount - 1),
      MAX_DELAY_MS,
    );

    setTimeout(() => {
      this.startConnection();
    }, delay);
  }

  private closeSocket(): void {
    if (this.ws !== null) {
      this.ws.onopen = null;
      this.ws.onmessage = null;
      this.ws.onerror = null;
      this.ws.onclose = null;
      if (
        this.ws.readyState === WebSocket.OPEN ||
        this.ws.readyState === WebSocket.CONNECTING
      ) {
        this.ws.close(1000, "disconnect");
      }
      this.ws = null;
    }
  }

  private setState(state: ConnectionState): void {
    if (state !== this.currentState) {
      this.currentState = state;
      this.options.onStateChange?.(state);
    }
  }

  private onError(error: Error): void {
    this.options.onError?.(error);
  }
}
