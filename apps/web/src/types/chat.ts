export type MessageRole = "user" | "assistant" | "system" | "tool" | "error";

export interface ChatMessage {
  readonly id: string;
  readonly role: MessageRole;
  readonly content: string;
  readonly createdAt: number;
  readonly metadata?: Record<string, unknown>;
}

export interface Conversation {
  readonly id: string;
  readonly title: string;
  readonly messages: ChatMessage[];
  readonly createdAt: number;
  readonly updatedAt: number;
  readonly model?: string;
  readonly tokenUsage?: TokenUsage;
}

export interface TokenUsage {
  readonly inputTokens: number;
  readonly outputTokens: number;
}

export interface StreamChunk {
  readonly content: string;
  readonly done: boolean;
  readonly error?: string;
}

export interface ChatRequest {
  readonly message: string;
  readonly conversationId?: string;
  readonly model?: string;
  readonly system?: string;
}
