import type { GitFile } from "../../types/git";
import { GitFileItem } from "./GitFileItem";

interface GitStatusListProps {
  readonly files: GitFile[];
  readonly onStage?: (path: string) => void;
  readonly onUnstage?: (path: string) => void;
  readonly onDiscard?: (path: string) => void;
}

export function GitStatusList({ files, onStage, onUnstage, onDiscard }: GitStatusListProps): React.ReactNode {
  if (files.length === 0) {
    return (
      <div className="px-3 py-4 text-center">
        <p className="text-xs text-slate-500">No file changes.</p>
      </div>
    );
  }

  const stagedFiles = files.filter((f) => f.staged);
  const unstagedFiles = files.filter((f) => !f.staged && !f.isUntracked);
  const untrackedFiles = files.filter((f) => f.isUntracked);

  return (
    <div className="space-y-3">
      {stagedFiles.length > 0 && (
        <div>
          <SectionHeader title="Staged" count={stagedFiles.length} />
          {stagedFiles.map((f) => (
            <GitFileItem key={f.path} file={f} onUnstage={onUnstage} onDiscard={onDiscard} />
          ))}
        </div>
      )}
      {unstagedFiles.length > 0 && (
        <div>
          <SectionHeader title="Changes" count={unstagedFiles.length} />
          {unstagedFiles.map((f) => (
            <GitFileItem key={f.path} file={f} onStage={onStage} onDiscard={onDiscard} />
          ))}
        </div>
      )}
      {untrackedFiles.length > 0 && (
        <div>
          <SectionHeader title="Untracked" count={untrackedFiles.length} />
          {untrackedFiles.map((f) => (
            <GitFileItem key={f.path} file={f} onStage={onStage} onDiscard={onDiscard} />
          ))}
        </div>
      )}
    </div>
  );
}

function SectionHeader({ title, count }: { readonly title: string; readonly count: number }): React.ReactNode {
  return (
    <div className="flex items-center gap-2 px-3 py-1">
      <span className="text-[10px] font-semibold uppercase tracking-widest text-slate-500">{title}</span>
      <span className="rounded-full bg-slate-800 px-1.5 text-[10px] text-slate-500">{String(count)}</span>
    </div>
  );
}
