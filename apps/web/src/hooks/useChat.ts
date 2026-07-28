import { useCallback, useRef, useState } from "react";
import { sendMessage } from "../lib/chat/chatClient";
import type { ChatMessage } from "../types/chat";

interface UseChatOptions {
  readonly conversationId?: string;
  readonly system?: string;
}

interface UseChatResult {
  readonly messages: ChatMessage[];
  readonly isStreaming: boolean;
  readonly error: string | null;
  readonly send: (content: string) => void;
  readonly stop: () => void;
  readonly clear: () => void;
  readonly retry: () => void;
}

let _msgCounter = 0;

function createId(): string {
  _msgCounter++;
  return `msg-${String(_msgCounter)}-${Date.now().toString(36)}`;
}

export function useChat(options?: UseChatOptions): UseChatResult {
  const { conversationId } = options ?? {};
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);
  const lastMessageRef = useRef<string>("");

  const send = useCallback(
    (content: string) => {
      if (content.trim().length === 0 || isStreaming) return;

      setError(null);
      setIsStreaming(true);
      lastMessageRef.current = content;

      const userMessage: ChatMessage = {
        id: createId(),
        role: "user",
        content,
        createdAt: Date.now(),
      };

      const assistantMessage: ChatMessage = {
        id: createId(),
        role: "assistant",
        content: "",
        createdAt: Date.now(),
      };

      setMessages((prev) => [...prev, userMessage, assistantMessage]);

      const controller = new AbortController();
      abortRef.current = controller;

      sendMessage({
        message: content,
        conversationId,
        signal: controller.signal,
        onChunk: (chunk) => {
          if (chunk.done) return;
          setMessages((prev) => {
            const updated = [...prev];
            const last = updated[updated.length - 1];
            if (last.role === "assistant") {
              updated[updated.length - 1] = {
                ...last,
                content: last.content + chunk.content,
              };
            }
            return updated;
          });
        },
        onError: (err) => {
          setError(err.message);
          setIsStreaming(false);
          setMessages((prev) => {
            const updated = [...prev];
            const last = updated[updated.length - 1];
            if (last.role === "assistant") {
              updated[updated.length - 1] = {
                ...last,
                role: "error",
                content: err.message,
              };
            }
            return updated;
          });
        },
        onDone: () => {
          setIsStreaming(false);
        },
      }).catch(() => {
        setIsStreaming(false);
      });
    },
    [conversationId, isStreaming],
  );

  const stop = useCallback(() => {
    if (abortRef.current !== null) {
      abortRef.current.abort();
      abortRef.current = null;
    }
    setIsStreaming(false);
  }, []);

  const clear = useCallback(() => {
    setMessages([]);
    setError(null);
    setIsStreaming(false);
  }, []);

  const retry = useCallback(() => {
    const lastContent = lastMessageRef.current;
    if (lastContent.length === 0) return;
    setMessages((prev) => prev.slice(0, -2));
    send(lastContent);
  }, [send]);

  return { messages, isStreaming, error, send, stop, clear, retry };
}
