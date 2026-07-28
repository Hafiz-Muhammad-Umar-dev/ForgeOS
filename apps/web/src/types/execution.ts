import type { AgentRole } from "./agent";

export type TaskStatus = "pending" | "running" | "completed" | "failed" | "skipped";

export interface ExecutionNode {
  readonly id: string;
  readonly agentRole: AgentRole;
  readonly label: string;
  readonly status: TaskStatus;
  readonly progress: number;
  readonly runtime: number;
  readonly tokens: number;
  readonly cost: number;
  readonly parentId?: string;
}

export interface ExecutionEdge {
  readonly id: string;
  readonly source: string;
  readonly target: string;
  readonly animated?: boolean;
}

export interface ExecutionPlan {
  readonly id: string;
  readonly intentId: string;
  readonly status: "pending" | "running" | "completed" | "failed";
  readonly nodes: ExecutionNode[];
  readonly edges: ExecutionEdge[];
  readonly createdAt: number;
  readonly startedAt?: number;
  readonly completedAt?: number;
}

export interface ExecutionEvent {
  readonly id: string;
  readonly type: "tool_started" | "tool_completed" | "tool_failed" | "reasoning" | "memory" | "cost_update";
  readonly agentId: string;
  readonly content: string;
  readonly timestamp: number;
  readonly metadata?: Record<string, unknown>;
}

export interface ExecutionMetrics {
  readonly totalTokens: number;
  readonly promptTokens: number;
  readonly completionTokens: number;
  readonly estimatedCost: number;
  readonly executionDuration: number;
  readonly averageLatency: number;
  readonly toolsExecuted: number;
}
