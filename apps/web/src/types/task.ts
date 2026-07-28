export interface TaskView {
  readonly id: string;
  readonly intent_id: string;
  readonly agent_name?: string;
  readonly status: string;
  readonly summary?: string;
  readonly error?: string;
  readonly input_tokens: number;
  readonly output_tokens: number;
  readonly created_at: string;
  readonly updated_at: string;
}
