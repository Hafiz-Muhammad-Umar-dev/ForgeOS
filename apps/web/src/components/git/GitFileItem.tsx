import type { GitFile, GitFileStatus } from "../../types/git";

interface GitFileItemProps {
  readonly file: GitFile;
  readonly onStage?: (path: string) => void;
  readonly onUnstage?: (path: string) => void;
  readonly onDiscard?: (path: string) => void;
  readonly onOpen?: (path: string) => void;
}

const statusLabels: Record<GitFileStatus, string> = {
  added: "A",
  modified: "M",
  deleted: "D",
  renamed: "R",
  copied: "C",
  unmerged: "U",
};

const statusColors: Record<GitFileStatus, string> = {
  added: "text-green-400",
  modified: "text-yellow-400",
  deleted: "text-red-400",
  renamed: "text-blue-400",
  copied: "text-cyan-400",
  unmerged: "text-purple-400",
};

export function GitFileItem({ file, onStage, onUnstage, onDiscard }: GitFileItemProps): React.ReactNode {
  return (
    <div className="group flex items-center gap-2 px-3 py-1 text-sm hover:bg-slate-800/30">
      <span className={`w-5 text-center text-[10px] font-bold ${statusColors[file.status]}`}>
        {statusLabels[file.status]}
      </span>
      <span className="flex-1 truncate text-slate-300">{file.path}</span>
      <div className="flex gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
        {file.staged
          ? onUnstage !== undefined && (
              <button type="button" onClick={() => { onUnstage(file.path); }} className="rounded px-1 text-[10px] text-slate-500 hover:bg-slate-700 hover:text-slate-200" title="Unstage">
                -
              </button>
            )
          : onStage !== undefined && (
              <button type="button" onClick={() => { onStage(file.path); }} className="rounded px-1 text-[10px] text-slate-500 hover:bg-slate-700 hover:text-slate-200" title="Stage">
                +
              </button>
            )}
        {onDiscard !== undefined && (
          <button type="button" onClick={() => { onDiscard(file.path); }} className="rounded px-1 text-[10px] text-red-500 hover:bg-red-950/30" title="Discard">
            ×
          </button>
        )}
      </div>
    </div>
  );
}
