import { describe, expect, it } from "vitest";
import { resolveWordAudioSource, wordAudioButtonLabel } from "../src/lib/wordAudio";

describe("word audio", () => {
  it("uses only a provided MinIO URL", () => {
    expect(resolveWordAudioSource(" /ai-file-navigation/word_tts/value.wav ")).toEqual({
      kind: "minio",
      url: "/ai-file-navigation/word_tts/value.wav",
    });
  });

  it("marks missing audio without a speech fallback", () => {
    expect(resolveWordAudioSource("")).toEqual({ kind: "missing" });
    expect(resolveWordAudioSource(undefined)).toEqual({ kind: "missing" });
  });

  it("labels inflected answers as base-word playback", () => {
    expect(wordAudioButtonLabel("value", "values", true)).toBe("播放原词 value");
    expect(wordAudioButtonLabel("value", "value", false)).toBe("暂无原词音频");
  });
});
