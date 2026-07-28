interface DeploymentActionsProps {
  readonly onStart?: () => void;
  readonly onStop?: () => void;
  readonly onRestart?: () => void;
  readonly onPause?: () => void;
  readonly onResume?: () => void;
  readonly onDelete?: () => void;
  readonly isRunning?: boolean;
}

export function DeploymentActions({ onStart, onStop, onRestart, onPause, onResume, onDelete, isRunning }: DeploymentActionsProps): React.ReactNode {
  const btn = "rounded-lg border border-slate-700 px-3 py-1.5 text-xs font-medium text-slate-300 transition-colors hover:bg-slate-800 hover:text-white";

  return (
    <div className="flex flex-wrap gap-2">
      {isRunning ? (
        <>
          {onStop !== undefined && <button type="button" onClick={onStop} className={btn}>Stop</button>}
          {onPause !== undefined && <button type="button" onClick={onPause} className={btn}>Pause</button>}
          {onRestart !== undefined && <button type="button" onClick={onRestart} className={btn}>Restart</button>}
        </>
      ) : (
        <>
          {onStart !== undefined && <button type="button" onClick={onStart} className={btn}>Start</button>}
          {onResume !== undefined && <button type="button" onClick={onResume} className={btn}>Resume</button>}
        </>
      )}
      {onDelete !== undefined && (
        <button type="button" onClick={onDelete} className="rounded-lg border border-red-900/50 px-3 py-1.5 text-xs font-medium text-red-400 transition-colors hover:bg-red-950/30">
          Delete
        </button>
      )}
    </div>
  );
}
