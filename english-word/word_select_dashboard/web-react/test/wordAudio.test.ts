import assert from "node:assert/strict";
import test from "node:test";

import { playableTTSAudioURL } from "../src/utils/wordAudio.ts";

test("returns the trimmed MinIO URL for a successful TTS record", () => {
  assert.equal(
    playableTTSAudioURL("success", "  /ai-file-navigation/word_clean_tts/abase.wav  "),
    "/ai-file-navigation/word_clean_tts/abase.wav",
  );
});

test("returns null when TTS generation did not succeed", () => {
  assert.equal(playableTTSAudioURL("failed", "/ai-file-navigation/word_clean_tts/abase.wav"), null);
});

test("returns null when the audio URL is blank", () => {
  assert.equal(playableTTSAudioURL("success", "   "), null);
});
