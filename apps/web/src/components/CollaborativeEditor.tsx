import { useCallback, useEffect, useRef } from "react";
import Editor, { type OnMount } from "@monaco-editor/react";
import type { editor } from "monaco-editor";
import * as Y from "yjs";
import { useCollaborativeDocument } from "../hooks/useCollaborativeDocument";
import { useAwareness } from "../hooks/useAwareness";
import { ConnectionBadge } from "./ConnectionBadge";
import { PresenceBar } from "./PresenceBar";

interface CollaborativeEditorProps {
  readonly intentId: string;
  readonly username?: string;
  readonly language?: string;
}

function ytextToString(yt: Y.Text | null): string {
  if (yt === null) return "";
  // yjs Y.Text implements toString() returning the full text content.
  return (yt as unknown as { toString: () => string }).toString();
}

export function CollaborativeEditor({
  intentId,
  username = "anonymous",
  language = "markdown",
}: CollaborativeEditorProps): React.ReactNode {
  const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null);
  const unbindRef = useRef<(() => void) | null>(null);

  const { provider, ytext, connectionState } =
    useCollaborativeDocument({ intentId });

  const users = useAwareness(provider, username);

  // Bind Y.Text to Monaco editor.
  const handleEditorMount: OnMount = useCallback(
    (editorInstance) => {
      editorRef.current = editorInstance;
      const model = editorInstance.getModel();
      if (model === null || ytext === null) return;
      const ytextRef = ytext;

      // Apply initial content from Y.Text.
      const initialContent = ytextToString(ytextRef);
      if (initialContent.length > 0) {
        model.setValue(initialContent);
      }

      // Observe Y.Text changes and apply to Monaco.
      const observer = () => {
        const content = ytextToString(ytextRef);
        if (model.getValue() !== content) {
          const caretPosition = editorInstance.getPosition();
          model.setValue(content);
          if (caretPosition !== null) {
            editorInstance.setPosition(caretPosition);
          }
        }
      };
      ytextRef.observe(observer);

      // Observe Monaco edits and apply to Y.Text.
      const disposable = model.onDidChangeContent(() => {
        const currentValue = model.getValue();
        const content = ytextToString(ytextRef);
        if (currentValue !== content) {
          const doc = ytextRef.doc;
          if (doc !== null) {
            doc.transact(() => {
              ytextRef.delete(0, ytextRef.length);
              ytextRef.insert(0, currentValue);
            }, provider);
          }
        }
      });

      unbindRef.current = () => {
        ytextRef.unobserve(observer);
        disposable.dispose();
      };
    },
    [ytext, provider],
  );

  // Cleanup on unmount.
  useEffect(() => {
    return () => {
      if (unbindRef.current !== null) {
        unbindRef.current();
        unbindRef.current = null;
      }
      editorRef.current = null;
    };
  }, []);

  return (
    <div className="rounded-xl border border-slate-800 bg-slate-900/50">
      <div className="flex items-center justify-between border-b border-slate-800 px-4 py-2">
        <div className="flex items-center gap-4">
          <h3 className="text-sm font-semibold text-white">Collaborative Editor</h3>
          <ConnectionBadge state={connectionState} />
        </div>
        <PresenceBar users={users} />
      </div>

      <Editor
        height="500px"
        language={language}
        theme="vs-dark"
        options={{
          minimap: { enabled: false },
          scrollBeyondLastLine: false,
          fontSize: 14,
          wordWrap: "on",
          automaticLayout: true,
        }}
        onMount={handleEditorMount}
      />
    </div>
  );
}
