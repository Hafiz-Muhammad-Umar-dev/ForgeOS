interface DeploymentToolbarProps {
  readonly onRefresh: () => void;
}

export function DeploymentToolbar({ onRefresh }: DeploymentToolbarProps): React.ReactNode {
  return (
    <div className="flex items-center justify-between border-b border-slate-800 px-4 py-2">
      <h2 className="text-sm font-semibold text-white">Deployments</h2>
      <button
        type="button"
        onClick={onRefresh}
        className="rounded-lg px-2.5 py-1 text-xs font-medium text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200"
      >
        ↻ Refresh
      </button>
    </div>
  );
}
