// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ReviewPracticeLauncher, SoloTrainingLauncher } from "../src/components/PracticeLaunchers";

afterEach(cleanup);

describe("practice launchers", () => {
  it("keeps the review home focused on review and solo entry", () => {
    const onStart = vi.fn();
    const onOpenSolo = vi.fn();
    const onOpenWrongSentences = vi.fn();
    render(
      <ReviewPracticeLauncher
        dueCount={3}
        wrongCount={18}
        loading={false}
        onStart={onStart}
        onOpenWrongSentences={onOpenWrongSentences}
        onOpenSolo={onOpenSolo}
      />,
    );

    expect(screen.getByRole("button", { name: "开始答题" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "错题集 18" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "单独训练" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "选择难度" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "开始答题" }));
    fireEvent.click(screen.getByRole("button", { name: "错题集 18" }));
    fireEvent.click(screen.getByRole("button", { name: "单独训练" }));
    expect(onStart).toHaveBeenCalledOnce();
    expect(onOpenWrongSentences).toHaveBeenCalledOnce();
    expect(onOpenSolo).toHaveBeenCalledOnce();
  });

  it("keeps an empty wrong collection accessible", () => {
    render(
      <ReviewPracticeLauncher
        dueCount={0}
        wrongCount={0}
        loading={false}
        onStart={vi.fn()}
        onOpenWrongSentences={vi.fn()}
        onOpenSolo={vi.fn()}
      />,
    );

    expect(screen.getByRole("button", { name: "错题集 0" })).toBeEnabled();
  });

  it("shows all solo training controls", () => {
    const onClose = vi.fn();
    render(
      <SoloTrainingLauncher
        selectedLabel="初中英语"
        batchText="每轮 10 句"
        loading={false}
        showClose
        onClose={onClose}
        onChooseDifficulty={vi.fn()}
        onOpenSentences={vi.fn()}
        onOpenResults={vi.fn()}
        onStart={vi.fn()}
      />,
    );

    for (const name of ["关闭单独训练", "选择难度", "句子列表", "答题结果", "开始训练"]) {
      expect(screen.getByRole("button", { name })).toBeInTheDocument();
    }
    expect(screen.queryByRole("button", { name: "返回" })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "关闭单独训练" }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("hides the solo close action while a child fullscreen layer is open", () => {
    render(
      <SoloTrainingLauncher
        selectedLabel="初中英语"
        batchText="每轮 10 句"
        loading={false}
        showClose={false}
        onClose={vi.fn()}
        onChooseDifficulty={vi.fn()}
        onOpenSentences={vi.fn()}
        onOpenResults={vi.fn()}
        onStart={vi.fn()}
      />,
    );

    expect(screen.queryByRole("button", { name: "关闭单独训练" })).not.toBeInTheDocument();
  });
});
