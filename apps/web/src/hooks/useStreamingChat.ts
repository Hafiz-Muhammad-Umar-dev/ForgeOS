import { useCallback, useRef, useState } from "react";
import type { StreamChunk } from "../types/chat";

interface UseStreamingChatOptions {
  readonly url: string;
  readonly onChunk?: (chunk: StreamChunk) => void;
  readonly onDone?: () => void;
  readonly onError?: (error: Error) => void;
}

interface UseStreamingChatResult {
  readonly isStreaming: boolean;
  readonly connect: (body: Record<string, unknown>) => void;
  readonly stop: () => void;
}

export function useStreamingChat(
  options: UseStreamingChatOptions,
): UseStreamingChatResult {
  const { url, onChunk, onDone, onError } = options;
  const [isStreaming, setIsStreaming] = useState(false);
  const abortRef = useRef<AbortController | null>(null);

  const connect = useCallback(
    (body: Record<string, unknown>) => {
      if (isStreaming) return;

      setIsStreaming(true);
      const controller = new AbortController();
      abortRef.current = controller;

      const headers: Record<string, string> = {
        "Content-Type": "application/json",
      };

      void runStream(url, headers, body, controller, onChunk, onDone, onError)
        .catch(() => undefined)
        .finally(() => {
          setIsStreaming(false);
        });
    },
    [url, isStreaming, onChunk, onDone, onError],
  );

  const stop = useCallback(() => {
    if (abortRef.current !== null) {
      abortRef.current.abort();
      abortRef.current = null;
    }
    setIsStreaming(false);
  }, []);

  return { isStreaming, connect, stop };
}

async function runStream(
  url: string,
  headers: Record<string, string>,
  body: Record<string, unknown>,
  controller: AbortController,
  onChunk?: (chunk: StreamChunk) => void,
  onDone?: () => void,
  onError?: (error: Error) => void,
): Promise<void> {
  try {
    const response = await fetch(url, {
      method: "POST",
      headers,
      body: JSON.stringify(body),
      signal: controller.signal,
    });

    if (!response.ok) {
      throw new Error(`HTTP ${String(response.status)}`);
    }

    const reader = response.body?.getReader();
    if (reader === undefined) {
      throw new Error("Response body is not readable");
    }

    const decoder = new TextDecoder();
    let buffer = "";

    let isDone = false;
    while (!isDone) {
      if (controller.signal.aborted) break;

      const result = await reader.read();
      isDone = result.done;
      if (result.done) break;

      buffer += decoder.decode(result.value, { stream: true });
      const lines = buffer.split("\n");
      buffer = lines.pop() ?? "";

      for (const line of lines) {
        if (line.startsWith("data: ")) {
          const payload = line.slice(6).trim();
          if (payload === "[DONE]") {
            onDone?.();
          } else {
            try {
              const parsed = JSON.parse(payload) as { content?: string };
              onChunk?.({ content: parsed.content ?? "", done: false });
            } catch {
              onChunk?.({ content: payload, done: false });
            }
          }
        }
      }
    }
  } catch (err: unknown) {
    if (err instanceof DOMException && err.name === "AbortError") return;
    const error = err instanceof Error ? err : new Error(String(err));
    onError?.(error);
  } finally {
    onDone?.();
  }
}
