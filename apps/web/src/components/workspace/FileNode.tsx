import { memo } from "react";
import { File, Folder, FileCode, FileJson, FileText, Image } from "lucide-react";
import type { FileNode as FileNodeType } from "../../types/workspace";

interface FileNodeProps {
  readonly node: FileNodeType;
  readonly depth: number;
  readonly isExpanded: boolean;
  readonly isSelected: boolean;
  readonly onToggle: (path: string) => void;
  readonly onSelect: (path: string) => void;
}

function getIcon(name: string, type: "file" | "folder"): React.ReactNode {
  if (type === "folder") return <Folder className="h-4 w-4 text-amber-400" />;
  const ext = name.split(".").pop()?.toLowerCase() ?? "";
  switch (ext) {
    case "ts":
    case "tsx":
    case "js":
    case "jsx":
      return <FileCode className="h-4 w-4 text-blue-400" />;
    case "json":
      return <FileJson className="h-4 w-4 text-yellow-400" />;
    case "md":
      return <FileText className="h-4 w-4 text-slate-400" />;
    case "png":
    case "jpg":
    case "svg":
      return <Image className="h-4 w-4 text-green-400" />;
    default:
      return <File className="h-4 w-4 text-slate-400" />;
  }
}

export const FileNodeRow = memo(function FileNodeRow({
  node,
  depth,
  isExpanded,
  isSelected,
  onToggle,
  onSelect,
}: FileNodeProps): React.ReactNode {
  const paddingLeft = `${String(depth * 12 + 8)}px`;

  if (node.type === "folder") {
    return (
      <div>
        <button
          type="button"
          onClick={() => { onToggle(node.path); }}
          className={`flex w-full items-center gap-2 px-2 py-1 text-left text-sm transition-colors ${
            isSelected
              ? "bg-forge-800/50 text-white"
              : "text-slate-400 hover:bg-slate-800 hover:text-slate-200"
          }`}
          style={{ paddingLeft }}
        >
          <span className="text-[10px] text-slate-600">
            {isExpanded ? "▾" : "▸"}
          </span>
          {getIcon(node.name, "folder")}
          <span className="truncate">{node.name}</span>
        </button>
      </div>
    );
  }

  return (
    <button
      type="button"
      onClick={() => { onSelect(node.path); }}
      className={`flex w-full items-center gap-2 px-2 py-1 text-left text-sm transition-colors ${
        isSelected
          ? "bg-forge-800/50 text-white"
          : "text-slate-400 hover:bg-slate-800 hover:text-slate-200"
      }`}
      style={{ paddingLeft }}
    >
      {getIcon(node.name, "file")}
      <span className="truncate">{node.name}</span>
    </button>
  );
});
