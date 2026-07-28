import type { GitCommit } from "../../types/git";

interface CommitCardProps {
  readonly commit: GitCommit;
  readonly onSelect?: (oid: string) => void;
}

export function CommitCard({ commit, onSelect }: CommitCardProps): React.ReactNode {
  return (
    <button
      type="button"
      onClick={() => onSelect?.(commit.oid)}
      className="w-full border-b border-slate-800/50 px-3 py-2 text-left transition-colors hover:bg-slate-800/30"
    >
      <p className="truncate text-xs font-medium text-slate-200">{commit.message}</p>
      <div className="mt-1 flex items-center gap-2 text-[10px] text-slate-500">
        <span className="font-mono">{commit.oid.slice(0, 7)}</span>
        <span>{commit.author}</span>
        <span>{new Date(commit.timestamp).toLocaleDateString()}</span>
      </div>
    </button>
  );
}
