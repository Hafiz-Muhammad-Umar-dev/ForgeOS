import { memo } from "react";
import type { ChatMessage } from "../../types/chat";
import { MarkdownRenderer } from "./MarkdownRenderer";

interface MessageBubbleProps {
  readonly message: ChatMessage;
}

const roleStyles: Record<string, string> = {
  user: "bg-forge-900/40 border-forge-800/30 ml-12",
  assistant: "bg-slate-800/30 border-slate-700/30 mr-12",
  system: "bg-amber-900/20 border-amber-800/20 mx-12 text-center text-xs",
  tool: "bg-emerald-900/20 border-emerald-800/20 mr-12",
  error: "bg-red-900/20 border-red-800/20 mr-12",
};

const roleLabels: Record<string, string> = {
  user: "You",
  assistant: "AI",
  system: "System",
  tool: "Tool",
  error: "Error",
};

export const MessageBubble = memo(function MessageBubble({
  message,
}: MessageBubbleProps): React.ReactNode {
  const borderStyle = roleStyles[message.role] ?? roleStyles.assistant;

  return (
    <div className={`rounded-xl border p-4 ${borderStyle}`}>
      <div className="mb-1.5 flex items-center gap-2">
        <span className="text-xs font-medium text-slate-400">
          {roleLabels[message.role] ?? message.role}
        </span>
        <span className="text-[10px] text-slate-600">
          {new Date(message.createdAt).toLocaleTimeString()}
        </span>
      </div>
      {message.role === "assistant" || message.role === "user" ? (
        <MarkdownRenderer content={message.content} />
      ) : (
        <p className="whitespace-pre-wrap text-sm text-slate-300">
          {message.content}
        </p>
      )}
    </div>
  );
});
