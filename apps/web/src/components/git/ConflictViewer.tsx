import type { MergeConflict } from "../../types/git";

interface ConflictViewerProps {
  readonly conflicts: MergeConflict[];
}

export function ConflictViewer({ conflicts }: ConflictViewerProps): React.ReactNode {
  if (conflicts.length === 0) {
    return (
      <div className="p-3 text-center">
        <p className="text-xs text-slate-500">No merge conflicts.</p>
      </div>
    );
  }

  return (
    <div className="space-y-2 p-3">
      <p className="text-xs font-semibold text-slate-400">
        {String(conflicts.length)} merge conflict(s)
      </p>
      {conflicts.map((c) => (
        <div
          key={c.file}
          className="flex items-center justify-between rounded border border-slate-700 px-3 py-2"
        >
          <span className="text-xs text-slate-300">{c.file}</span>
          <span
            className={`text-[10px] font-medium ${
              c.status === "resolved" ? "text-green-400" : "text-red-400"
            }`}
          >
            {c.status}
          </span>
        </div>
      ))}
    </div>
  );
}
