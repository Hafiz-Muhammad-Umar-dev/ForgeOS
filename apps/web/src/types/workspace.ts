export interface WorkspaceFile {
  readonly name: string;
  readonly path: string;
  readonly type: "file";
  readonly size?: number;
  readonly language?: string;
}

export interface WorkspaceFolder {
  readonly name: string;
  readonly path: string;
  readonly type: "folder";
  readonly children: Array<WorkspaceFile | WorkspaceFolder>;
}

export interface FileContent {
  readonly path: string;
  readonly content: string;
  readonly language?: string;
}

export type FileNode = WorkspaceFile | WorkspaceFolder;

export interface EditorTab {
  readonly id: string;
  readonly path: string;
  readonly name: string;
  readonly language: string;
  readonly isDirty: boolean;
  readonly content: string;
  readonly savedContent: string;
}

export interface AgentEvent {
  readonly id: string;
  readonly type: "planning" | "searching" | "executing" | "reading" | "writing" | "running" | "completed" | "failed";
  readonly description: string;
  readonly tool?: string;
  readonly status: "running" | "completed" | "failed";
  readonly timestamp: number;
  readonly duration?: number;
}
