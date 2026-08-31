import { describe, expect, it } from "vitest";
import { nextLauncherModeAfterBatch, shouldReuseSubmission } from "../src/lib/practiceMode";

describe("practice source flow", () => {
  it("returns review batches to the review home", () => {
    expect(nextLauncherModeAfterBatch("review")).toBe("review");
  });

  it("keeps solo batches in solo training", () => {
    expect(nextLauncherModeAfterBatch("solo")).toBe("solo");
  });
});

describe("submission retry identity", () => {
  it("reuses a key only for the exact same payload", () => {
    const pending = {
      taskId: 10,
      actionType: "answer" as const,
      practiceSource: "review" as const,
      answers: ["first", "answer"],
    };

    expect(shouldReuseSubmission(pending, { ...pending, answers: ["first", "answer"] })).toBe(true);
    expect(shouldReuseSubmission(pending, { ...pending, answers: ["changed", "answer"] })).toBe(false);
    expect(shouldReuseSubmission(pending, { ...pending, practiceSource: "solo" })).toBe(false);
  });
});
