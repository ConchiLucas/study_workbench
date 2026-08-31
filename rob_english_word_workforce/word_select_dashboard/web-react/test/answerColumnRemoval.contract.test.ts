import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const appSource = readFileSync(new URL("../src/App.tsx", import.meta.url), "utf8");
const stylesSource = readFileSync(new URL("../src/styles/app.css", import.meta.url), "utf8");
const userApiSource = readFileSync(
  new URL("../../server/api/v1/system/sys_app_user.go", import.meta.url),
  "utf8",
);

test("admin user result and wrong-word views remove submitted answers", () => {
  assert.doesNotMatch(appSource, /你的答案/);
  assert.match(appSource, /正确答案/);
  assert.match(stylesSource, /\.user-result-word-grid/);
  assert.match(stylesSource, /\.wrong-detail-head/);
  assert.match(stylesSource, /\.user-wrong-history-grid/);
  assert.match(stylesSource, /\.user-cloze-history-grid/);
});

test("admin primary wrong-word list uses unfinished review progress", () => {
  assert.match(userApiSource, /wrong_word_review_progress/);
  assert.match(userApiSource, /status <> 'completed'/);
  assert.match(userApiSource, /sentence_cloze_answer_record/);
  assert.match(userApiSource, /jsonb_array_elements_text/);
  assert.match(userApiSource, /latest_event/);
  assert.match(userApiSource, /COALESCE\(d\.create_time, r\.start_time, CURRENT_TIMESTAMP\) AS event_time/);
  assert.match(appSource, /未完成复习/);
});
