export function TypingIndicator(): React.ReactNode {
  return (
    <div className="mr-12 flex items-center gap-3 rounded-xl border border-slate-700/30 bg-slate-800/30 px-4 py-3">
      <div className="flex gap-1">
        <span className="h-2 w-2 animate-bounce rounded-full bg-slate-500 [animation-delay:0ms]" />
        <span className="h-2 w-2 animate-bounce rounded-full bg-slate-500 [animation-delay:150ms]" />
        <span className="h-2 w-2 animate-bounce rounded-full bg-slate-500 [animation-delay:300ms]" />
      </div>
      <span className="text-xs text-slate-400">AI is thinking...</span>
    </div>
  );
}
