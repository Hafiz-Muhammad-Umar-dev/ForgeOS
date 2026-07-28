import { memo } from "react";
import type { WorkspaceFolder } from "../../types/workspace";
import { FileNodeRow } from "./FileNode";

interface FileTreeProps {
  readonly folder: WorkspaceFolder;
  readonly depth: number;
  readonly expandedFolders: Set<string>;
  readonly selectedFile: string | null;
  readonly onToggle: (path: string) => void;
  readonly onSelect: (path: string) => void;
}

export const FileTree = memo(function FileTree({
  folder,
  depth,
  expandedFolders,
  selectedFile,
  onToggle,
  onSelect,
}: FileTreeProps): React.ReactNode {
  const isExpanded = expandedFolders.has(folder.path);

  return (
    <div>
      <FileNodeRow
        node={folder}
        depth={depth}
        isExpanded={isExpanded}
        isSelected={selectedFile === folder.path}
        onToggle={onToggle}
        onSelect={onSelect}
      />
      {isExpanded && (
        <div>
          {folder.children.map((child) => {
            if (child.type === "folder") {
              return (
                <FileTree
                  key={child.path}
                  folder={child}
                  depth={depth + 1}
                  expandedFolders={expandedFolders}
                  selectedFile={selectedFile}
                  onToggle={onToggle}
                  onSelect={onSelect}
                />
              );
            }
            return (
              <FileNodeRow
                key={child.path}
                node={child}
                depth={depth + 1}
                isExpanded={false}
                isSelected={selectedFile === child.path}
                onToggle={onToggle}
                onSelect={onSelect}
              />
            );
          })}
        </div>
      )}
    </div>
  );
});
