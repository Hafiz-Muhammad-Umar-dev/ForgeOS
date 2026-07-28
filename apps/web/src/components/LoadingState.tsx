interface LoadingStateProps {
  readonly count?: number;
}

export function LoadingState({ count = 3 }: LoadingStateProps): React.ReactNode {
  return (
    <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
      {Array.from({ length: count }, (_, i) => (
        <div
          key={i}
          className="animate-pulse rounded-xl border border-slate-800 bg-slate-900/50 p-6"
        >
          <div className="mb-3 h-3 w-24 rounded bg-slate-800" />
          <div className="mb-2 h-5 w-16 rounded bg-slate-800" />
          <div className="h-3 w-40 rounded bg-slate-800" />
        </div>
      ))}
    </div>
  );
}
