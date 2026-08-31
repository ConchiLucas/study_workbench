import { Fragment, useEffect, useMemo, useRef, useState, type CSSProperties, type FormEvent, type KeyboardEvent } from "react";
import {
  ApiError,
  createDifficultyBatch,
  getAnsweredTasks,
  getClozePreferences,
  getDueReviewTasks,
  getHistory,
  getPendingTasks,
  getStats,
  login,
  register,
  submitAnswer,
  updateSoloDifficulty,
  type Credentials,
} from "./lib/api";
import { ReviewPracticeLauncher, SoloTrainingLauncher } from "./components/PracticeLaunchers";
import { FullscreenCloseButton } from "./components/FullscreenCloseButton";
import { WrongSentenceCollection } from "./components/WrongSentenceCollection";
import { WordAudioButton } from "./components/WordAudioButton";
import {
  nextLauncherModeAfterBatch,
  shouldReuseSubmission,
  type LauncherMode,
  type PracticeSource,
  type SubmissionIdentity,
} from "./lib/practiceMode";
import { resolveSentenceAudioSource } from "./lib/sentenceAudio";
import { resolveWordAudioSource } from "./lib/wordAudio";
import {
  DEFAULT_SOLO_DIFFICULTY,
  SOLO_DIFFICULTY_OPTIONS,
  normalizeSoloDifficulty,
  selectedSoloDifficultyText,
  type SelectedSoloDifficulty as SelectedDifficulty,
  type SoloDifficultyChildOption as DifficultyChildOption,
  type SoloDifficultyParentKey as DifficultyParentKey,
  type SoloDifficultyParentOption as DifficultyParentOption,
} from "./lib/soloDifficulty";
import type {
  AuthResponse,
  ClozePracticeAnswerResponse,
  ClozePracticeHistoryItem,
  ClozePracticeStats,
  ClozePracticeTask,
} from "./types/cloze";

const AUTH_STORAGE_KEY = "cloze_practice_auth";
const TOKEN_EXPIRY_SKEW_MS = 30_000;
const DIFFICULTY_BATCH_SIZE = 10;
const FALLBACK_BLANK_LENGTH = 8;
const MIN_BLANK_LENGTH = 2;
const SENTENCE_BLANK_SPEECH_TEXT = "blank";
const RESULT_ROUND_SIZE = 10;
const RESULT_ROUND_GAP_MS = 30 * 60 * 1000;

type AuthMode = "login" | "register";
type SentenceListMode = "pending" | "review" | "mastered" | "wrong";

interface WordPhonetic {
  word: string;
  phonetic: string;
}

interface ClozeResultRound {
  id: string;
  index: number;
  startTime: string;
  endTime: string;
  details: ClozePracticeHistoryItem[];
  correctCount: number;
  wrongCount: number;
  totalCostMs: number;
  accuracy: number;
}

type InlineAnswerStyle = CSSProperties & {
  "--inline-answer-width": string;
};

const difficultyOptions = SOLO_DIFFICULTY_OPTIONS;

function readStoredAuth(): AuthResponse | null {
  const raw = window.localStorage.getItem(AUTH_STORAGE_KEY);
  if (!raw) {
    return null;
  }
  try {
    const value = JSON.parse(raw) as AuthResponse;
    if (!value.token || isTokenExpired(value.token)) {
      clearStoredAuth();
      return null;
    }
    return value;
  } catch {
    clearStoredAuth();
    return null;
  }
}

function clearStoredAuth() {
  window.localStorage.removeItem(AUTH_STORAGE_KEY);
}

function persistAuth(auth: AuthResponse) {
  window.localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(auth));
}

function isTokenExpired(token: string) {
  const payload = decodeJwtPayload(token);
  if (!payload || typeof payload.exp !== "number") {
    return false;
  }
  return payload.exp * 1000 <= Date.now() + TOKEN_EXPIRY_SKEW_MS;
}

function decodeJwtPayload(token: string): { exp?: number } | null {
  try {
    const [, payload] = token.split(".");
    if (!payload) {
      return null;
    }
    const normalized = payload.replace(/-/g, "+").replace(/_/g, "/");
    const padded = normalized.padEnd(normalized.length + ((4 - (normalized.length % 4)) % 4), "=");
    return JSON.parse(window.atob(padded));
  } catch {
    return null;
  }
}

function isAuthError(error: unknown) {
  return error instanceof ApiError && (error.status === 401 || error.status === 403);
}

