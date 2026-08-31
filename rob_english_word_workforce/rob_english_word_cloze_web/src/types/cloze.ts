export interface AuthResponse {
  token: string;
  userId: number;
  username: string;
  nickname?: string | null;
}

export interface ClozePracticePreference {
  soloDifficultyGroup: string;
  soloDifficultyLevel: string;
}

export interface ClozePracticeTask {
  id: number;
  word?: string | null;
  wordAudioUrl?: string | null;
  sentence?: string | null;
  sentenceAudioUrl?: string | null;
  clozeSentence: string;
  translationZh: string;
  blankCount: number;
  blankLengths?: number[];
  attemptCount: number;
  wrongCount: number;
  difficultyGroup?: string | null;
  difficultyLevel?: string | null;
  difficultyLabel?: string | null;
  source?: string | null;
  model?: string | null;
  latestAnswerCorrect?: boolean | null;
  latestAnswerTime?: string | null;
  nextReviewTime?: string | null;
  createTime: string;
}

export interface ClozePracticeAnswerResponse {
  recordId: number;
  clozeItemId: number;
  correct: boolean;
  answerText: string;
  answers: string[];
  expectedWords: string[];
  attemptNo: number;
  message: string;
}

export interface ClozePracticeHistoryItem {
  id: number;
  clozeItemId: number;
  clozeSentence: string;
  translationZh: string;
  answerText: string;
  expectedWordsJson: string;
  isCorrect: boolean;
  attemptNo: number;
  costMs?: number | null;
  createTime: string;
}

export interface ClozePracticeStats {
  totalTasks: number;
  completedTasks: number;
  pendingTasks: number;
  totalAnswers: number;
  correctAnswers: number;
  wrongAnswers: number;
  activeWrongSentences: number;
  dueReviewTasks: number;
  accuracy: number;
}

export interface WrongSentenceItem {
  progressId: number;
  clozeItemId: number;
  clozeSentence: string;
  sentence: string;
  translationZh: string;
  targetWords: string[];
  wrongBlankIndexes: number[];
  wrongBlankCount: number;
  practiceContext: "review" | "solo";
  contentSource: string;
  difficultyLabel: string;
  status: "active" | "completed";
  reviewStage: number;
  nextReviewTime?: string | null;
  wrongCount: number;
  firstWrongTime: string;
  lastWrongTime: string;
  lastCostMs?: number | null;
}

export interface WrongSentenceAttempt {
  recordId: number;
  correct: boolean;
  costMs?: number | null;
  practiceContext: "review" | "solo";
  actionType: "answer" | "reveal";
  answeredAt: string;
}

export interface WrongSentenceBlankReview {
  index: number;
  word: string;
  lastCorrect?: boolean | null;
  meaning: string;
  wordReviewStage?: number | null;
  wordReviewStatus?: string | null;
}

export interface WrongSentenceReviewStage {
  stage: number;
  label: string;
  state: "completed" | "current" | "upcoming";
}

export interface WrongSentenceDetail {
  item: WrongSentenceItem;
  blanks: WrongSentenceBlankReview[];
  attempts: WrongSentenceAttempt[];
  reviewStages: WrongSentenceReviewStage[];
}

export interface WrongSentenceSummary {
  activeCount: number;
  dueCount: number;
  stage1Count: number;
  stage2Count: number;
  completedCount: number;
}

export interface WrongSentencePageResponse {
  items: WrongSentenceItem[];
  total: number;
  current: number;
  pages: number;
  summary: WrongSentenceSummary;
}

export interface WrongSentenceQuery {
  status?: "active" | "completed";
  source?: "all" | "review" | "solo";
  availability?: "all" | "due" | "waiting";
  keyword?: string;
  sort?: "nextReview" | "recent" | "wrongCount";
  page?: number;
  size?: number;
}
