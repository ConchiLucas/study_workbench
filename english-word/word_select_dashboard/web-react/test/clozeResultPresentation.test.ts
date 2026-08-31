import assert from "node:assert/strict";
import { existsSync } from "node:fs";
import test from "node:test";

const helperUrl = new URL("../src/features/clozeResultPresentation.ts", import.meta.url);

test("source word count uses enriched rows and falls back to generated words", async () => {
  assert.equal(existsSync(helperUrl), true, "cloze result presentation helper must exist");
  const { sourceWordCount } = await import(helperUrl.href);

  assert.equal(
    sourceWordCount({
      sourceWords: [{ word: "raw" }],
      words: ["raw", "stone"],
      word: "raw",
    }),
    1,
  );
  assert.equal(
    sourceWordCount({
      sourceWords: [],
      words: ["raw", "stone"],
      word: "raw",
    }),
    2,
  );
  assert.equal(sourceWordCount({ sourceWords: [], words: [], word: "raw" }), 1);
});

test("entry mode and library difficulty stay in separate columns", async () => {
  assert.equal(existsSync(helperUrl), true, "cloze result presentation helper must exist");
  const { sourceWordDifficultyLabel, sourceWordEntryModeLabel } = await import(helperUrl.href);

  assert.equal(
    sourceWordEntryModeLabel({
      sourceLabel: "游戏答题",
      mode: "单人训练",
    }),
    "游戏答题 / 单人训练",
  );
  assert.equal(
    sourceWordDifficultyLabel({
      difficultyGroup: "大学英语",
      difficultyLevel: "cet4",
    }),
    "大学英语 / cet4",
  );
  assert.equal(
    sourceWordDifficultyLabel({
      difficultyGroup: "advanced_exam",
      difficultyLevel: "advanced_exam",
    }),
    "高阶考试英语",
  );
  assert.equal(
    sourceWordDifficultyLabel({
      difficultyGroup: "college",
      difficultyLevel: "college_cet4",
    }),
    "大学英语 / 四级",
  );
  assert.equal(
    sourceWordEntryModeLabel({
      sourceLabel: "句子挖空练习",
      mode: "-",
    }),
    "句子挖空练习",
  );
  assert.equal(
    sourceWordDifficultyLabel({
      difficultyGroup: "",
      difficultyLevel: "",
    }),
    "-",
  );
});

test("answer time and trace fallbacks remain explicit", async () => {
  assert.equal(existsSync(helperUrl), true, "cloze result presentation helper must exist");
  const { formatSourceAnswerTime, sourceWordTraceLabel } = await import(helperUrl.href);

  assert.equal(formatSourceAnswerTime(1200), "1.2s");
  assert.equal(formatSourceAnswerTime(850), "850ms");
  assert.equal(formatSourceAnswerTime(undefined), "-");
  assert.equal(
    sourceWordTraceLabel({ traceStatus: "historical", traceText: "" }),
    "历史生成，无答题来源",
  );
  assert.equal(
    sourceWordTraceLabel({ traceStatus: "missing", traceText: "" }),
    "来源记录缺失",
  );
  assert.equal(
    sourceWordTraceLabel({
      traceStatus: "available",
      traceText: "",
      sourceEventId: 21,
      sourceAnswerDetailId: 101,
      sourceRecordId: 11,
    }),
    "事件 #21 · 答题 #101 · 记录 #11",
  );
});
