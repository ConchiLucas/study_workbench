// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { WrongSentenceCollection } from "../src/components/WrongSentenceCollection";
import { getWrongSentenceDetail, getWrongSentences } from "../src/lib/api";

vi.mock("../src/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../src/lib/api")>();
  return {
    ...actual,
    getWrongSentences: vi.fn(),
    getWrongSentenceDetail: vi.fn(),
  };
});

const item = {
  progressId: 81,
  clozeItemId: 91,
  clozeSentence: "The team lost ____ after the delay.",
  sentence: "The team lost momentum after the delay.",
  translationZh: "延误之后，团队失去了动力。",
  targetWords: ["momentum"],
  wrongBlankIndexes: [0],
  wrongBlankCount: 1,
  practiceContext: "review" as const,
  contentSource: "word-agent",
  difficultyLabel: "大学英语",
  status: "active" as const,
  reviewStage: 0,
  nextReviewTime: "2026-08-02T21:00:00",
  wrongCount: 2,
  firstWrongTime: "2026-08-01T20:00:00",
  lastWrongTime: "2026-08-02T21:00:00",
  lastCostMs: 2100,
};

beforeEach(() => {
  vi.mocked(getWrongSentences).mockResolvedValue({
    items: [item],
    total: 1,
    current: 1,
    pages: 1,
    summary: { activeCount: 1, dueCount: 1, stage1Count: 0, stage2Count: 0, completedCount: 0 },
  });
  vi.mocked(getWrongSentenceDetail).mockResolvedValue({
    item,
    blanks: [{ index: 0, word: "momentum", lastCorrect: false, meaning: "动力", wordReviewStage: 0, wordReviewStatus: "due" }],
    attempts: [{ recordId: 301, correct: false, costMs: 2100, practiceContext: "review", actionType: "answer", answeredAt: "2026-08-02T21:00:00" }],
    reviewStages: [
      { stage: 0, label: "立即", state: "current" },
      { stage: 1, label: "7 天", state: "upcoming" },
      { stage: 2, label: "15 天", state: "upcoming" },
      { stage: 3, label: "完成", state: "upcoming" },
    ],
  });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("wrong sentence collection", () => {
  it("loads active items and expands a learning-focused detail", async () => {
    render(<WrongSentenceCollection token="token" onClose={vi.fn()} onAuthExpired={vi.fn()} />);

    expect(screen.getByText("句子错题集")).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "待复习" })).toHaveAttribute("aria-selected", "true");
    expect(await screen.findByText("立即复习")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "展开" }));

    expect(await screen.findByText("复习轨迹")).toBeInTheDocument();
    expect(screen.getByText("momentum")).toBeInTheDocument();
    expect(screen.queryByText("你的答案")).not.toBeInTheDocument();
  });

  it("keeps close and completed filters accessible", async () => {
    const onClose = vi.fn();
    render(<WrongSentenceCollection token="token" onClose={onClose} onAuthExpired={vi.fn()} />);
    await screen.findByText("立即复习");

    fireEvent.click(screen.getByRole("button", { name: "关闭错题集" }));
    expect(onClose).toHaveBeenCalledOnce();

    fireEvent.click(screen.getByRole("tab", { name: "已完成" }));
    expect(getWrongSentences).toHaveBeenLastCalledWith("token", expect.objectContaining({ status: "completed" }));
  });

  it("moves focus into the modal, traps it, and restores it on close", () => {
    const previous = document.createElement("button");
    document.body.append(previous);
    previous.focus();

    const { unmount } = render(
      <WrongSentenceCollection token="token" onClose={vi.fn()} onAuthExpired={vi.fn()} />,
    );
    const dialog = screen.getByRole("dialog");
    expect(dialog).toHaveFocus();

    fireEvent.keyDown(dialog, { key: "Tab", shiftKey: true });
    expect(dialog).toContainElement(document.activeElement as HTMLElement);
    expect(dialog).not.toHaveFocus();

    unmount();
    expect(previous).toHaveFocus();
    previous.remove();
  });
});
