export interface AppUserItem {
  id: number;
  username: string;
  nickname: string;
  rank: number;
  exp: number;
  totalWins: number;
  totalGames: number;
  currentWinStreak: number;
  trainingRank: number;
  trainingExp: number;
  trainingTotalWins: number;
  trainingTotalGames: number;
  createTime: string;
  updateTime: string;
}

export interface UserTrainingAnswerDetail {
  id: number;
  roundNo: number;
  wordContent: string;
  wordDifficulty: number;
  option1: string;
  option2: string;
  option3: string;
  option4: string;
  correctAnswerIndex: number;
  selectedAnswerIndex?: number | null;
  isCorrect: number;
  score: number;
  answerTimeMs: number;
}

export interface UserTrainingRound {
  recordId: number;
  mode: "solo_training" | "match" | string;
  startTime: string;
  durationSeconds: number;
  trainingDifficultyGroup: string;
  trainingDifficultyLevel: string;
  opponentName: string;
  resultLabel: string;
  correctCount: number;
  totalCount: number;
  score: number;
  details: UserTrainingAnswerDetail[];
}

export interface UserWrongWordHistory {
  detailId: number;
  recordId: number;
  startTime: string;
  mode: "solo_training" | "match" | string;
  trainingDifficultyGroup: string;
  trainingDifficultyLevel: string;
  roundNo: number;
  wordDifficulty: number;
  option1: string;
  option2: string;
  option3: string;
  option4: string;
  correctAnswerIndex: number;
  selectedAnswerIndex?: number | null;
  answerTimeMs: number;
}

export interface UserWrongWordItem {
  userId: number;
  userName: string;
  wordContent: string;
  wrongCount: number;
  totalAttempts: number;
  avgDifficulty: number;
  lastWrongTime: string;
  latestMode: "solo_training" | "match" | string;
  latestGroup: string;
  latestLevel: string;
  reviewStatus: "waiting_sentence" | "due" | "waiting" | string;
  reviewStage: number;
  nextReviewTime?: string | null;
  recentHistories: UserWrongWordHistory[];
}

export interface UserClozeWrongHistory {
  recordId: number;
  attemptNo: number;
  answerText: string;
  answers: string[];
  expectedWords: string[];
  costMs: number;
  createTime: string;
}

export interface UserClozeWrongItem {
  userId: number;
  userName: string;
  clozeItemId: number;
  word: string;
  words: string[];
  blankWords: string[];
  sentence: string;
  translationZh: string;
  clozeSentence: string;
  source: string;
  wrongCount: number;
  totalAttempts: number;
  lastWrongTime: string;
  latestAttemptNo: number;
  recentHistories: UserClozeWrongHistory[];
}

export interface UserMasteredWordItem {
  userId: number;
  userName: string;
  wordId: number;
  wordContent: string;
  correctMeaning: string;
  status: "learning" | "mastered" | string;
  stage: number;
  correctCount: number;
  wordDifficulty: number;
  libraryName: string;
  libraryMeaning: string;
  firstCorrectTime?: string | null;
  day1CorrectTime?: string | null;
  day7CorrectTime?: string | null;
  nextReviewTime?: string | null;
  lastCorrectTime?: string | null;
  masteredTime?: string | null;
}
