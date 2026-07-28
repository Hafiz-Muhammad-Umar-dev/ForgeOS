import type { IntentView } from "./intent";
import type { TaskView } from "./task";

export type StreamEventType =
  | "intent.created"
  | "intent.updated"
  | "intent.completed"
  | "intent.failed"
  | "task.created"
  | "task.updated"
  | "task.completed"
  | "task.failed"
  | "node.started"
  | "node.completed"
  | "node.failed"
  | "notification";

export interface StreamEvent {
  readonly id: string;
  readonly type: StreamEventType;
  readonly data: string;
  readonly parsed: Record<string, unknown> | null;
  readonly timestamp: number;
}

export type ConnectionState =
  | "connected"
  | "connecting"
  | "reconnecting"
  | "disconnected"
  | "error";

export interface StreamPayloads {
  readonly "intent.created": IntentView;
  readonly "intent.updated": Partial<IntentView>;
  readonly "intent.completed": { readonly id: string; readonly status: string; readonly summary?: string };
  readonly "intent.failed": { readonly id: string; readonly error: string };
  readonly "task.created": TaskView;
  readonly "task.updated": Partial<TaskView>;
  readonly "task.completed": { readonly id: string; readonly intent_id: string; readonly status: string; readonly summary?: string };
  readonly "task.failed": { readonly id: string; readonly intent_id: string; readonly error: string };
  readonly "node.started": { readonly node_id: string; readonly task_id: string };
  readonly "node.completed": { readonly node_id: string; readonly task_id: string; readonly status: string };
  readonly "node.failed": { readonly node_id: string; readonly task_id: string; readonly error: string };
  readonly "notification": { readonly message: string; readonly severity?: string };
}
