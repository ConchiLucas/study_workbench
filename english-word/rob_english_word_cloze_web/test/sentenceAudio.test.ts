import assert from "node:assert/strict";
import test from "node:test";

import { resolveSentenceAudioSource } from "../src/lib/sentenceAudio.ts";

test("uses the MinIO proxy URL when the task provides one", () => {
  assert.deepEqual(
    resolveSentenceAudioSource(" /ai-file-navigation/word_clean_tts/value.mp3 "),
    {
      kind: "minio",
      url: "/ai-file-navigation/word_clean_tts/value.mp3",
    },
  );
});

test("falls back to browser speech for old tasks without an audio URL", () => {
  assert.deepEqual(resolveSentenceAudioSource(""), { kind: "speech" });
  assert.deepEqual(resolveSentenceAudioSource(undefined), { kind: "speech" });
});
