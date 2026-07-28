import { useCallback, useEffect, useRef, useState } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { TerminalClient } from "../lib/workspace/terminalClient";

interface UseTerminalOptions {
  readonly sessionId?: string;
}

interface UseTerminalResult {
  readonly terminalRef: (element: HTMLDivElement | null) => void;
  readonly isConnected: boolean;
  readonly write: (data: string) => void;
  readonly clear: () => void;
  readonly backendAvailable: boolean;
}

export function useTerminal(options?: UseTerminalOptions): UseTerminalResult {
  const { sessionId = "default" } = options ?? {};
  const [isConnected, setIsConnected] = useState(false);
  const terminalRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const clientRef = useRef<TerminalClient | null>(null);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [backendAvailable] = useState(true);

  const attach = useCallback(
    (element: HTMLDivElement | null) => {
      if (element === null) return;
      if (containerRef.current === element) return;
      containerRef.current = element;

      // Cleanup previous terminal.
      if (terminalRef.current !== null) {
        terminalRef.current.dispose();
      }

      const term = new Terminal({
        cursorBlink: true,
        cursorStyle: "block",
        fontSize: 13,
        fontFamily: "'Cascadia Code', 'Fira Code', 'JetBrains Mono', monospace",
        theme: {
          background: "#0f172a",
          foreground: "#e2e8f0",
          cursor: "#6366f1",
          selectionBackground: "#334155",
          black: "#1e293b",
          red: "#ef4444",
          green: "#22c55e",
          yellow: "#eab308",
          blue: "#6366f1",
          magenta: "#a855f7",
          cyan: "#22d3ee",
          white: "#e2e8f0",
          brightBlack: "#475569",
          brightRed: "#f87171",
          brightGreen: "#4ade80",
          brightYellow: "#facc15",
          brightBlue: "#818cf8",
          brightMagenta: "#c084fc",
          brightCyan: "#67e8f9",
          brightWhite: "#f8fafc",
        },
      });

      const fitAddon = new FitAddon();
      term.loadAddon(fitAddon);
      term.open(element);

      // Fit after opening.
      requestAnimationFrame(() => {
        fitAddon.fit();
      });

      terminalRef.current = term;
      fitAddonRef.current = fitAddon;
    },
    [],
  );

  // Connect terminal client.
  useEffect(() => {
    const client = new TerminalClient();
    clientRef.current = client;

    client.connect(sessionId, (data: string) => {
      terminalRef.current?.write(data);
    });

    setIsConnected(client.connected);

    const interval = setInterval(() => {
      setIsConnected(clientRef.current?.connected ?? false);
    }, 5000);

    return () => {
      clearInterval(interval);
      client.disconnect();
      terminalRef.current?.dispose();
    };
  }, [sessionId]);

  const write = useCallback((data: string) => {
    clientRef.current?.write(data);
  }, []);

  const clear = useCallback(() => {
    terminalRef.current?.clear();
  }, []);

  return {
    terminalRef: attach,
    isConnected,
    write,
    clear,
    backendAvailable,
  };
}
