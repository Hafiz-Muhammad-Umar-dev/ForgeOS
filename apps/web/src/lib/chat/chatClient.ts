import { getAuthToken } from "../api";
import { StreamParser } from "./streamParser";
import type { StreamChunk } from "../../types/chat";

const BASE_URL: string =
  (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "/v1";

export interface SendMessageOptions {
  readonly message: string;
  readonly conversationId?: string;
  readonly signal?: AbortSignal;
  readonly onChunk?: (chunk: StreamChunk) => void;
  readonly onError?: (error: Error) => void;
  readonly onDone?: () => void;
}

export async function sendMessage(options: SendMessageOptions): Promise<string> {
  const { message, conversationId, signal, onChunk, onError, onDone } = options;

  const token = getAuthToken();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (token !== null) {
    headers["authorization"] = `Bearer ${token}`;
  }

  const body: Record<string, unknown> = { message };
  if (conversationId !== undefined) {
    body.conversation_id = conversationId;
  }

  try {
    const response = await fetch(`${BASE_URL}/chat/completions`, {
      method: "POST",
      headers,
      body: JSON.stringify(body),
      signal,
    });

    if (!response.ok) {
      const errorText = await response.text().catch(() => "Unknown error");
      throw new Error(`Chat API error ${String(response.status)}: ${errorText}`);
    }

    const contentType = response.headers.get("content-type") ?? "";

    if (contentType.includes("text/event-stream")) {
      return await handleStreamResponse(response, onChunk, onDone, signal);
    }
    const data = (await response.json()) as { content?: string };
    const content = data.content ?? "";
    onChunk?.({ content, done: true });
    onDone?.();
    return content;
  } catch (err: unknown) {
    if (err instanceof DOMException && err.name === "AbortError") {
      onDone?.();
      return "";
    }
    const error = err instanceof Error ? err : new Error(String(err));
    onError?.(error);
    throw error;
  }
}

async function handleStreamResponse(
  response: Response,
  onChunk?: (chunk: StreamChunk) => void,
  onDone?: () => void,
  signal?: AbortSignal,
): Promise<string> {
  const reader = response.body?.getReader();
  if (reader === undefined) {
    throw new Error("Response body is not readable");
  }

  const decoder = new TextDecoder();
  const parser = new StreamParser();
  let fullContent = "";

  try {
    for (;;) {
      if (signal?.aborted === true) break;

      const { done, value } = await reader.read();
      if (done) break;

      const text = decoder.decode(value, { stream: true });
      const chunks = parser.append(text);

      for (const chunk of chunks) {
        if (chunk.error !== undefined) {
          throw new Error(chunk.error);
        }
        fullContent += chunk.content;
        onChunk?.(chunk);
      }
    }
  } finally {
    reader.releaseLock();
  }

  onChunk?.({ content: "", done: true });
  onDone?.();
  return fullContent;
}
