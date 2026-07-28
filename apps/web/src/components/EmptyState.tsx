interface EmptyStateProps {
  readonly message?: string;
}

export function EmptyState({
  message = "No data available.",
}: EmptyStateProps): React.ReactNode {
  return (
    <div className="rounded-xl border border-slate-800 bg-slate-900/50 p-8 text-center">
      <p className="text-sm text-slate-400">{message}</p>
    </div>
  );
}
