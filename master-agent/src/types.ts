export interface IterationTask {
  id: string;
  description: string;
  files_affected: string[];
  rationale: string;
}

export interface IterationPlan {
  round_number: number;
  title: string;
  rationale: string;
  priority: "low" | "medium" | "high" | "critical";
  tasks: IterationTask[];
  estimated_scope: "small" | "medium" | "large";
  skip_reason?: string;
}

export interface DecideRequest {
  extra_context?: string;
}

export interface DecideResponse {
  plan: IterationPlan;
  model: string;
  thinking_enabled: boolean;
}
