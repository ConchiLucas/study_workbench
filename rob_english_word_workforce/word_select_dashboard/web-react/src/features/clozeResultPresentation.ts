import type { ClozeResultItem, ClozeSourceWord } from "../types/clozeResult.ts";

interface SourceWordCountInput {
  sourceWords?: Array<Pick<ClozeSourceWord, "word">>;
  words?: string[];
  word?: string;
}

interface EntryModeInput {
  sourceLabel?: string;
  mode?: string;
}

interface DifficultyInput {
  difficultyGroup?: string;
  difficultyLevel?: string;
}

const difficultyGroupLabels: Record<string, string> = {
  rank: "段位难度",
  primary: "小学英语",
  junior: "初中英语",
  senior: "高中英语",
  college: "大学英语",
  entrance: "升学考试英语",
  business_abroad: "商务与出国英语",
  professional: "专业英语",
  advanced_exam: "高阶考试英语",
};

const difficultyLevelLabels: Record<string, string> = {
  rank_current: "段位难度",
  college_cet4: "四级",
  college_cet6: "六级",
  entrance_kaoyan: "考研",
  business_bec: "BEC",
  business_ielts: "雅思",
  business_toefl: "托福",
  business_gmat: "GMAT",
  professional_tem4: "专四级",
  professional_tem8: "专八级",
  advanced_gre: "GRE",
  advanced_sat: "SAT",
};

type TraceInput = Partial<
  Pick<
    ClozeSourceWord,
    "traceStatus" | "traceText" | "sourceEventId" | "sourceAnswerDetailId" | "sourceRecordId"
  >
>;

export function sourceWordCount(item: SourceWordCountInput) {
  if (item.sourceWords && item.sourceWords.length > 0) {
    return item.sourceWords.length;
  }
  const words = (item.words ?? []).filter((word) => word.trim());
  if (words.length > 0) {
    return words.length;
  }
  return item.word?.trim() ? 1 : 0;
}

export function sourceWordEntryModeLabel(word: EntryModeInput) {
  const sourceLabel = word.sourceLabel?.trim() ?? "";
  const mode = word.mode?.trim() ?? "";
  if (sourceLabel && mode && mode !== "-") {
    return `${sourceLabel} / ${mode}`;
  }
  return sourceLabel || (mode && mode !== "-" ? mode : "-");
}

export function sourceWordDifficultyLabel(word: DifficultyInput) {
  const rawGroup = word.difficultyGroup?.trim() ?? "";
  const rawLevel = word.difficultyLevel?.trim() ?? "";
  const group = difficultyGroupLabels[rawGroup] ?? rawGroup;
  const level = difficultyLevelLabels[rawLevel] ?? rawLevel;
  if (rawGroup && rawGroup === rawLevel && group !== rawGroup) {
    return group;
  }
  if (group && level) {
    return `${group} / ${level}`;
  }
  return group || level || "-";
}

export function formatSourceAnswerTime(value?: number | null) {
  if (value === undefined || value === null || value < 0) {
    return "-";
  }
  if (value < 1000) {
    return `${value}ms`;
  }
  const seconds = value / 1000;
  return `${Number.isInteger(seconds) ? seconds.toFixed(0) : seconds.toFixed(1)}s`;
}

export function sourceWordTraceLabel(word: TraceInput) {
  const traceText = word.traceText?.trim();
  if (traceText) {
    return traceText;
  }
  if (word.traceStatus === "historical") {
    return "历史生成，无答题来源";
  }
  if (word.traceStatus !== "available") {
    return "来源记录缺失";
  }
  return `事件 #${word.sourceEventId ?? 0} · 答题 #${word.sourceAnswerDetailId ?? 0} · 记录 #${word.sourceRecordId ?? 0}`;
}

export function clozeResultSourceLabel(
  item: Pick<ClozeResultItem, "source" | "sourceEventIds">,
) {
  if (item.sourceEventIds.length > 0) {
    return "外部错题触发";
  }
  if (item.source === "word-agent") {
    return "Python 生成";
  }
  return item.source || "-";
}

export function sourceWordsForDisplay(
  item: Pick<ClozeResultItem, "sourceWords" | "words" | "word" | "source">,
): ClozeSourceWord[] {
  if (item.sourceWords?.length > 0) {
    return item.sourceWords;
  }
  const words = item.words.length > 0 ? item.words : [item.word].filter(Boolean);
  return words.map((word) => ({
    word,
    traceStatus: "historical",
    source: item.source,
    sourceLabel: "历史生成",
    sourceEventId: 0,
    sourceAnswerDetailId: 0,
    sourceRecordId: 0,
    sourceWordId: 0,
    wrongTime: null,
    mode: "-",
    difficultyGroup: "",
    difficultyLevel: "",
    wordDifficulty: null,
    answerTimeMs: null,
    selectedAnswer: "",
    correctAnswer: "",
    traceText: "历史生成，无答题来源",
  }));
}
