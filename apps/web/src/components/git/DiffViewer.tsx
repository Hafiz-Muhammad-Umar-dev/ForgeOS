import { useGitDiff } from "../../hooks/useGitDiff";

interface DiffViewerProps {
  readonly commitId?: string;
  readonly filePath?: string;
}

export function DiffViewer({ commitId, filePath }: DiffViewerProps): React.ReactNode {
  const { diffs, isLoading } = useGitDiff(commitId, filePath);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-8">
        <p className="text-xs text-slate-500">Loading diff...</p>
      </div>
    );
  }

  if (diffs.length === 0) {
    return (
      <div className="flex items-center justify-center p-8">
        <p className="text-xs text-slate-500">No changes to display.</p>
      </div>
    );
  }

  return (
    <div className="space-y-4 p-3">
      {diffs.map((diff) => (
        <div key={diff.file} className="rounded-lg border border-slate-800">
          <div className="flex items-center justify-between border-b border-slate-800 bg-slate-900/50 px-3 py-1.5">
            <span className="text-xs font-medium text-slate-300">{diff.file}</span>
            <div className="flex gap-3 text-[10px]">
              <span className="text-green-400">+{String(diff.added)}</span>
              <span className="text-red-400">-{String(diff.removed)}</span>
            </div>
          </div>
          <div className="overflow-x-auto p-2">
            <table className="w-full font-mono text-[11px]">
              <tbody>
                {diff.content.split("\n").map((line, i) => {
                  const type = line.startsWith("+")
                    ? "add"
                    : line.startsWith("-")
                      ? "del"
                      : line.startsWith("@")
                        ? "hunk"
                        : "ctx";
                  const bgClass =
                    type === "add"
                      ? "bg-green-950/30"
                      : type === "del"
                        ? "bg-red-950/30"
                        : type === "hunk"
                          ? "bg-blue-950/30"
                          : "";
                  const textClass =
                    type === "add"
                      ? "text-green-300"
                      : type === "del"
                        ? "text-red-300"
                        : type === "hunk"
                          ? "text-blue-300"
                          : "text-slate-400";
                  return (
                    <tr key={i} className={bgClass}>
                      <td className="select-none px-2 text-right text-slate-600">{String(i + 1)}</td>
                      <td className={`whitespace-pre ${textClass}`}>{line}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      ))}
    </div>
  );
}
