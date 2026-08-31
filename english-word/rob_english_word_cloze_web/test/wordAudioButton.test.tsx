// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { WordAudioButton } from "../src/components/WordAudioButton";

describe("WordAudioButton", () => {
  it("plays the supplied TTS for an inflected answer", () => {
    const onPlay = vi.fn();
    render(
      <WordAudioButton
        baseWord="value"
        displayedWord="values"
        audioUrl="/word/value.wav"
        onPlay={onPlay}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "播放原词 value" }));

    expect(onPlay).toHaveBeenCalledWith("/word/value.wav");
  });

  it("is disabled when word TTS is missing", () => {
    render(
      <WordAudioButton
        baseWord="value"
        displayedWord="value"
        audioUrl=""
        onPlay={vi.fn()}
      />,
    );

    expect(screen.getByRole("button", { name: "暂无原词音频" })).toBeDisabled();
  });
});
