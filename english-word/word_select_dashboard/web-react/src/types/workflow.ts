export type RunStatus = "pending" | "running" | "success" | "failed" | "skipped" | "retrying";

export type StepStage = "input" | "selection" | "generation" | "review" | "delivery";

export interface WorkflowStep {
  id: string;
  name: string;
  stage: StepStage;
  status: RunStatus;
  description: string;
  startedAt?: string;
  finishedAt?: string;
  durationMs?: number;
  input?: Record<string, unknown>;
  output?: Record<string, unknown>;
  error?: string;
  logs: string[];
}

export interface WorkflowEdge {
  source: string;
  target: string;
}

export interface WorkflowEvent {
  id: string;
  stepId: string;
  level: "info" | "warning" | "error";
  message: string;
  createdAt: string;
}

export interface WorkflowRun {
  id: string;
  businessType: string;
  title: string;
  owner: string;
  status: RunStatus;
  currentStepId: string;
  summary?: string;
  error?: string;
  durationMs?: number;
  startedAt: string;
  finishedAt?: string;
  steps: WorkflowStep[];
  edges: WorkflowEdge[];
  events: WorkflowEvent[];
}
