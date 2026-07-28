export type AgentStatus = "idle" | "running" | "thinking" | "waiting" | "completed" | "failed";

export type AgentRole =
  | "planner"
  | "researcher"
  | "coder"
  | "reviewer"
  | "tester"
  | "deployer";

export interface AgentInfo {
  readonly id: string;
  readonly name: string;
  readonly role: AgentRole;
  readonly status: AgentStatus;
  readonly model: string;
  readonly temperature: number;
  readonly currentTool?: string;
  readonly reasoning?: string;
  readonly memory?: string;
  readonly output?: string;
  readonly queueLength: number;
  readonly executionTime: number;
  readonly tokenUsage: TokenUsage;
  readonly cost: number;
}

export interface TokenUsage {
  readonly prompt: number;
  readonly completion: number;
  readonly total: number;
}
