import type { RunStatus } from "./workflow";

export interface ExecutionRunRecord {
  runId: string;
  businessType: string;
  title: string;
  status: RunStatus;
  currentStepId: string;
  durationMs: number;
  error: string;
  startedAt: string;
  finishedAt: string;
  createdAt: string;
}
