import { useCallback, useEffect, useRef } from "react";
import Editor, { type OnMount } from "@monaco-editor/react";
import type { editor } from "monaco-editor";
import type { EditorTab } from "../../types/workspace";
import { EditorTabs } from "./EditorTabs";

interface EditorViewProps {
  readonly tabs: EditorTab[];
  readonly activeTabId: string | null;
  readonly onSelectTab: (id: string) => void;
  readonly onCloseTab: (id: string) => void;
  readonly onContentChange: (id: string, content: string) => void;
  readonly onSave: (id: string) => void;
}

export function EditorView({
  tabs,
  activeTabId,
  onSelectTab,
  onCloseTab,
  onContentChange,
  onSave,
}: EditorViewProps): React.ReactNode {
  const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null);
  const activeTab = tabs.find((t) => t.id === activeTabId) ?? null;

  const handleEditorMount: OnMount = useCallback(
    (editorInstance) => {
      editorRef.current = editorInstance;

      // Ctrl+S / Cmd+S
      editorInstance.addCommand(
        /* monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS */ 2048 | 49,
        () => {
          if (activeTabId !== null) {
            onSave(activeTabId);
          }
        },
      );
    },
    [activeTabId, onSave],
  );

  // Update editor content when active tab changes.
  useEffect(() => {
    const editor = editorRef.current;
    if (editor === null || activeTab === null) return;
    const model = editor.getModel();
    if (model !== null && model.getValue() !== activeTab.content) {
      model.setValue(activeTab.content);
    }
  }, [activeTab?.id]); // eslint-disable-line react-hooks/exhaustive-deps

  const handleChange = useCallback(
    (value: string | undefined) => {
      if (activeTabId !== null && value !== undefined) {
        onContentChange(activeTabId, value);
      }
    },
    [activeTabId, onContentChange],
  );

  if (tabs.length === 0) {
    return (
      <div className="flex h-full flex-col">
        <div className="flex items-center justify-center flex-1">
          <p className="text-sm text-slate-500">
            Select a file from the explorer to start editing
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      <EditorTabs
        tabs={tabs}
        activeTabId={activeTabId}
        onSelect={onSelectTab}
        onClose={onCloseTab}
      />

      <div className="flex-1">
        {activeTab !== null && (
          <Editor
            key={activeTab.id}
            height="100%"
            language={activeTab.language}
            theme="vs-dark"
            value={activeTab.content}
            onChange={handleChange}
            options={{
              minimap: { enabled: true },
              scrollBeyondLastLine: false,
              fontSize: 14,
              wordWrap: "on",
              automaticLayout: true,
              lineNumbers: "on",
            }}
            onMount={handleEditorMount}
          />
        )}
      </div>
    </div>
  );
}
