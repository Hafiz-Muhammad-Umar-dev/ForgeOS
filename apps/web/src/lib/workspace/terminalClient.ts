const WS_BASE: string =
  (import.meta.env.VITE_WS_URL as string | undefined) ??
  `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}/v1`;

export class TerminalClient {
  private ws: WebSocket | null = null;
  private onDataHandler: ((data: string) => void) | null = null;
  private isConnected = false;

  public get connected(): boolean {
    return this.isConnected;
  }

  public connect(sessionId: string, onData: (data: string) => void): void {
    this.onDataHandler = onData;

    try {
      this.ws = new WebSocket(`${WS_BASE}/terminal?session=${encodeURIComponent(sessionId)}`);

      this.ws.onopen = () => {
        this.isConnected = true;
      };

      this.ws.onmessage = (event: MessageEvent) => {
        if (typeof event.data === "string") {
          this.onDataHandler?.(event.data);
        } else if (event.data instanceof Blob) {
          void event.data.text().then((text) => {
            this.onDataHandler?.(text);
          });
        }
      };

      this.ws.onclose = () => {
        this.isConnected = false;
      };

      this.ws.onerror = () => {
        this.isConnected = false;
      };
    } catch {
      this.isConnected = false;
    }
  }

  public write(data: string): void {
    if (this.ws !== null && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(data);
    }
  }

  public resize(cols: number, rows: number): void {
    if (this.ws !== null && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: "resize", cols, rows }));
    }
  }

  public disconnect(): void {
    if (this.ws !== null) {
      this.ws.onopen = null;
      this.ws.onmessage = null;
      this.ws.onclose = null;
      this.ws.onerror = null;
      if (
        this.ws.readyState === WebSocket.OPEN ||
        this.ws.readyState === WebSocket.CONNECTING
      ) {
        this.ws.close();
      }
      this.ws = null;
    }
    this.isConnected = false;
    this.onDataHandler = null;
  }
}