function formatTime(value?: string | null) {
  if (!value) {
    return "-";
  }
  return new Date(value).toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatDateTime(value?: string | null) {
  if (!value) {
    return "-";
  }
  return new Date(value).toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatResultTime(value?: string | null) {
  if (!value) {
    return "-";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  const hour = String(date.getHours()).padStart(2, "0");
  const minute = String(date.getMinutes()).padStart(2, "0");
  return `${year}/${month}/${day} ${hour}:${minute}`;
}

function formatCostMs(ms?: number | null) {
  if (!ms) {
    return "-";
  }
  if (ms < 1000) {
    return `${ms}ms`;
  }
  const seconds = ms / 1000;
  if (seconds < 60) {
    return `${seconds.toFixed(1)}s`;
  }
  const minutes = Math.floor(seconds / 60);
  const rest = Math.round(seconds % 60);
  return `${minutes}m ${rest}s`;
}

function parseExpectedWords(json: string) {
  try {
    const value = JSON.parse(json);
    return Array.isArray(value) ? value.join(", ") : json;
  } catch {
    return json;
  }
}

function groupClozeResultRounds(items: ClozePracticeHistoryItem[]) {
  const sorted = [...items].sort((left, right) => {
    const byTime = new Date(left.createTime).getTime() - new Date(right.createTime).getTime();
    return byTime || left.id - right.id;
  });
  const groups: ClozePracticeHistoryItem[][] = [];

  for (const item of sorted) {
    const current = groups[groups.length - 1];
    const last = current?.[current.length - 1];
    const gap = last ? new Date(item.createTime).getTime() - new Date(last.createTime).getTime() : 0;
    if (!current || current.length >= RESULT_ROUND_SIZE || gap > RESULT_ROUND_GAP_MS) {
      groups.push([item]);
    } else {
      current.push(item);
    }
  }

  return groups.map((details, index) => {
    const correctCount = details.filter((item) => item.isCorrect).length;
    const totalCostMs = details.reduce((sum, item) => sum + (item.costMs || 0), 0);
    return {
      id: `${details[0]?.id || index}-${details[details.length - 1]?.id || index}`,
      index: index + 1,
      startTime: details[0]?.createTime || "",
      endTime: details[details.length - 1]?.createTime || "",
      details,
      correctCount,
      wrongCount: details.length - correctCount,
      totalCostMs,
      accuracy: details.length ? Math.round((correctCount / details.length) * 100) : 0,
    } satisfies ClozeResultRound;
  }).reverse();
}

function formatReviewAvailability(task: ClozePracticeTask, mode: SentenceListMode) {
  if (mode === "mastered") {
    return formatDateTime(task.latestAnswerTime || task.createTime);
  }
  if (!task.nextReviewTime) {
    return "现在可答";
  }
  const nextReview = new Date(task.nextReviewTime);
  return nextReview.getTime() <= Date.now() ? "现在可答" : formatDateTime(task.nextReviewTime);
}

function sentenceListTimeHeader(mode: SentenceListMode) {
  if (mode === "mastered") {
    return "最近答对";
  }
  if (mode === "review") {
    return "下次可答";
  }
  return "可答时间";
}

function emptyAnswers(count: number) {
  return Array.from({ length: Math.max(count, 1) }, () => "");
}

function createSubmissionKey() {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function normalizeAnswer(value?: string) {
  return (value || "").trim().toLowerCase();
}

export default function App() {
  const [auth, setAuth] = useState<AuthResponse | null>(() => readStoredAuth());
  const [authMode, setAuthMode] = useState<AuthMode>("login");
  const [credentials, setCredentials] = useState<Credentials>({ username: "", password: "", nickname: "" });
  const [task, setTask] = useState<ClozePracticeTask | null>(null);
  const [answers, setAnswers] = useState<string[]>([""]);
  const [stats, setStats] = useState<ClozePracticeStats | null>(null);
  const [result, setResult] = useState<ClozePracticeAnswerResponse | null>(null);
  const [startedAt, setStartedAt] = useState<number>(() => Date.now());
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState("");
  const practiceFormRef = useRef<HTMLFormElement | null>(null);
  const answerInputRefs = useRef<Array<HTMLInputElement | null>>([]);

  // New UI states
  const [activeInputIndex, setActiveInputIndex] = useState<number | null>(null);
  const [phonetics, setPhonetics] = useState<Record<string, WordPhonetic>>({});
  const [fetchingPhonetics, setFetchingPhonetics] = useState(false);
  const [toast, setToast] = useState<{ message: string; type: "info" | "success" | "error" } | null>(null);
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const [isPracticeActive, setIsPracticeActive] = useState(false);
  const [isSpeakingSentence, setIsSpeakingSentence] = useState(false);
  const [showDifficultyPicker, setShowDifficultyPicker] = useState(false);
  const [showPendingTasks, setShowPendingTasks] = useState(false);
  const [showAnswerResults, setShowAnswerResults] = useState(false);
  const [showWrongSentences, setShowWrongSentences] = useState(false);
  const [pendingTasks, setPendingTasks] = useState<ClozePracticeTask[]>([]);
  const [answerResultRounds, setAnswerResultRounds] = useState<ClozeResultRound[]>([]);
  const [sentenceListMode, setSentenceListMode] = useState<SentenceListMode>("pending");
  const [pendingTasksLoading, setPendingTasksLoading] = useState(false);
  const [answerResultsLoading, setAnswerResultsLoading] = useState(false);
  const [expandedResultRounds, setExpandedResultRounds] = useState<Set<string>>(() => new Set());
  const [launcherMode, setLauncherMode] = useState<LauncherMode>("review");
  const [practiceSource, setPracticeSource] = useState<PracticeSource>("review");
  const [dueReviewTasks, setDueReviewTasks] = useState<ClozePracticeTask[]>([]);
  const [expandedDifficultyKey, setExpandedDifficultyKey] = useState<DifficultyParentKey | null>("junior");
  const [selectedDifficulty, setSelectedDifficulty] = useState<SelectedDifficulty>(DEFAULT_SOLO_DIFFICULTY);
  const [batchTasks, setBatchTasks] = useState<ClozePracticeTask[]>([]);
  const [batchIndex, setBatchIndex] = useState(0);
  const sentenceAudioRef = useRef<HTMLAudioElement | null>(null);
  const sentenceSpeechRef = useRef<SpeechSynthesisUtterance | null>(null);
  const pendingSubmissionRef = useRef<(SubmissionIdentity & { key: string }) | null>(null);

  const token = auth?.token || "";
  const displayName = auth?.nickname || auth?.username || "未登录";
  const canSubmit = Boolean(task && answers.length > 0 && answers.every((answer) => answer.trim()));
  const selectedDifficultyText = selectedSoloDifficultyText(selectedDifficulty);
  const batchProgressText = batchTasks.length > 0 ? `${batchIndex + 1}/${batchTasks.length}` : `每轮 ${DIFFICULTY_BATCH_SIZE} 句`;
  const sentenceListMeta = {
    pending: {
      kicker: "PENDING SENTENCES",
      title: "待答句子",
      subtitle: `共 ${pendingTasks.length} 条还需要作答的挖空句子`,
      countLabel: "待答句子",
      emptyTitle: "暂无待答句子",
      emptyText: "可以先选择难度并开始答题，系统会生成新的挖空句子。",
    },
    review: {
      kicker: "REVIEW SCHEDULE",
      title: "复习计划",
      subtitle: `共 ${pendingTasks.length} 条后续可能要作答的错题句子`,
      countLabel: "计划句子",
      emptyTitle: "暂无复习计划",
      emptyText: "答错的句子会进入这里，并按 1 天、7 天、15 天安排复习。",
    },
    mastered: {
      kicker: "MASTERED SENTENCES",
      title: "已会句子",
      subtitle: `共 ${pendingTasks.length} 条最新答对的挖空句子`,
      countLabel: "已会句子",
      emptyTitle: "暂无已会句子",
      emptyText: "答对的句子会自动进入这里。",
    },
    wrong: {
      kicker: "WRONG SENTENCES",
      title: "错题句子",
      subtitle: `共 ${pendingTasks.length} 条最新答错的挖空句子`,
      countLabel: "错题句子",
      emptyTitle: "暂无错题句子",
      emptyText: "答错的句子会自动进入复习计划，连续答对 3 次后转为已会。",
    },
  } satisfies Record<SentenceListMode, {
    kicker: string;
    title: string;
    subtitle: string;
    countLabel: string;
    emptyTitle: string;
    emptyText: string;
  }>;
  const activeSentenceListMeta = sentenceListMeta[sentenceListMode];
  const sentenceListTabs = (Object.keys(sentenceListMeta) as SentenceListMode[]).map((mode) => ({
    mode,
    label: sentenceListMeta[mode].title,
    count: mode === sentenceListMode ? pendingTasks.length : undefined,
  }));
  const resultSummary = useMemo(() => {
    const total = answerResultRounds.reduce((sum, round) => sum + round.details.length, 0);
    const correct = answerResultRounds.reduce((sum, round) => sum + round.correctCount, 0);
    const costMs = answerResultRounds.reduce((sum, round) => sum + round.totalCostMs, 0);
    return {
      rounds: answerResultRounds.length,
      total,
      correct,
      accuracy: total ? Math.round((correct / total) * 100) : 0,
      costMs,
    };
  }, [answerResultRounds]);

  const accuracyText = useMemo(() => {
    if (!stats) {
      return "0%";
    }
    return `${stats.accuracy.toFixed(1)}%`;
  }, [stats]);

  // Toast auto-dismissal
  useEffect(() => {
    if (!toast) return;
    const timer = setTimeout(() => setToast(null), 2200);
    return () => clearTimeout(timer);
  }, [toast]);

  function showToastMessage(msg: string, type: "info" | "success" | "error" = "info") {
    setToast({ message: msg, type });
  }

  function resetPracticeState() {
    setTask(null);
    setStats(null);
    setResult(null);
    setAnswers([""]);
    setBatchTasks([]);
    setBatchIndex(0);
    setIsPracticeActive(false);
    setShowWrongSentences(false);
    pendingSubmissionRef.current = null;
  }

  function handleAuthExpired() {
    clearStoredAuth();
    setAuth(null);
    resetPracticeState();
    setMessage("登录已过期，请重新登录");
    showToastMessage("登录已过期，请重新登录", "error");
  }

  // Fetch phonetic text only. Word audio always comes from word_clean_tts.
  async function fetchWordPhonetics(word: string): Promise<WordPhonetic | null> {
    const cleanWord = word.trim().toLowerCase().replace(/[^a-z-]/g, "");
    if (!cleanWord) return null;
    try {
      const res = await fetch(`https://api.dictionaryapi.dev/api/v2/entries/en/${cleanWord}`);
      if (!res.ok) {
        throw new Error("Word not found");
      }
      const data = await res.json();
      if (Array.isArray(data) && data.length > 0) {
        const entry = data[0];
        const phoneticText = entry.phonetic || entry.phonetics?.find((p: any) => p.text)?.text || "";
        return {
          word: entry.word || word,
          phonetic: phoneticText,
        };
      }
    } catch {
      // Quietly ignore or fall back
    }
    return null;
  }

  // Load and cache phonetics for a list of words
  async function loadPhoneticsForWords(words: string[]) {
    setFetchingPhonetics(true);
    const updatedPhonetics = { ...phonetics };
    try {
      await Promise.all(
        words.map(async (word) => {
          const lower = word.toLowerCase();
          if (updatedPhonetics[lower]) return;
          const info = await fetchWordPhonetics(word);
          if (info) {
            updatedPhonetics[lower] = info;
          } else {
            updatedPhonetics[lower] = { word, phonetic: "" };
          }
        })
      );
      setPhonetics(updatedPhonetics);
    } catch (e) {
      console.error(e);
    } finally {
      setFetchingPhonetics(false);
    }
  }

  function playWordAudio(wordAudioUrl?: string | null) {
    const source = resolveWordAudioSource(wordAudioUrl);
    if (source.kind === "missing") {
      showToastMessage("暂无原词音频", "info");
      return;
    }
    const audio = new Audio(source.url);
    void audio.play().catch(() => {
      showToastMessage("单词音频播放失败，请确认 MinIO 文件服务已启动", "error");
    });
  }

  function stopSentenceSpeech() {
    if (sentenceAudioRef.current) {
      sentenceAudioRef.current.pause();
      sentenceAudioRef.current.currentTime = 0;
      sentenceAudioRef.current = null;
    }
    if ("speechSynthesis" in window) {
      window.speechSynthesis.cancel();
    }
    sentenceSpeechRef.current = null;
    setIsSpeakingSentence(false);
  }

  function buildSentenceSpeechText(currentTask: ClozePracticeTask, expectedWords?: string[]) {
    const parts = currentTask.clozeSentence.split("____");
    const speechText = parts
      .map((part, index) => {
        if (index >= parts.length - 1) {
          return part;
        }
        return `${part} ${expectedWords?.[index]?.trim() || SENTENCE_BLANK_SPEECH_TEXT} `;
      })
      .join("");
    return speechText.replace(/\s+/g, " ").trim();
  }

  function playSentenceAudio(currentTask: ClozePracticeTask) {
    if (isSpeakingSentence) {
      stopSentenceSpeech();
      return;
    }

    const audioSource = resolveSentenceAudioSource(currentTask.sentenceAudioUrl);
    if (audioSource.kind === "minio") {
      if ("speechSynthesis" in window) {
        window.speechSynthesis.cancel();
      }
      const audio = new Audio(audioSource.url);
      sentenceAudioRef.current = audio;
      const clearAudioState = () => {
        if (sentenceAudioRef.current === audio) {
          sentenceAudioRef.current = null;
          setIsSpeakingSentence(false);
        }
      };
      audio.onended = clearAudioState;
      audio.onerror = () => {
        clearAudioState();
        showToastMessage("句子音频播放失败，请确认文件服务已启动", "error");
      };
      setIsSpeakingSentence(true);
      void audio.play().catch(() => {
        clearAudioState();
        showToastMessage("句子音频播放失败，请稍后重试", "error");
      });
      return;
    }

    if (!("speechSynthesis" in window)) {
      showToastMessage("您的浏览器不支持语音播放", "error");
      return;
    }

    const expectedWords = result?.expectedWords?.length ? result.expectedWords : undefined;
    const speechText = buildSentenceSpeechText(currentTask, expectedWords);
    if (!speechText) {
      return;
    }

    window.speechSynthesis.cancel();
    const utterance = new SpeechSynthesisUtterance(speechText);
    sentenceSpeechRef.current = utterance;
    utterance.lang = "en-US";
    utterance.rate = 0.88;
    utterance.pitch = 1;
    utterance.onend = () => {
      if (sentenceSpeechRef.current === utterance) {
        sentenceSpeechRef.current = null;
        setIsSpeakingSentence(false);
      }
    };
    utterance.onerror = () => {
      if (sentenceSpeechRef.current === utterance) {
        sentenceSpeechRef.current = null;
        setIsSpeakingSentence(false);
      }
    };
    setIsSpeakingSentence(true);
    window.speechSynthesis.speak(utterance);
  }

  // Initialize server-backed preferences and due reviews after authentication.
  useEffect(() => {
    if (!auth) {
      return;
    }
    void initializeSession();
  }, [auth?.token]);

  // Focus the first input on new task
  useEffect(() => {
    answerInputRefs.current = [];
    stopSentenceSpeech();
    if (!task) {
      return;
    }
    window.setTimeout(() => {
      answerInputRefs.current[0]?.focus();
      setActiveInputIndex(0);
    }, 50);
  }, [task?.id]);

  // Audio autoplay on correct answer
  useEffect(() => {
    if (result && result.expectedWords && result.expectedWords.length > 0) {
      const targetWords = result.expectedWords;
      if (result.correct) {
        void loadPhoneticsForWords(targetWords);
        if (task?.wordAudioUrl) {
          playWordAudio(task.wordAudioUrl);
        }
      } else {
        void loadPhoneticsForWords(targetWords);
      }
    }
  }, [result?.clozeItemId, result?.correct, task?.wordAudioUrl]);

  async function initializeSession() {
    if (!token) {
      return;
    }
    setLoading(true);
    setMessage("");
    try {
      const [preference, reviews, nextStats] = await Promise.all([
        getClozePreferences(token),
        getDueReviewTasks(token, DIFFICULTY_BATCH_SIZE),
        getStats(token),
      ]);
      const nextDifficulty = normalizeSoloDifficulty(preference);
      setSelectedDifficulty(nextDifficulty);
      setExpandedDifficultyKey(nextDifficulty.parentKey);
      setDueReviewTasks(reviews);
      setTask(null);
      setBatchTasks([]);
      setBatchIndex(0);
      setIsPracticeActive(false);
      setAnswers([""]);
      setStats(nextStats);
      setResult(null);
      setActiveInputIndex(0);
      setStartedAt(Date.now());
    } catch (error) {
      if (isAuthError(error)) {
        handleAuthExpired();
        return;
      }
      setMessage(error instanceof Error ? error.message : "加载练习数据失败");
    } finally {
      setLoading(false);
    }
  }

  async function refreshDueReviewTasks() {
    if (!token) return [];
    const reviews = await getDueReviewTasks(token, DIFFICULTY_BATCH_SIZE);
    setDueReviewTasks(reviews);
    return reviews;
  }

  function activateTask(nextTask: ClozePracticeTask, nextIndex: number, nextBatch: ClozePracticeTask[]) {
    setTask(nextTask);
    setBatchTasks(nextBatch);
    setBatchIndex(nextIndex);
    setAnswers(emptyAnswers(nextTask.blankCount));
    setResult(null);
    setActiveInputIndex(0);
    setStartedAt(Date.now());
    window.setTimeout(() => {
      answerInputRefs.current[0]?.focus();
      setActiveInputIndex(0);
    }, 80);
  }

  async function saveSelectedDifficulty(nextDifficulty: SelectedDifficulty) {
    if (!token) return;
    setLoading(true);
    try {
      const preference = await updateSoloDifficulty(token, nextDifficulty.parentKey, nextDifficulty.key);
      setSelectedDifficulty(normalizeSoloDifficulty(preference));
      setExpandedDifficultyKey(nextDifficulty.parentKey);
      setShowDifficultyPicker(false);
      setBatchTasks([]);
      setBatchIndex(0);
    } catch (error) {
      if (isAuthError(error)) {
        handleAuthExpired();
        return;
      }
      showToastMessage(error instanceof Error ? error.message : "保存训练难度失败", "error");
    } finally {
      setLoading(false);
    }
  }

  function handleSelectParentDifficulty(parent: DifficultyParentOption) {
    const nextDifficulty: SelectedDifficulty = {
      key: parent.key,
      parentKey: parent.key,
      title: parent.title,
    };
    void saveSelectedDifficulty(nextDifficulty);
  }

  function handleSelectChildDifficulty(parent: DifficultyParentOption, child: DifficultyChildOption) {
    const nextDifficulty: SelectedDifficulty = {
      key: child.key,
      parentKey: parent.key,
      title: child.title,
      avgDifficulty: child.avgDifficulty,
    };
    void saveSelectedDifficulty(nextDifficulty);
  }

  function openDifficultyPicker() {
    setExpandedDifficultyKey(selectedDifficulty.parentKey);
    setShowDifficultyPicker(true);
  }

  function closePractice() {
    setIsPracticeActive(false);
    showToastMessage("已退出全屏练习模式", "info");
  }

  async function loadDifficultyBatch(activate = false) {
    if (!token) {
      return;
    }
    setLoading(true);
    setMessage("");
    try {
      const [tasks, nextStats] = await Promise.all([
        createDifficultyBatch(token, {
          difficultyGroup: selectedDifficulty.parentKey,
          difficultyLevel: selectedDifficulty.key,
          limit: DIFFICULTY_BATCH_SIZE,
        }),
        getStats(token),
      ]);
      setStats(nextStats);
      if (!tasks.length) {
        setTask(null);
        setBatchTasks([]);
        setBatchIndex(0);
        setIsPracticeActive(false);
        setResult(null);
        showToastMessage("当前难度暂无可练句子", "error");
        return;
      }
      activateTask(tasks[0], 0, tasks);
      if (activate) {
        setPracticeSource("solo");
        setIsPracticeActive(true);
      }
    } catch (error) {
      if (isAuthError(error)) {
        handleAuthExpired();
        return;
      }
      setMessage(error instanceof Error ? error.message : "加载难度练习失败");
    } finally {
      setLoading(false);
    }
  }

  async function startDifficultyPractice() {
    await loadDifficultyBatch(true);
  }

  async function startReviewPractice() {
    if (!token) return;
    setLoading(true);
    setMessage("");
    try {
      const reviews = await refreshDueReviewTasks();
      if (!reviews.length) {
        showToastMessage("今天没有到期错题", "info");
        return;
      }
      setPracticeSource("review");
      activateTask(reviews[0], 0, reviews);
      setIsPracticeActive(true);
    } catch (error) {
      if (isAuthError(error)) {
        handleAuthExpired();
        return;
      }
      setMessage(error instanceof Error ? error.message : "加载错题复习失败");
    } finally {
      setLoading(false);
    }
  }

  async function loadSentenceTasks(mode: SentenceListMode, openPanel = false) {
    if (!token) {
      return;
    }
    setPendingTasksLoading(true);
    setSentenceListMode(mode);
    setMessage("");
    try {
      const tasks = mode === "pending" ? await getPendingTasks(token, 100) : await getAnsweredTasks(token, mode, 100);
      setPendingTasks(tasks);
      if (openPanel) {
        setShowPendingTasks(true);
      }
    } catch (error) {
      if (isAuthError(error)) {
        handleAuthExpired();
        return;
      }
      setMessage(error instanceof Error ? error.message : "加载句子列表失败");
    } finally {
      setPendingTasksLoading(false);
    }
  }

  async function loadPendingTasks(openPanel = false) {
    await loadSentenceTasks("pending", openPanel);
  }

  async function loadAnswerResults(openPanel = false) {
    if (!token) {
      return;
    }
    setAnswerResultsLoading(true);
    setMessage("");
    try {
      const historyItems = await getHistory(token, 100);
      const rounds = groupClozeResultRounds(historyItems);
      setAnswerResultRounds(rounds);
      setExpandedResultRounds(rounds.length > 0 ? new Set([rounds[0].id]) : new Set());
      if (openPanel) {
        setShowAnswerResults(true);
      }
    } catch (error) {
      if (isAuthError(error)) {
        handleAuthExpired();
        return;
      }
      setMessage(error instanceof Error ? error.message : "加载答题结果失败");
    } finally {
      setAnswerResultsLoading(false);
    }
  }

  function toggleResultRound(roundId: string) {
    setExpandedResultRounds((current) => {
      const next = new Set(current);
      if (next.has(roundId)) {
        next.delete(roundId);
      } else {
        next.add(roundId);
      }
      return next;
    });
  }

  async function goNextPracticeTask() {
    if (batchTasks.length > 0 && batchIndex + 1 < batchTasks.length) {
      activateTask(batchTasks[batchIndex + 1], batchIndex + 1, batchTasks);
      return;
    }
    const nextMode = nextLauncherModeAfterBatch(practiceSource);
    setLauncherMode(nextMode);
    setIsPracticeActive(false);
    setTask(null);
    setBatchTasks([]);
    setBatchIndex(0);
    setResult(null);
    if (practiceSource === "review") {
      try {
        await refreshDueReviewTasks();
      } catch (error) {
        if (isAuthError(error)) {
          handleAuthExpired();
          return;
        }
        setMessage(error instanceof Error ? error.message : "刷新错题复习失败");
      }
    }
  }

  async function handleAuthSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setMessage("");
    try {
      const response = authMode === "login" ? await login(credentials) : await register(credentials);
      persistAuth(response);
      setAuth(response);
      showToastMessage("欢迎回来！", "success");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "认证失败");
    } finally {
      setLoading(false);
    }
  }

  function handleLogout() {
    clearStoredAuth();
    setAuth(null);
    resetPracticeState();
    setMessage("");
    showToastMessage("已安全退出", "info");
  }

  async function handleSubmitAnswer(event?: FormEvent<HTMLFormElement>) {
    if (event) {
      event.preventDefault();
    }
    if (!task || !token) {
      return;
    }
    setLoading(true);
    setMessage("");
    const submittedAnswers = answers.map((answer) => answer.trim());
    const identity: SubmissionIdentity = {
      taskId: task.id,
      actionType: "answer",
      practiceSource,
      answers: submittedAnswers,
    };
    const pendingSubmission = pendingSubmissionRef.current;
    const submission = pendingSubmission && shouldReuseSubmission(pendingSubmission, identity)
      ? pendingSubmission
      : { ...identity, key: createSubmissionKey() };
    pendingSubmissionRef.current = submission;
    try {
      const response = await submitAnswer(token, {
        clozeItemId: task.id,
        answers: submittedAnswers,
        answerText: answers.join(", "),
        costMs: Date.now() - startedAt,
        submissionKey: submission.key,
        practiceContext: practiceSource,
        actionType: "answer",
      });
      pendingSubmissionRef.current = null;
      setResult(response);
      const nextStats = await getStats(token);
      setStats(nextStats);
      if (response.correct) {
        showToastMessage("回答正确！🎉", "success");
      } else {
        showToastMessage("拼写有误，再试一次吧", "error");
      }
    } catch (error) {
      if (isAuthError(error)) {
        handleAuthExpired();
        return;
      }
      setMessage(error instanceof Error ? error.message : "提交答案失败");
    } finally {
      setLoading(false);
    }
  }

  // Force reveal the answer by submitting placeholder text
  async function handleRevealAnswer() {
    if (!task || !token || loading) return;
    setLoading(true);
    setMessage("");
    const revealedAnswers = emptyAnswers(task.blankCount).map(() => "?");
    const identity: SubmissionIdentity = {
      taskId: task.id,
      actionType: "reveal",
      practiceSource,
      answers: revealedAnswers,
    };
    const pendingSubmission = pendingSubmissionRef.current;
    const submission = pendingSubmission && shouldReuseSubmission(pendingSubmission, identity)
      ? pendingSubmission
      : { ...identity, key: createSubmissionKey() };
    pendingSubmissionRef.current = submission;
    try {
      const response = await submitAnswer(token, {
        clozeItemId: task.id,
        answers: revealedAnswers,
        answerText: "?",
        costMs: Date.now() - startedAt,
        submissionKey: submission.key,
        practiceContext: practiceSource,
        actionType: "reveal",
      });
      pendingSubmissionRef.current = null;
      setResult(response);
      const nextStats = await getStats(token);
      setStats(nextStats);
      showToastMessage("答案已显示", "info");
    } catch (error) {
      if (isAuthError(error)) {
        handleAuthExpired();
        return;
      }
      setMessage(error instanceof Error ? error.message : "获取答案失败");
    } finally {
      setLoading(false);
    }
  }

  function updateAnswer(index: number, value: string) {
    // Only allow modification when not answered correctly yet
    if (result?.correct) return;
    setAnswers((current) => current.map((answer, answerIndex) => (answerIndex === index ? value : answer)));
  }

  function focusAnswer(index: number) {
    window.requestAnimationFrame(() => {
      answerInputRefs.current[index]?.focus();
      setActiveInputIndex(index);
    });
  }

  function handleInlineAnswerKeyDown(index: number, event: KeyboardEvent<HTMLInputElement>) {
    const shouldMoveBySpace = event.key === " " && answers[index]?.trim();
    const shouldMove = event.key === "Enter" || event.key === "Tab" || shouldMoveBySpace;
    if (!shouldMove) {
      return;
    }

    event.preventDefault();
    const nextIndex = index + 1;
    if (nextIndex < answers.length) {
      focusAnswer(nextIndex);
      return;
    }

    if (canSubmit && !result?.correct) {
      void handleSubmitAnswer();
    }
  }

  // Keyboard hotkeys handler
  useEffect(() => {
    const handleGlobalKeyDown = (event: globalThis.KeyboardEvent) => {
      if (!auth) return;

      if (event.key === "Escape") {
        event.preventDefault();
        if (isPracticeActive) {
          closePractice();
        } else if (showAnswerResults) {
          setShowAnswerResults(false);
        } else if (showPendingTasks) {
          setShowPendingTasks(false);
        } else if (showDifficultyPicker) {
          setShowDifficultyPicker(false);
        } else if (showWrongSentences) {
          setShowWrongSentences(false);
        } else if (launcherMode === "solo") {
          setLauncherMode("review");
        }
        return;
      }

      // Restrict active practice hotkeys
      if (!isPracticeActive) return;

      const isCtrl = event.ctrlKey || event.metaKey;

      // Ctrl + L: Read sentence
      if (isCtrl && event.key.toLowerCase() === "l") {
        event.preventDefault();
        if (task) {
          playSentenceAudio(task);
        }
        return;
      }

      // Ctrl + P: Play the generated base-word TTS after the answer is known.
      if (isCtrl && event.key.toLowerCase() === "p") {
        event.preventDefault();
        if (result && task?.wordAudioUrl) {
          playWordAudio(task.wordAudioUrl);
        } else {
          showToastMessage("暂无原词音频", "info");
        }
        return;
      }

      // Ctrl + M: Mastered
      if (isCtrl && event.key.toLowerCase() === "m") {
        event.preventDefault();
        showToastMessage("标记已掌握", "success");
        void goNextPracticeTask();
        return;
      }

      // Ctrl + . : Show answer
      if (isCtrl && event.key === ".") {
        event.preventDefault();
        if (!result && task) {
          void handleRevealAnswer();
        }
        return;
      }

      // Enter outside fields when answered
      if (event.key === "Enter" && result) {
        event.preventDefault();
        if (result.correct) {
          void goNextPracticeTask();
        } else {
          // Clear result so they can retry
          setResult(null);
          window.setTimeout(() => {
            answerInputRefs.current[0]?.focus();
            setActiveInputIndex(0);
          }, 50);
        }
      }
    };

    window.addEventListener("keydown", handleGlobalKeyDown);
    return () => {
      window.removeEventListener("keydown", handleGlobalKeyDown);
    };
  }, [auth, task, result, answers, activeInputIndex, phonetics, token, isPracticeActive, isSpeakingSentence, batchTasks, batchIndex, selectedDifficulty, showAnswerResults, showPendingTasks, showDifficultyPicker, showWrongSentences, launcherMode]);

  function answerResultClass(index: number) {
    if (!result) {
      return "";
    }
    return normalizeAnswer(result.expectedWords[index]) === normalizeAnswer(answers[index]) ? "correct" : "wrong";
  }

  function getInlineAnswerStyle(currentTask: ClozePracticeTask, index: number): InlineAnswerStyle {
    const targetLength = currentTask.blankLengths?.[index];
    const hasTargetLength = typeof targetLength === "number" && Number.isFinite(targetLength);
    const typedLength = Array.from((answers[index] || "").trim()).length;
    const displayLength = Math.max(
      hasTargetLength ? targetLength : FALLBACK_BLANK_LENGTH,
      typedLength,
      MIN_BLANK_LENGTH,
    );
    return { "--inline-answer-width": `${displayLength}ch` };
  }

  function renderInlineSentence(currentTask: ClozePracticeTask) {
    const parts = currentTask.clozeSentence.split("____");
    return parts.map((part, index) => (
      <Fragment key={`${currentTask.id}-${index}`}>
        <span className="sentence-part-text">{part}</span>
        {index < answers.length ? (
          <input
            ref={(element) => {
              answerInputRefs.current[index] = element;
            }}
            className={`inline-answer ${answers[index]?.trim() ? "filled" : ""} ${answerResultClass(index)} ${activeInputIndex === index ? "active" : ""}`}
            style={getInlineAnswerStyle(currentTask, index)}
            value={answers[index] || ""}
            onChange={(event) => updateAnswer(index, event.target.value)}
            onKeyDown={(event) => handleInlineAnswerKeyDown(index, event)}
            onFocus={() => setActiveInputIndex(index)}
            onBlur={() => {
              if (activeInputIndex === index) {
                setActiveInputIndex(null);
              }
            }}
            aria-label={`空位 ${index + 1}`}
            title="Enter / Tab / 空格切换"
            autoComplete="off"
            disabled={loading || Boolean(result?.correct)}
          />
        ) : null}
      </Fragment>
    ));
  }

  return (
    <main className="app-shell">
      {/* Toast Notification */}
      {toast && (
        <div className={`toast-message ${toast.type}`}>
          <span className="toast-icon">
            {toast.type === "success" && "✓"}
            {toast.type === "error" && "✕"}
            {toast.type === "info" && "ℹ"}
          </span>
          <span className="toast-text">{toast.message}</span>
        </div>
      )}

      {/* Sidebar Overlay Trigger for Mobile */}
      <button
        className="sidebar-toggle-btn"
        type="button"
        onClick={() => setIsSidebarOpen(!isSidebarOpen)}
        title="切换菜单"
      >
        {isSidebarOpen ? "✕" : "☰"}
      </button>

      <aside className={`side-panel ${isSidebarOpen ? "open" : ""}`}>
        <div className="brand-row">
          <div className="brand-mark">CW</div>
          <div>
            <h1>挖空练习</h1>
            <p>根据翻译完成目标单词</p>
          </div>
        </div>

        {auth ? (
          <section className="account-panel">
            <div>
              <span className="muted-label">当前用户</span>
              <strong>{displayName}</strong>
            </div>
            <button className="ghost-button" type="button" onClick={handleLogout}>
              退出登录
            </button>
          </section>
        ) : (
          <form className="auth-panel" onSubmit={handleAuthSubmit}>
            <div className="segmented-control" role="tablist" aria-label="认证方式">
              <button
                type="button"
                className={authMode === "login" ? "active" : ""}
                onClick={() => setAuthMode("login")}
              >
                登录
              </button>
              <button
                type="button"
                className={authMode === "register" ? "active" : ""}
                onClick={() => setAuthMode("register")}
              >
                注册
              </button>
            </div>
            <label>
              用户名
              <input
                autoComplete="username"
                value={credentials.username}
                onChange={(event) => setCredentials((value) => ({ ...value, username: event.target.value }))}
                required
              />
            </label>
            {authMode === "register" ? (
              <label>
                昵称
                <input
                  value={credentials.nickname}
                  onChange={(event) => setCredentials((value) => ({ ...value, nickname: event.target.value }))}
                  required
                />
              </label>
            ) : null}
            <label>
              密码
              <input
                type="password"
                autoComplete={authMode === "login" ? "current-password" : "new-password"}
                value={credentials.password}
                onChange={(event) => setCredentials((value) => ({ ...value, password: event.target.value }))}
                required
              />
            </label>
            <button className="primary-button submit-auth-btn" disabled={loading} type="submit">
              {authMode === "login" ? "确认登录" : "注册并登录"}
            </button>
          </form>
        )}

        <section className="stats-grid">
          <div>
            <span>待练</span>
            <strong>{stats?.pendingTasks ?? 0}</strong>
          </div>
          <div>
            <span>已完成</span>
            <strong>{stats?.completedTasks ?? 0}</strong>
          </div>
          <div>
            <span>正确率</span>
            <strong>{accuracyText}</strong>
          </div>
          <div>
            <span>已答题数</span>
            <strong>{stats?.totalAnswers ?? 0}</strong>
          </div>
        </section>
      </aside>

      <section className="workspace">
        <header className="toolbar">
          <div>
            <span className="muted-label">CLOZE PRACTICE</span>
            <h2>{launcherMode === "review" ? "错题挖空复习" : "单独训练"}</h2>
          </div>
        </header>

        {message ? <div className="notice error">{message}</div> : null}

        {!auth ? (
          <section className="empty-state welcome-card">
            <div className="welcome-icon">✍️</div>
            <h2>登录以开始练习</h2>
            <p>系统将基于您的答题错词，智能定制每日的挖空句型演练。</p>
          </section>
        ) : launcherMode === "review" ? (
          <ReviewPracticeLauncher
            dueCount={stats?.dueReviewTasks ?? dueReviewTasks.length}
            wrongCount={stats?.activeWrongSentences ?? 0}
            loading={loading}
            onStart={() => void startReviewPractice()}
            onOpenWrongSentences={() => setShowWrongSentences(true)}
            onOpenSolo={() => setLauncherMode("solo")}
          />
        ) : (
          <SoloTrainingLauncher
            selectedLabel={selectedDifficultyText}
            batchText={batchProgressText}
            loading={loading}
            showClose={!showDifficultyPicker && !showPendingTasks && !showAnswerResults && !showWrongSentences && !isPracticeActive}
            onClose={() => setLauncherMode("review")}
            onChooseDifficulty={openDifficultyPicker}
            onOpenSentences={() => void loadPendingTasks(true)}
            onOpenResults={() => void loadAnswerResults(true)}
            onStart={() => void startDifficultyPractice()}
          />
        )}
      </section>

      {showWrongSentences && auth ? (
        <WrongSentenceCollection
          token={token}
          onClose={() => setShowWrongSentences(false)}
          onAuthExpired={handleAuthExpired}
        />
      ) : null}

      {showDifficultyPicker && (
        <div className="difficulty-picker-overlay">
          <FullscreenCloseButton label="关闭难度选择" onClose={() => setShowDifficultyPicker(false)} />
          <div className="difficulty-picker-header">
            <div>
              <h2>选择造句难度</h2>
              <p>{selectedDifficultyText}</p>
            </div>
          </div>

          <div className="difficulty-card-grid">
            {difficultyOptions.map((option) => {
              const expanded = expandedDifficultyKey === option.key;
              const parentSelected = selectedDifficulty.parentKey === option.key;
              const exactParentSelected = selectedDifficulty.key === option.key;
              return (
                <section className={`difficulty-card ${expanded ? "expanded" : ""} ${parentSelected ? "active" : ""}`} key={option.key}>
                  <div
                    className="difficulty-card-title"
                  >
                    <button
                      className="difficulty-toggle-button"
                      type="button"
                      onClick={() => setExpandedDifficultyKey(expanded ? null : option.key)}
                      aria-label={`${expanded ? "收起" : "展开"}${option.title}`}
                    >
                      {expanded ? "⌄" : "›"}
                    </button>
                    <button
                      className={`difficulty-parent-button ${exactParentSelected ? "active" : ""}`}
                      type="button"
                      onClick={() => handleSelectParentDifficulty(option)}
                    >
                      <span>{option.title}</span>
                      <small>全部随机</small>
                    </button>
                  </div>
                  {expanded ? (
                    <div className="difficulty-child-grid">
                      {option.children.map((child) => (
                        <button
                          className={`difficulty-child-card ${selectedDifficulty.key === child.key ? "active" : ""}`}
                          key={child.key}
                          type="button"
                          onClick={() => handleSelectChildDifficulty(option, child)}
                        >
                          <strong>{child.title}</strong>
                          <span>平均难度 {child.avgDifficulty}</span>
                        </button>
                      ))}
                    </div>
                  ) : null}
                </section>
              );
            })}
          </div>
        </div>
      )}

      {showPendingTasks && (
        <div className="pending-tasks-overlay">
          <FullscreenCloseButton label="关闭句子列表" onClose={() => setShowPendingTasks(false)} />
          <div className="pending-tasks-header">
            <span aria-hidden="true" />
            <div>
              <span className="launcher-kicker">{activeSentenceListMeta.kicker}</span>
              <h2>{activeSentenceListMeta.title}</h2>
              <p>{activeSentenceListMeta.subtitle}</p>
            </div>
            <button className="ghost-button" type="button" onClick={() => loadSentenceTasks(sentenceListMode)} disabled={pendingTasksLoading}>
              刷新
            </button>
          </div>

          <div className="sentence-list-tabs" role="tablist" aria-label="句子类型">
            {sentenceListTabs.map((tab) => (
              <button
                className={`sentence-list-tab ${sentenceListMode === tab.mode ? "active" : ""}`}
                key={tab.mode}
                type="button"
                role="tab"
                aria-selected={sentenceListMode === tab.mode}
                onClick={() => loadSentenceTasks(tab.mode)}
                disabled={pendingTasksLoading && sentenceListMode === tab.mode}
              >
                <span>{tab.label}</span>
                {typeof tab.count === "number" ? <strong>{tab.count}</strong> : null}
              </button>
            ))}
          </div>

          <div className="pending-summary-row">
            <div>
              <span>当前难度</span>
              <strong>{selectedDifficultyText}</strong>
            </div>
            <div>
              <span>{activeSentenceListMeta.countLabel}</span>
              <strong>{pendingTasks.length}</strong>
            </div>
            <div>
              <span>当前批次</span>
              <strong>{batchProgressText}</strong>
            </div>
          </div>

          <section className="pending-table-shell">
            {pendingTasks.length === 0 ? (
              <div className="pending-empty-state">
                <strong>{activeSentenceListMeta.emptyTitle}</strong>
                <span>{activeSentenceListMeta.emptyText}</span>
              </div>
            ) : (
              <>
                <div className="pending-table-header">
                  <span>时间</span>
                  <span>单词</span>
                  <span>挖空句</span>
                  <span>翻译</span>
                  <span>{sentenceListTimeHeader(sentenceListMode)}</span>
                </div>
                <div className="pending-table-body">
                  {pendingTasks.map((item) => (
                    <article className="pending-row" key={item.id}>
                      <time>{formatDateTime(item.latestAnswerTime || item.createTime)}</time>
                      <strong className="pending-word">{item.word || "-"}</strong>
                      <p className="pending-sentence">{item.clozeSentence}</p>
                      <p className="pending-translation">{item.translationZh || "-"}</p>
                      <span>{formatReviewAvailability(item, sentenceListMode)}</span>
                    </article>
                  ))}
                </div>
              </>
            )}
          </section>
        </div>
      )}

      {showAnswerResults && (
        <div className="answer-results-overlay">
          <FullscreenCloseButton label="关闭答题结果" onClose={() => setShowAnswerResults(false)} />
          <header className="answer-results-header">
            <span aria-hidden="true" />
            <div>
              <span className="launcher-kicker">ANSWER RESULTS</span>
              <h2>答题结果</h2>
              <p>按最近答题记录自动汇总为练习轮次</p>
            </div>
            <button className="ghost-button" type="button" onClick={() => loadAnswerResults()} disabled={answerResultsLoading}>
              刷新
            </button>
          </header>

          {answerResultRounds.length > 0 ? (
            <section className="answer-results-summary">
              <div>
                <span>练习轮次</span>
                <strong>{resultSummary.rounds}</strong>
              </div>
              <div>
                <span>答题数</span>
                <strong>{resultSummary.total}</strong>
              </div>
              <div>
                <span>正确</span>
                <strong>{resultSummary.correct}</strong>
              </div>
              <div>
                <span>准确率</span>
                <strong>{resultSummary.accuracy}%</strong>
              </div>
              <div>
                <span>总耗时</span>
                <strong>{formatCostMs(resultSummary.costMs)}</strong>
              </div>
            </section>
          ) : null}

          {answerResultsLoading ? (
            <section className="answer-results-state">加载中...</section>
          ) : answerResultRounds.length === 0 ? (
            <section className="answer-results-state">
              <strong>暂无答题结果</strong>
              <span>完成一轮挖空练习后，这里会按轮次展示每次作答情况。</span>
            </section>
          ) : (
            <section className="answer-round-list">
              {answerResultRounds.map((round) => {
                const expanded = expandedResultRounds.has(round.id);
                return (
                  <article
                    className={`answer-round-card ${round.wrongCount === 0 ? "correct" : "wrong"} ${expanded ? "expanded" : ""}`}
                    key={round.id}
                  >
                    <button className="answer-round-summary" type="button" onClick={() => toggleResultRound(round.id)}>
                      <span className="answer-round-index">第 {round.index} 轮</span>
                      <span className="answer-round-title">{formatResultTime(round.startTime)}</span>
                      <span className="answer-metric-cell">句子 {round.details.length}</span>
                      <span className="answer-metric-cell">正确 {round.correctCount}/{round.details.length}</span>
                      <span className="answer-metric-cell">准确率 {round.accuracy}%</span>
                      <span className="answer-metric-cell">耗时 {formatCostMs(round.totalCostMs)}</span>
                      <span className="answer-expand-cell">{expanded ? "收起" : "展开"}</span>
                    </button>

                    {expanded ? (
                      <div className="answer-round-detail">
                        <div className="answer-detail-grid answer-detail-head">
                          <span>序号</span>
                          <span>时间</span>
                          <span>结果</span>
                          <span>尝试</span>
                          <span>耗时</span>
                          <span>你的答案</span>
                          <span>正确答案</span>
                          <span>挖空句</span>
                          <span>翻译</span>
                        </div>
                        {round.details.map((detail, index) => (
                          <div
                            className={`answer-detail-grid answer-detail-row ${detail.isCorrect ? "correct" : "wrong"}`}
                            key={detail.id}
                          >
                            <span>{index + 1}</span>
                            <time>{formatResultTime(detail.createTime)}</time>
                            <span>{detail.isCorrect ? "答对" : "答错"}</span>
                            <span>第 {detail.attemptNo} 次</span>
                            <span>{formatCostMs(detail.costMs)}</span>
                            <span>{detail.answerText || "未作答"}</span>
                            <span>{parseExpectedWords(detail.expectedWordsJson)}</span>
                            <p>{detail.clozeSentence}</p>
                            <p>{detail.translationZh || "-"}</p>
                          </div>
                        ))}
                      </div>
                    ) : null}
                  </article>
                );
              })}
            </section>
          )}
        </div>
      )}

      {/* Immersive Full-screen Practice Overlay */}
      {isPracticeActive && task && (
        <div className="immersive-practice-overlay">
          <FullscreenCloseButton label="退出练习" onClose={closePractice} />

          <div className="immersive-content">
            {/* Display translation in the top center, large and bold, exactly like screenshot 1 */}
            <div className="translation-box">
              <p className="chinese-translation">{task.translationZh}</p>
            </div>

            {/* Answer Correct: display success card replacing typing area */}
            {result && result.correct ? (
              <div className="success-word-card">
                <div className="success-badge">✓ 答题成功</div>
                {result.expectedWords.map((word, idx) => {
                  const phoneticInfo = phonetics[word.toLowerCase()];
                  return (
                    <div key={idx} className="word-detail-container">
                      <h1 className="success-word-title">{word}</h1>
                      {phoneticInfo?.phonetic ? (
                        <span className="success-word-phonetic">{phoneticInfo.phonetic}</span>
                      ) : fetchingPhonetics ? (
                        <span className="success-word-phonetic muted">音标获取中...</span>
                      ) : null}
                      {idx === 0 ? (
                        <WordAudioButton
                          baseWord={task.word || word}
                          displayedWord={word}
                          audioUrl={task.wordAudioUrl}
                          onPlay={playWordAudio}
                        />
                      ) : null}
                    </div>
                  );
                })}

                <div className="sentence-success-display">
                  {task.clozeSentence.split("____").map((part, idx) => (
                    <Fragment key={idx}>
                      <span>{part}</span>
                      {idx < result.expectedWords.length ? (
                        <strong className="success-filled-word">{result.expectedWords[idx]}</strong>
                      ) : null}
                    </Fragment>
                  ))}
                </div>

                <div className="card-actions">
                  <button
                    className={`ghost-button sentence-audio-btn ${isSpeakingSentence ? "active" : ""}`}
                    onClick={() => playSentenceAudio(task)}
                    type="button"
                    title="朗读完整句子 (Ctrl+L)"
                  >
                    🔊 {isSpeakingSentence ? "停止朗读" : "朗读完整句子"}
                  </button>
                  <button className="primary-button next-btn-highlight" onClick={goNextPracticeTask} type="button">
                    下一题 (Enter)
                  </button>
                </div>

                <div className="keyboard-shortcuts-guide">
                  <span className="key-hint"><kbd>Enter</kbd> 下一题</span>
                  <span className="key-hint"><kbd>Ctrl</kbd>+<kbd>P</kbd> 发音</span>
                  <span className="key-hint"><kbd>Ctrl</kbd>+<kbd>L</kbd> 句子</span>
                  <span className="key-hint"><kbd>Ctrl</kbd>+<kbd>M</kbd> 掌握</span>
                </div>
              </div>
            ) : (
              /* Normal mode or incorrect mode (input box remains editable) */
              <form
                className="answer-form"
                ref={practiceFormRef}
                onSubmit={(e) => {
                  e.preventDefault();
                  if (canSubmit) void handleSubmitAnswer();
                }}
              >
                {/* Sentence typing area */}
                <div className="sentence-box centered-sentence" aria-label="挖空句子">
                  {renderInlineSentence(task)}
                </div>

                <div className="action-row text-center">
                  <button
                    className={`ghost-button sentence-audio-btn ${isSpeakingSentence ? "active" : ""}`}
                    type="button"
                    onClick={() => playSentenceAudio(task)}
                    title={result ? "朗读完整句子 (Ctrl+L)" : "朗读挖空句子 (Ctrl+L)"}
                  >
                    🔊 {isSpeakingSentence ? "停止朗读" : result ? "朗读完整句子" : "朗读句子"}
                  </button>
                  <button
                    className="primary-button submit-btn"
                    disabled={!canSubmit || loading}
                    type="submit"
                  >
                    {result && !result.correct ? "重新提交" : "提交答案 (Enter)"}
                  </button>
                  <button
                    className="ghost-button reveal-btn"
                    type="button"
                    onClick={handleRevealAnswer}
                    disabled={loading}
                    title="显示答案 (Ctrl+.)"
                  >
                    显示答案
                  </button>
                </div>
              </form>
            )}

            {/* Answer Incorrect: Display failure card below typing area */}
            {result && !result.correct && (
              <div className="failure-word-card">
                <div className="failure-badge">✕ 拼写不正确</div>
                
                <div className="comparison-box">
                  {result.expectedWords.map((word, idx) => {
                    const userTyped = answers[idx] || "";
                    const phoneticInfo = phonetics[word.toLowerCase()];
                    return (
                      <div key={idx} className="comparison-row">
                        <div className="comparison-side wrong">
                          <span className="comp-label">您的输入：</span>
                          <span className="comp-word">{userTyped || "(空)"}</span>
                        </div>
                        <div className="comparison-side correct">
                          <span className="comp-label">正确答案：</span>
                          <span className="comp-word">{word}</span>
                          {phoneticInfo?.phonetic ? (
                            <span className="comp-phonetic">{phoneticInfo.phonetic}</span>
                          ) : fetchingPhonetics ? (
                            <span className="comp-phonetic muted">...</span>
                          ) : null}
                          {idx === 0 ? (
                            <WordAudioButton
                              baseWord={task.word || word}
                              displayedWord={word}
                              audioUrl={task.wordAudioUrl}
                              onPlay={playWordAudio}
                              compact
                            />
                          ) : null}
                        </div>
                      </div>
                    );
                  })}
                </div>

                <div className="card-actions-failure">
                  <button
                    className="primary-button retry-btn"
                    onClick={() => {
                      setResult(null);
                      window.setTimeout(() => {
                        answerInputRefs.current[0]?.focus();
                        setActiveInputIndex(0);
                      }, 50);
                    }}
                    type="button"
                  >
                    修改拼写
                  </button>
                  <button className="ghost-button skip-btn" onClick={goNextPracticeTask} type="button">
                    跳过此题 (Ctrl+M)
                  </button>
                </div>

                <div className="keyboard-shortcuts-guide">
                  <span className="key-hint"><kbd>Enter</kbd> 重新输入</span>
                  <span className="key-hint"><kbd>Ctrl</kbd>+<kbd>.</kbd> 答案详情</span>
                  <span className="key-hint"><kbd>Ctrl</kbd>+<kbd>P</kbd> 发音</span>
                  <span className="key-hint"><kbd>Ctrl</kbd>+<kbd>L</kbd> 句子</span>
                </div>
              </div>
            )}

            <div className="task-sub-meta">
              <span>空位: {task.blankCount}</span>
              <span>已尝试: {task.attemptCount}</span>
              <span>错误数: {task.wrongCount}</span>
              <span>难度: {task.difficultyLabel || selectedDifficultyText}</span>
              <span>本轮: {batchProgressText}</span>
              <span>创建时间: {formatTime(task.createTime)}</span>
            </div>
          </div>
        </div>
      )}
    </main>
  );
}
