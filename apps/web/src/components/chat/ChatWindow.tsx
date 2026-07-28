import { useEffect, useRef } from "react";
import { useChat } from "../../hooks/useChat";
import { MessageBubble } from "./MessageBubble";
import { MessageInput } from "./MessageInput";
import { TypingIndicator } from "./TypingIndicator";

interface ChatWindowProps {
  readonly conversationId?: string;
}

export function ChatWindow({ conversationId }: ChatWindowProps): React.ReactNode {
  const { messages, isStreaming, error, send, stop, clear, retry } = useChat({
    conversationId,
  });

  const scrollRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (scrollRef.current !== null) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages]);

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-slate-800 px-4 py-3">
        <h2 className="text-sm font-semibold text-white">AI Chat</h2>
        <div className="flex items-center gap-2">
          {messages.length > 0 && (
            <button
              type="button"
              onClick={clear}
              className="rounded px-2 py-1 text-xs text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200"
            >
              Clear
            </button>
          )}
          {error !== null && (
            <button
              type="button"
              onClick={retry}
              className="rounded px-2 py-1 text-xs text-red-400 transition-colors hover:bg-red-950/30 hover:text-red-300"
            >
              Retry
            </button>
          )}
        </div>
      </div>

      {/* Messages */}
      <div ref={scrollRef} className="flex-1 space-y-4 overflow-y-auto p-4">
        {messages.length === 0 && !isStreaming && (
          <div className="flex h-full items-center justify-center">
            <div className="text-center">
              <p className="text-sm text-slate-500">Start a conversation with the AI.</p>
              <p className="mt-1 text-xs text-slate-600">
                Ask a question or describe what you want to build.
              </p>
            </div>
          </div>
        )}

        {messages.map((msg) => (
          <MessageBubble key={msg.id} message={msg} />
        ))}

        {isStreaming && messages[messages.length - 1]?.content.length === 0 && (
          <TypingIndicator />
        )}
      </div>

      {/* Input */}
      <MessageInput
        onSend={send}
        onStop={stop}
        isStreaming={isStreaming}
      />
    </div>
  );
}
