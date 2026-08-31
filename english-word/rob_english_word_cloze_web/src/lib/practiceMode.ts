export type LauncherMode = "review" | "solo";
export type PracticeSource = "review" | "solo";

export interface SubmissionIdentity {
  taskId: number;
  actionType: "answer" | "reveal";
  practiceSource: PracticeSource;
  answers: readonly string[];
}

export function nextLauncherModeAfterBatch(source: PracticeSource): LauncherMode {
  return source;
}

export function shouldReuseSubmission(
  pending: SubmissionIdentity | null,
  candidate: SubmissionIdentity,
): boolean {
  return Boolean(
    pending
      && pending.taskId === candidate.taskId
      && pending.actionType === candidate.actionType
      && pending.practiceSource === candidate.practiceSource
      && pending.answers.length === candidate.answers.length
      && pending.answers.every((answer, index) => answer === candidate.answers[index]),
  );
}
