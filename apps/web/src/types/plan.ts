export interface PlanView {
  readonly id: string;
  readonly intent_id: string;
  readonly status: string;
  readonly dag?: unknown;
  readonly summary?: string;
  readonly error?: string;
  readonly created_at: string;
  readonly updated_at: string;
}
