import { useWorkspace } from "../../hooks/useWorkspace";
import { FileTree } from "./FileTree";
import { readFile } from "../../lib/workspace/workspaceClient";

interface WorkspaceExplorerProps {
  readonly onOpenFile: (path: string, name: string, content: string) => void;
}

export function WorkspaceExplorer({ onOpenFile }: WorkspaceExplorerProps): React.ReactNode {
  const {
    root,
    isLoading,
    error,
    expandedFolders,
    toggleFolder,
    selectedFile,
    selectFile,
    refresh,
  } = useWorkspace();

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b border-slate-800 px-3 py-2">
        <span className="text-[10px] font-semibold uppercase tracking-widest text-slate-500">
          Explorer
        </span>
        <button
          type="button"
          onClick={refresh}
          className="rounded px-1.5 py-0.5 text-xs text-slate-500 hover:bg-slate-800 hover:text-slate-300"
        >
          ↻
        </button>
      </div>

      <div className="flex-1 overflow-y-auto py-1">
        {isLoading && (
          <div className="px-3 py-4 text-center">
            <p className="text-xs text-slate-500">Loading workspace...</p>
          </div>
        )}

        {error !== null && (
          <div className="px-3 py-4 text-center">
            <p className="text-xs text-red-400">{error}</p>
            <button
              type="button"
              onClick={refresh}
              className="mt-2 rounded bg-slate-800 px-2 py-1 text-xs text-slate-400"
            >
              Retry
            </button>
          </div>
        )}

        {!isLoading && error === null && root !== null && (
          <FileTree
            folder={root}
            depth={0}
            expandedFolders={expandedFolders}
            selectedFile={selectedFile}
            onToggle={toggleFolder}
            onSelect={(path) => {
              selectFile(path);
              readFile(path)
                .then((file) => {
                  onOpenFile(path, path.split("/").pop() ?? path, file.content);
                })
                .catch(() => undefined);
            }}
          />
        )}
      </div>
    </div>
  );
}
