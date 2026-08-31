export interface WordLibraryItem {
  id: number;
  libraryName: string;
  libraryMeaning: string;
  status: number;
  wordCount: number;
  createdBy?: number | null;
  createTime: string;
  updateTime: string;
}

export interface WordLibraryWordItem {
  id: number;
  libraryId: number;
  word: string;
  meaning: string;
  pronunciationUs: string;
  pronunciationUk: string;
  frequency: number;
  difficulty: number;
  status: number;
  phrase: string;
  phraseTranslation: string;
  sentence: string;
  sentenceTranslation: string;
  bestSentenceTtsStatus: string;
  bestSentenceTtsBucket: string;
  bestSentenceTtsObjectKey: string;
  bestSentenceTtsObjectUrl: string;
  createTime: string;
  updateTime: string;
}

export interface WordCleanItem {
  id: number;
  word: string;
  meaning: string;
  difficulty: number;
  frequency: number;
  sentence: string;
  pepDifficulty?: number | null;
  pepDifficultyLabel: string;
  sourceDifficulty?: number | null;
  sourceLabel: string;
  wordTtsStatus: string;
  wordTtsBucket: string;
  wordTtsObjectKey: string;
  wordTtsObjectUrl: string;
  bestSentenceId?: number | null;
  bestSourceSentenceId?: number | null;
  bestSourceModelName: string;
  bestSentence: string;
  bestSentenceTranslation: string;
  bestSentenceScore?: number | null;
  bestSentenceScoreReason: string;
  bestSentenceScoreModelName: string;
  bestSentenceScoredAt?: string | null;
  bestSentenceTtsStatus: string;
  bestSentenceTtsBucket: string;
  bestSentenceTtsObjectKey: string;
  bestSentenceTtsObjectUrl: string;
  bestSentenceTtsContentType: string;
  bestSentenceTtsFileSize?: number | null;
  bestSentenceTtsDurationMs?: number | null;
  bestSentenceTtsGeneratedAt?: string | null;
  bestSentenceTtsErrorMessage: string;
}

export interface WordCleanSentenceItem {
  id: number;
  wordCleanId: number;
  word: string;
  modelName: string;
  sentence: string;
  sentenceTranslation: string;
  score?: number | null;
  scoreReason: string;
  scoreModelName: string;
  scoredAt?: string | null;
}

export interface ScoreWordCleanSentencesResponse {
  status: string;
  message: string;
  judgeModel: string;
  processedCount: number;
  scoredCount: number;
  failedCount: number;
  items: Array<{
    id: number;
    wordCleanId: number;
    word: string;
    modelName: string;
    score: number;
    scoreReason: string;
  }>;
  bestItems: Array<{
    id: number;
    wordCleanId: number;
    word: string;
    meaning: string;
    sourceSentenceId: number;
    sourceModelName: string;
    sentence: string;
    sentenceTranslation: string;
    score: number;
    scoreReason: string;
    scoreModelName: string;
    scoredAt?: string | null;
    ttsStatus: string;
    ttsBucket: string;
    ttsObjectKey: string;
    ttsObjectUrl: string;
  }>;
}

export interface PaginatedResult<T> {
  list: T[];
  total: number;
  page: number;
  pageSize: number;
}
