import { useGitHistory } from "../../hooks/useGitHistory";
import { CommitCard } from "./CommitCard";

interface HistoryPanelProps {
  readonly onSelectCommit?: (oid: string) => void;
}

export function HistoryPanel({ onSelectCommit }: HistoryPanelProps): React.ReactNode {
  const { commits, isLoading, refresh } = useGitHistory(50);

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b border-slate-800 px-3 py-1.5">
        <span className="text-[10px] font-semibold uppercase tracking-widest text-slate-500">
          History
        </span>
        <button
          type="button"
          onClick={refresh}
          className="rounded px-1.5 text-xs text-slate-500 hover:bg-slate-800 hover:text-slate-300"
        >
          ↻
        </button>
      </div>
      <div className="flex-1 overflow-y-auto">
        {isLoading && (
          <div className="p-4 text-center">
            <p className="text-xs text-slate-500">Loading history...</p>
          </div>
        )}
        {!isLoading && commits.length === 0 && (
          <div className="p-4 text-center">
            <p className="text-xs text-slate-500">No commits yet.</p>
          </div>
        )}
        {!isLoading && commits.length > 0 && (
          <div>
            {commits.map((c) => (
              <CommitCard key={c.oid} commit={c} onSelect={onSelectCommit} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
