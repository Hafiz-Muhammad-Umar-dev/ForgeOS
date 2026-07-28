interface OutputPanelProps {
  readonly outputs?: string[];
}

export function OutputPanel({ outputs }: OutputPanelProps): React.ReactNode {
  const lines = outputs ?? [];

  return (
    <div className="flex h-full flex-col">
      <div className="border-b border-slate-800 px-3 py-1.5">
        <span className="text-[10px] font-semibold uppercase tracking-widest text-slate-500">
          Output
        </span>
      </div>
      <div className="flex-1 overflow-y-auto p-3">
        {lines.length === 0 && (
          <p className="text-xs text-slate-500">No output yet.</p>
        )}
        {lines.map((line, i) => (
          <pre key={i} className="font-mono text-xs text-slate-400">
            {line}
          </pre>
        ))}
      </div>
    </div>
  );
}
