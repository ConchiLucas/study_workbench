export interface ClozeResultUserSummary {
  userId: number;
  userName: string;
  totalCount: number;
  latestTime: string;
  latestWords: string[];
  latestSource: string;
}

export type ClozeSourceTraceStatus = "available" | "historical" | "missing";

export interface ClozeSourceWord {
  word: string;
  traceStatus: ClozeSourceTraceStatus;
  source: string;
  sourceLabel: string;
  sourceEventId: number;
  sourceAnswerDetailId: number;
  sourceRecordId: number;
  sourceWordId: number;
  wrongTime: string | null;
  mode: string;
  difficultyGroup: string;
  difficultyLevel: string;
  wordDifficulty: number | null;
  answerTimeMs: number | null;
  selectedAnswer: string;
  correctAnswer: string;
  traceText: string;
}

export interface ClozeResultItem {
  id: number;
  userId: number;
  userName: string;
  word: string;
  words: string[];
  blankWords: string[];
  sentence: string;
  translationZh: string;
  explanationZh: string;
  clozeSentence: string;
  providerId: string;
  providerLabel: string;
  model: string;
  source: string;
  sourceEventIds: number[];
  sourceAnswerDetailIds: number[];
  sourceRecordIds: number[];
  sourceWordIds: number[];
  sourceWords: ClozeSourceWord[];
  createTime: string;
  updateTime: string;
}

export interface PaginatedResult<T> {
  list: T[];
  total: number;
  page: number;
  pageSize: number;
}
