interface Problem {
  readonly file: string;
  readonly line: number;
  readonly column: number;
  readonly message: string;
  readonly severity: "error" | "warning" | "info";
}

const severityColors: Record<string, string> = {
  error: "text-red-400",
  warning: "text-yellow-400",
  info: "text-blue-400",
};

interface ProblemsPanelProps {
  readonly problems?: Problem[];
}

export function ProblemsPanel({ problems }: ProblemsPanelProps): React.ReactNode {
  const items = problems ?? [];

  return (
    <div className="flex h-full flex-col">
      <div className="border-b border-slate-800 px-3 py-1.5">
        <span className="text-[10px] font-semibold uppercase tracking-widest text-slate-500">
          Problems
        </span>
      </div>
      <div className="flex-1 overflow-y-auto">
        {items.length === 0 && (
          <div className="p-3 text-center">
            <p className="text-xs text-slate-500">No problems detected.</p>
          </div>
        )}
        {items.map((problem, i) => (
          <div
            key={i}
            className="flex items-start gap-2 border-b border-slate-800/50 px-3 py-2"
          >
            <span
              className={`mt-0.5 shrink-0 text-xs font-bold uppercase ${severityColors[problem.severity]}`}
            >
              {problem.severity}
            </span>
            <div className="min-w-0 flex-1">
              <p className="truncate text-xs text-slate-300">{problem.message}</p>
              <p className="text-[10px] text-slate-500">
                {problem.file}:{String(problem.line)}:{String(problem.column)}
              </p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
