import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const source = readFileSync(new URL("../src/App.tsx", import.meta.url), "utf8");

test("word and sentence audio use distinct targets in one player", () => {
  assert.match(source, /`word:\$\{item\.id\}`/);
  assert.match(source, /`sentence:\$\{item\.id\}`/);
  assert.match(source, /item\.wordTtsStatus/);
  assert.match(source, /item\.wordTtsObjectUrl/);
  assert.match(source, /播放.*单词发音/);
});
