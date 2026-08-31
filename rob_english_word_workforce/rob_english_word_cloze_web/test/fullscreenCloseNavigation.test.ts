import { describe, expect, it } from "vitest";
import appSource from "../src/App.tsx?raw";
import launcherSource from "../src/components/PracticeLaunchers.tsx?raw";
import wrongCollectionSource from "../src/components/WrongSentenceCollection.tsx?raw";

describe("fullscreen close navigation contract", () => {
  it("removes visible return actions from solo training and its child pages", () => {
    expect(launcherSource).not.toMatch(/>\s*返回\s*</);
    expect(appSource).not.toMatch(/>\s*返回\s*</);
  });

  it("uses explicit close actions for each fullscreen layer", () => {
    expect(launcherSource).toContain('label="关闭单独训练"');
    expect(wrongCollectionSource).toContain('label="关闭错题集"');

    for (const label of ["关闭难度选择", "关闭句子列表", "关闭答题结果", "退出练习"]) {
      expect(appSource).toContain(`label="${label}"`);
    }
  });

  it("closes each child layer without leaving solo training", () => {
    expect(appSource).toContain('onClose={() => setShowDifficultyPicker(false)}');
    expect(appSource).toContain('onClose={() => setShowPendingTasks(false)}');
    expect(appSource).toContain('onClose={() => setShowAnswerResults(false)}');
    expect(appSource).toContain('onClose={() => setShowWrongSentences(false)}');
    expect(appSource).toContain('onClose={closePractice}');
  });

  it("hides the underlying solo close action while a child layer is visible", () => {
    expect(appSource).toContain(
      "showClose={!showDifficultyPicker && !showPendingTasks && !showAnswerResults && !showWrongSentences && !isPracticeActive}",
    );
  });

  it("closes only the topmost visible layer when Escape is pressed", () => {
    const escapeBlock = appSource.match(/if \(event\.key === "Escape"\) \{([\s\S]*?)\n      \}/)?.[1] || "";
    const closeOrder = [
      "isPracticeActive",
      "showAnswerResults",
      "showPendingTasks",
      "showDifficultyPicker",
      "showWrongSentences",
      'launcherMode === "solo"',
    ].map((marker) => escapeBlock.indexOf(marker));

    expect(closeOrder.every((index) => index >= 0)).toBe(true);
    expect(closeOrder).toEqual([...closeOrder].sort((left, right) => left - right));
  });
});
