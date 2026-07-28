interface GitToolbarProps {
  readonly onCommit: () => void;
  readonly onPush: () => void;
  readonly onPull: () => void;
  readonly onFetch: () => void;
  readonly onRefresh: () => void;
  readonly isDisabled?: boolean;
}

export function GitToolbar({
  onCommit,
  onPush,
  onPull,
  onFetch,
  onRefresh,
  isDisabled = false,
}: GitToolbarProps): React.ReactNode {
  const btnClass =
    "rounded px-2 py-1 text-xs font-medium text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200 disabled:opacity-50";

  return (
    <div className="flex items-center gap-1 border-b border-slate-800 px-2 py-1.5">
      <button type="button" onClick={onCommit} disabled={isDisabled} className={btnClass}>
        Commit
      </button>
      <button type="button" onClick={onPush} disabled={isDisabled} className={btnClass}>
        Push
      </button>
      <button type="button" onClick={onPull} disabled={isDisabled} className={btnClass}>
        Pull
      </button>
      <button type="button" onClick={onFetch} disabled={isDisabled} className={btnClass}>
        Fetch
      </button>
      <div className="flex-1" />
      <button type="button" onClick={onRefresh} disabled={isDisabled} className={btnClass}>
        ↻
      </button>
    </div>
  );
}
