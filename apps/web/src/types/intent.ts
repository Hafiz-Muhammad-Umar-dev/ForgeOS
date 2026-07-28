export interface IntentView {
  readonly id: string;
  readonly user_id?: string;
  readonly org_id: string;
  readonly project_id?: string;
  readonly trace_id?: string;
  readonly text?: string;
  readonly status: string;
  readonly summary?: string;
  readonly error?: string;
  readonly created_at: string;
  readonly updated_at: string;
}
