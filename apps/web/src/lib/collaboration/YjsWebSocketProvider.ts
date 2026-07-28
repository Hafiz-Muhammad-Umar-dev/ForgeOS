import * as Y from "yjs";
import * as awarenessProtocol from "y-protocols/awareness";
import { WsProvider, type ConnectionState } from "./wsProvider";
import { encodeBase64, decodeBase64, type WsMessage } from "./messageCodec";

const BASE_URL: string =
  (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "/v1";

const WS_PROTOCOL = import.meta.env.VITE_WS_PROTOCOL as string | undefined;

function resolveWsUrl(path: string): string {
  const base = WS_PROTOCOL !== undefined ? WS_PROTOCOL : BASE_URL;
  if (base.startsWith("http")) {
    return base.replace(/^http/, "ws") + path;
  }
  if (base.startsWith("ws")) {
    return base + path;
  }
  // Relative path — use the current origin with WS protocol.
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${window.location.host}${base}${path}`;
}

export class YjsWebSocketProvider {
  public readonly doc: Y.Doc;
  public readonly awareness: awarenessProtocol.Awareness;
  public readonly wsProvider: WsProvider;

  private _synced = false;
  private _onSyncHandlers: Array<() => void> = [];

  public constructor(intentId: string) {
    this.doc = new Y.Doc();
    this.awareness = new awarenessProtocol.Awareness(this.doc);

    const url = resolveWsUrl("/v1/stream");

    this.wsProvider = new WsProvider({
      url,
      intentId,
      onMessage: (msg: WsMessage) => {
        this.handleMessage(msg);
      },
      onStateChange: () => {
        this.checkSync();
      },
      onError: (error: Error) => {
        console.error("collaboration: provider error:", error.message);
      },
    });

    // Observe Yjs updates and send them over the WebSocket.
    this.doc.on("update", (update: Uint8Array, origin: unknown) => {
      if (origin === this) return;
      const base64 = encodeBase64(update);
      this.wsProvider.send("update", base64);
    });

    // Observe awareness changes and broadcast them.
    this.awareness.on(
      "change",
      (
        changes: {
          added: number[];
          updated: number[];
          removed: number[];
        },
      ) => {
        const changedClients: number[] = [
          ...changes.added,
          ...changes.updated,
          ...changes.removed,
        ];
        const encodedUpdate =
          awarenessProtocol.encodeAwarenessUpdate(
            this.awareness,
            changedClients,
          );
        const base64 = encodeBase64(encodedUpdate);
        this.wsProvider.send("awareness", base64);
      },
    );

    this.wsProvider.connect();
  }

  // -------------------------------------------------------------------------
  // Public API
  // -------------------------------------------------------------------------

  public get synced(): boolean {
    return this._synced;
  }

  public get connectionState(): ConnectionState {
    return this.wsProvider.getState();
  }

  public onSync(handler: () => void): () => void {
    this._onSyncHandlers.push(handler);
    if (this._synced) {
      handler();
    }
    return () => {
      this._onSyncHandlers = this._onSyncHandlers.filter((h) => h !== handler);
    };
  }

  public disconnect(): void {
    this.wsProvider.disconnect();
  }

  public destroy(): void {
    this.disconnect();
    this.awareness.destroy();
    this.doc.destroy();
    this._onSyncHandlers = [];
  }

  // -------------------------------------------------------------------------
  // Message handling
  // -------------------------------------------------------------------------

  private handleMessage(msg: WsMessage): void {
    switch (msg.type) {
      case "update": {
        if (msg.data === undefined) break;
        const update = decodeBase64(msg.data);
        Y.applyUpdate(this.doc, update, this);
        this.checkSync();
        break;
      }

      case "awareness": {
        if (msg.data === undefined) break;
        const update = decodeBase64(msg.data);
        awarenessProtocol.applyAwarenessUpdate(
          this.awareness,
          update,
          this,
        );
        break;
      }

      default:
        break;
    }
  }

  private checkSync(): void {
    if (!this._synced && this.connectionState === "connected") {
      this._synced = true;
      for (const handler of this._onSyncHandlers) {
        handler();
      }
    }
  }
}
