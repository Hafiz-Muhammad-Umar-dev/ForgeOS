import { useCallback, useState } from "react";
import type { EditorTab } from "../types/workspace";

interface UseEditorResult {
  readonly tabs: EditorTab[];
  readonly activeTabId: string | null;
  readonly openFile: (path: string, name: string, content: string, language?: string) => void;
  readonly closeTab: (id: string) => void;
  readonly setActiveTab: (id: string) => void;
  readonly updateContent: (id: string, content: string) => void;
  readonly markSaved: (id: string) => void;
  readonly hasDirtyTabs: boolean;
  readonly closeAll: () => void;
}

let tabCounter = 0;

function detectLanguage(path: string): string {
  const ext = path.split(".").pop()?.toLowerCase() ?? "";
  const langMap: Record<string, string> = {
    ts: "typescript",
    tsx: "typescript",
    js: "javascript",
    jsx: "javascript",
    json: "json",
    md: "markdown",
    css: "css",
    html: "html",
    go: "go",
    py: "python",
    rs: "rust",
    yml: "yaml",
    yaml: "yaml",
    toml: "toml",
    sql: "sql",
    sh: "shell",
    bash: "shell",
  };
  return langMap[ext] ?? "plaintext";
}

export function useEditor(): UseEditorResult {
  const [tabs, setTabs] = useState<EditorTab[]>([]);
  const [activeTabId, setActiveTabId] = useState<string | null>(null);

  const openFile = useCallback(
    (path: string, name: string, content: string, language?: string) => {
      tabCounter++;
      const id = `tab-${String(tabCounter)}`;

      setTabs((prev) => {
        const existing = prev.find((t) => t.path === path);
        if (existing !== undefined) {
          setActiveTabId(existing.id);
          return prev;
        }
        const newTab: EditorTab = {
          id,
          path,
          name,
          language: language ?? detectLanguage(path),
          isDirty: false,
          content,
          savedContent: content,
        };
        setActiveTabId(id);
        return [...prev, newTab];
      });
    },
    [],
  );

  const closeTab = useCallback((id: string) => {
    setTabs((prev) => {
      const idx = prev.findIndex((t) => t.id === id);
      const next = prev.filter((t) => t.id !== id);
      if (next.length === 0) {
        setActiveTabId(null);
      } else if (prev.length > 0 && prev[prev.length - 1]?.id === id) {
        setActiveTabId(next[next.length - 1]?.id ?? null);
      } else if (idx > 0) {
        setActiveTabId(prev[idx - 1]?.id ?? null);
      }
      return next;
    });
  }, []);

  const setActiveTab = useCallback((id: string) => {
    setActiveTabId(id);
  }, []);

  const updateContent = useCallback((id: string, content: string) => {
    setTabs((prev) =>
      prev.map((t) =>
        t.id === id ? { ...t, content, isDirty: content !== t.savedContent } : t,
      ),
    );
  }, []);

  const markSaved = useCallback((id: string) => {
    setTabs((prev) =>
      prev.map((t) =>
        t.id === id ? { ...t, savedContent: t.content, isDirty: false } : t,
      ),
    );
  }, []);

  const closeAll = useCallback(() => {
    setTabs([]);
    setActiveTabId(null);
  }, []);

  const hasDirtyTabs = tabs.some((t) => t.isDirty);

  return {
    tabs,
    activeTabId,
    openFile,
    closeTab,
    setActiveTab,
    updateContent,
    markSaved,
    hasDirtyTabs,
    closeAll,
  };
}
