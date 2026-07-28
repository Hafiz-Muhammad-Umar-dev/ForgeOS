import { type FormEvent, useRef } from "react";

interface MessageInputProps {
  readonly onSend: (content: string) => void;
  readonly onStop: () => void;
  readonly isStreaming: boolean;
  readonly disabled?: boolean;
}

export function MessageInput({
  onSend,
  onStop,
  isStreaming,
  disabled = false,
}: MessageInputProps): React.ReactNode {
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);

  function handleSubmit(e: FormEvent): void {
    e.preventDefault();
    const textarea = textareaRef.current;
    if (textarea === null) return;
    const content = textarea.value.trim();
    if (content.length === 0) return;

    onSend(content);
    textarea.value = "";
    textarea.style.height = "auto";
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>): void {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      const form = (e.target as HTMLTextAreaElement).form;
      if (form !== null) {
        form.requestSubmit();
      }
    }
  }

  function handleInput(): void {
    const textarea = textareaRef.current;
    if (textarea === null) return;
    textarea.style.height = "auto";
    const targetH = Math.min(textarea.scrollHeight, 200);
    textarea.style.height = `${String(targetH)}px`;
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="flex items-end gap-3 border-t border-slate-800 bg-slate-900/80 p-4"
    >
      <div className="relative flex-1">
        <textarea
          ref={textareaRef}
          onInput={handleInput}
          onKeyDown={handleKeyDown}
          placeholder="Send a message... (Shift+Enter for new line)"
          disabled={disabled}
          rows={1}
          className="max-h-[200px] w-full resize-none rounded-lg border border-slate-700 bg-slate-800 px-3 py-2.5 text-sm text-white placeholder-slate-500 transition-colors focus:border-forge-500 focus:outline-none focus:ring-1 focus:ring-forge-500 disabled:opacity-50"
        />
      </div>

      {isStreaming ? (
        <button
          type="button"
          onClick={onStop}
          className="rounded-lg bg-red-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-red-500"
        >
          Stop
        </button>
      ) : (
        <button
          type="submit"
          disabled={disabled}
          className="rounded-lg bg-forge-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-forge-500 disabled:opacity-50"
        >
          Send
        </button>
      )}
    </form>
  );
}
