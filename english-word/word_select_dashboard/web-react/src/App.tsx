import {
  AudioOutlined,
  BookOutlined,
  CheckCircleOutlined,
  CloudSyncOutlined,
  DatabaseOutlined,
  DeleteOutlined,
  DownOutlined,
  EditOutlined,
  HistoryOutlined,
  LockOutlined,
  LoginOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  PlusOutlined,
  RightOutlined,
  ReloadOutlined,
  SaveOutlined,
  SearchOutlined,
  SendOutlined,
  SettingOutlined,
  TableOutlined,
  CaretDownOutlined,
  CaretUpOutlined,
  UserOutlined,
} from "@ant-design/icons";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button, ConfigProvider, Input, InputNumber, Modal, Pagination, Select, Spin, Switch, Tooltip, message, theme } from "antd";
import { useEffect, useMemo, useRef, useState } from "react";
import ClozeResultTable from "./components/ClozeResultTable";
import ExecutionConfigPage from "./components/ExecutionConfigPage";
import { createDefaultTTSProvider, validateTTSConfig } from "./features/ttsConfig";
import {
  AUTH_REQUIRED_EVENT,
  LoginAttemptCancelledError,
  authEvents,
  cancelPendingLoginAttempt,
  clearAuthSession,
  getAuthToken,
  login,
} from "./lib/auth";
import { listClozeResultItems, listClozeResultUsers } from "./lib/clozeResultApi";
import { formatDuration, formatTime } from "./lib/format";
import { SentenceApiError, generateSentence, listSentenceHistory } from "./lib/sentenceApi";
import { getTTSConfig, saveTTSConfig } from "./lib/ttsConfigApi";
import { listUserClozeWrongWords, listUserMasteredWords, listUserTrainingResults, listUserWrongWords, listUsers } from "./lib/userApi";
import {
  listWordCleanSentences,
  listWordCleanWords,
  listWordLibraries,
  listWordLibraryWords,
  scoreWordCleanSentences,
} from "./lib/wordLibraryApi";
import { listWorkflowRuns } from "./lib/workflowRepository";
import { playableBestSentenceAudioURL, playableTTSAudioURL } from "./utils/wordAudio";
import type { ClozeResultItem, ClozeResultUserSummary } from "./types/clozeResult";
import type { GenerateSentenceResponse, SentenceHistoryItem, PaginatedResult } from "./types/sentence";
import type { TTSConfig, TTSProviderConfig } from "./types/ttsConfig";
import type {
  AppUserItem,
  UserClozeWrongItem,
  UserMasteredWordItem,
  UserTrainingAnswerDetail,
  UserTrainingRound,
  UserWrongWordHistory,
  UserWrongWordItem,
} from "./types/user";
import type {
  WordCleanItem,
  WordCleanSentenceItem,
  WordLibraryItem,
  WordLibraryWordItem,
} from "./types/wordLibrary";
import type { RunStatus, WorkflowRun } from "./types/workflow";

type PageKey =
  | "runs"
  | "sentence"
  | "cloze-results"
  | "ai-config"
  | "tts-config"
  | "users"
  | "user-wrong-words"
  | "user-cloze-wrong-words"
  | "user-mastered-words"
  | "word-library"
  | "word-clean";
type WordSortBy = "difficulty" | "frequency";
type WordCleanSortBy = WordSortBy | "pepDifficulty" | "sourceDifficulty";
type SortOrder = "asc" | "desc";
type CleanWordAudioTarget = `word:${number}` | `sentence:${number}`;

const DEFAULT_PAGE_SIZE = 20;
const PAGE_SIZE_OPTIONS = ["20", "50", "100", "500"];
const WORD_CLEAN_DIFFICULTY_RANGES = [
  { value: "0-99", label: "0-99", min: 0, max: 99, count: 623 },
  { value: "100-199", label: "100-199", min: 100, max: 199, count: 1894 },
  { value: "200-249", label: "200-249", min: 200, max: 249, count: 803 },
  { value: "250-299", label: "250-299", min: 250, max: 299, count: 1855 },
  { value: "300-399", label: "300-399", min: 300, max: 399, count: 1769 },
  { value: "400-449", label: "400-449", min: 400, max: 449, count: 1213 },
  { value: "450-499", label: "450-499", min: 450, max: 499, count: 1779 },
  { value: "500-549", label: "500-549", min: 500, max: 549, count: 2237 },
  { value: "550-599", label: "550-599", min: 550, max: 599, count: 2651 },
  { value: "600-649", label: "600-649", min: 600, max: 649, count: 2071 },
  { value: "650-699", label: "650-699", min: 650, max: 699, count: 1692 },
  { value: "700-749", label: "700-749", min: 700, max: 749, count: 1383 },
  { value: "750-799", label: "750-799", min: 750, max: 799, count: 1079 },
  { value: "800-899", label: "800-899", min: 800, max: 899, count: 1157 },
  { value: "900-999", label: "900-999", min: 900, max: 999, count: 209 },
];

const TRAINING_DIFFICULTY_GROUP_LABELS: Record<string, string> = {
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

const TRAINING_DIFFICULTY_LEVEL_LABELS: Record<string, string> = {
  rank_current: "段位难度",
  primary_3_1: "3年级上册",
  primary_3_2: "3年级下册",
  primary_4_1: "4年级上册",
  primary_4_2: "4年级下册",
  primary_5_1: "5年级上册",
  primary_5_2: "5年级下册",
  primary_6_1: "6年级上册",
  primary_6_2: "6年级下册",
  junior_7_1: "7年级上册",
  junior_7_2: "7年级下册",
  junior_8_1: "8年级上册",
  junior_8_2: "8年级下册",
  junior_9_1: "9年级上册",
  senior_1: "上册",
  senior_2: "下册",
  senior_3: "第3册",
  senior_4: "第4册",
  senior_5: "第5册",
  senior_6: "第6册",
  senior_7: "第7册",
  senior_8: "第8册",
  senior_9: "第9册",
  senior_10: "第10册",
  senior_11: "第11册",
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

interface WordLibraryGroup {
  key: string;
  title: string;
  firstID: number;
  totalWordCount: number;
  libraries: WordLibraryItem[];
}

interface AppPaginationProps {
  current: number;
  pageSize: number;
  total: number;
  totalLabel: string;
  className?: string;
  onChange: (page: number, pageSize: number) => void;
}

function AppPagination({ current, pageSize, total, totalLabel, className, onChange }: AppPaginationProps) {
  if (total <= 0) {
    return null;
  }

  return (
    <Pagination
      className={className}
      current={current}
      pageSize={pageSize}
      total={total}
      onChange={onChange}
      showSizeChanger
      showQuickJumper
      pageSizeOptions={PAGE_SIZE_OPTIONS}
      showTotal={(nextTotal) => `共 ${nextTotal} ${totalLabel}`}
    />
  );
}

const PEP_DIFFICULTY_GROUPS = [
  {
    title: "小学英语",
    items: [
      { value: 1, label: "3年级上册", count: 63 },
      { value: 2, label: "3年级下册", count: 70 },
      { value: 3, label: "4年级上册", count: 74 },
      { value: 4, label: "4年级下册", count: 78 },
      { value: 5, label: "5年级上册", count: 108 },
      { value: 6, label: "5年级下册", count: 121 },
      { value: 7, label: "6年级上册", count: 105 },
      { value: 8, label: "6年级下册", count: 90 },
    ],
  },
  {
    title: "初中英语",
    items: [
      { value: 9, label: "7年级上册", count: 136 },
      { value: 10, label: "7年级下册", count: 192 },
      { value: 11, label: "8年级上册", count: 275 },
      { value: 12, label: "8年级下册", count: 322 },
      { value: 13, label: "9年级上册", count: 421 },
    ],
  },
  {
    title: "高中英语",
    items: [
      { value: 14, label: "上册", count: 203 },
      { value: 15, label: "下册", count: 195 },
      { value: 16, label: "第3册", count: 250 },
      { value: 17, label: "第4册", count: 218 },
      { value: 18, label: "第5册", count: 275 },
      { value: 19, label: "第6册", count: 314 },
      { value: 20, label: "第7册", count: 293 },
      { value: 21, label: "第8册", count: 332 },
      { value: 22, label: "第9册", count: 175 },
      { value: 23, label: "第10册", count: 182 },
      { value: 24, label: "第11册", count: 197 },
    ],
  },
];

const WORD_CLEAN_EXTRA_GROUPS = [
  {
    title: "大学英语",
    items: [
      { value: "cet4", label: "四级", count: 1638 },
      { value: "cet6", label: "六级", count: 892 },
    ],
  },
  {
    title: "升学考试英语",
    items: [
      { value: "kaoyan", label: "考研", count: 1035 },
    ],
  },
  {
    title: "商务与出国英语",
    items: [
      { value: "bec", label: "BEC", count: 850 },
      { value: "ielts", label: "雅思", count: 1265 },
      { value: "toefl", label: "托福", count: 2689 },
      { value: "gmat", label: "GMAT", count: 475 },
    ],
  },
  {
    title: "专业英语",
    items: [
      { value: "tem4", label: "专四级", count: 1022 },
      { value: "tem8", label: "专八级", count: 3483 },
    ],
  },
  {
    title: "高阶考试英语",
    items: [
      { value: "gre", label: "GRE", count: 2966 },
      { value: "sat", label: "SAT", count: 1106 },
    ],
  },
  {
    title: "其他来源",
    items: [
      { value: "other", label: "其他词库", count: 305 },
    ],
  },
];

const statusLabels: Record<RunStatus, string> = {
  pending: "等待中",
  running: "运行中",
  success: "成功",
  failed: "失败",
  skipped: "跳过",
  retrying: "重试中",
};

const statusClass: Record<RunStatus, string> = {
  pending: "status-pending",
  running: "status-running",
  success: "status-success",
  failed: "status-failed",
  skipped: "status-skipped",
  retrying: "status-retrying",
};

const businessTypeLabels: Record<string, string> = {
  sentence_generation: "单词造句",
};

const stepNameLabels: Record<string, string> = {
  receive_request: "接收任务",
  execute_business: "执行业务",
  save_result: "保存记录",
};

function businessTypeName(type: string) {
  return businessTypeLabels[type] ?? type ?? "-";
}

function currentStepName(run: WorkflowRun) {
  return run.steps.find((step) => step.id === run.currentStepId)?.name ?? stepNameLabels[run.currentStepId] ?? run.currentStepId ?? "-";
}

function runDuration(run: WorkflowRun) {
  if (run.durationMs !== undefined && run.durationMs >= 0) {
    return formatDuration(run.durationMs);
  }

  if (!run.finishedAt) {
    const firstRunning = run.steps.find((step) => step.status === "running");
    return formatDuration(firstRunning?.durationMs);
  }

  return formatDuration(new Date(run.finishedAt).getTime() - new Date(run.startedAt).getTime());
}

function TrackerTable({ runs }: { runs: WorkflowRun[] }) {
  return (
    <div className="table-scroll">
      <table className="data-table">
        <thead>
          <tr>
            <th>
              <span>执行日期</span>
              <em>Date</em>
            </th>
            <th>
              <span>Run ID</span>
              <em>String</em>
            </th>
            <th>
              <span>业务类型</span>
              <em>String</em>
            </th>
            <th>
              <span>业务名称</span>
              <em>String</em>
            </th>
            <th>
              <span>当前步骤</span>
              <em>String</em>
            </th>
            <th>
              <span>状态</span>
              <em>Enum</em>
            </th>
            <th>
              <span>事件数</span>
              <em>Integer</em>
            </th>
            <th>
              <span>耗时</span>
              <em>Duration</em>
            </th>
            <th>
              <span>错误信息</span>
              <em>Text</em>
            </th>
          </tr>
        </thead>
        <tbody>
          {runs.length > 0 ? (
            runs.map((run) => (
              <tr key={run.id}>
                <td>{formatTime(run.startedAt)}</td>
                <td className="mono">{run.id}</td>
                <td className="strong">{businessTypeName(run.businessType)}</td>
                <td>{run.title}</td>
                <td>{currentStepName(run)}</td>
                <td>
                  <span className={`status-chip ${statusClass[run.status]}`}>{statusLabels[run.status]}</span>
                </td>
                <td>{run.events.length}</td>
                <td>{runDuration(run)}</td>
                <td className="result-cell">
                  {run.error ? (
                    <Tooltip title={run.error} placement="topLeft">
                      <span>{run.error}</span>
                    </Tooltip>
                  ) : (
                    "-"
                  )}
                </td>
              </tr>
            ))
          ) : (
            <tr>
              <td colSpan={9} className="empty-table-cell">
                暂无执行记录
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

function parseWords(input: string) {
  return input
    .split(/[\s,，;；]+/)
    .map((word) => word.trim())
    .filter(Boolean);
}

function sentenceStatusClass(status: GenerateSentenceResponse["status"] | "running") {
  if (status === "running") {
    return "status-running";
  }
  if (status === "success") {
    return "status-success";
  }
  return "status-failed";
}

function clozeSourceName(source: string, sourceEventIds?: number[]) {
  if (sourceEventIds && sourceEventIds.length > 0) {
    return "外部错题触发";
  }
  if (source === "word-agent") {
    return "Python 生成";
  }
  return source || "-";
}

function clozeResultSourceName(item: Pick<ClozeResultItem, "source" | "sourceEventIds">) {
  return clozeSourceName(item.source, item.sourceEventIds);
}

function clozeWrongWordsLabel(item: Pick<UserClozeWrongItem, "word" | "words">) {
  const words = item.words.length > 0 ? item.words : [item.word].filter(Boolean);
  return words.length > 0 ? words.join(", ") : "-";
}

function clozeAnswerText(values: string[], fallback = "未作答") {
  return values.length > 0 ? values.join(" / ") : fallback;
}

function libraryDisplayName(library: WordLibraryItem) {
  return library.libraryMeaning || library.libraryName;
}

function libraryGroupTitle(library: WordLibraryItem) {
  return libraryDisplayName(library).replace(/(上册|下册|第\d+册)$/, "") || libraryDisplayName(library);
}

function libraryChildTitle(library: WordLibraryItem) {
  const displayName = libraryDisplayName(library);
  const match = displayName.match(/(上册|下册|第\d+册)$/);
  return match ? match[1] : displayName;
}

function libraryVolumeOrder(library: WordLibraryItem) {
  const childTitle = libraryChildTitle(library);
  if (childTitle === "上册") {
    return 1;
  }
  if (childTitle === "下册") {
    return 2;
  }
  const match = childTitle.match(/^第(\d+)册$/);
  return match ? Number(match[1]) : 999;
}

function groupWordLibraries(libraries: WordLibraryItem[]) {
  const groupMap = new Map<string, WordLibraryGroup>();
  libraries.forEach((library) => {
    const title = libraryGroupTitle(library);
    const existing = groupMap.get(title);
    if (existing) {
      existing.firstID = Math.min(existing.firstID, library.id);
      existing.totalWordCount += library.wordCount;
      existing.libraries.push(library);
      return;
    }
    groupMap.set(title, {
      key: title,
      title,
      firstID: library.id,
      totalWordCount: library.wordCount,
      libraries: [library],
    });
  });

  return Array.from(groupMap.values())
    .map((group) => ({
      ...group,
      libraries: [...group.libraries].sort((left, right) => {
        const volumeDiff = libraryVolumeOrder(left) - libraryVolumeOrder(right);
        return volumeDiff || left.id - right.id;
      }),
    }))
    .sort((left, right) => left.firstID - right.firstID);
}

function joinIDs(ids: number[]) {
  return ids.length > 0 ? ids.join(", ") : "-";
}

function SentenceTaskPage() {
  const [messageApi, contextHolder] = message.useMessage();
  const queryClient = useQueryClient();
  const [wordsInput, setWordsInput] = useState("brisk, anchor, harbor");
  const [latestResult, setLatestResult] = useState<GenerateSentenceResponse | null>(null);
  const [detailItem, setDetailItem] = useState<SentenceHistoryItem | null>(null);
  const sentenceHistoryQuery = useQuery({
    queryKey: ["sentence-history"],
    queryFn: () => listSentenceHistory() as Promise<SentenceHistoryItem[]>,
  });
  const history = sentenceHistoryQuery.data ?? [];

  const [historyModalOpen, setHistoryModalOpen] = useState(false);
  const [historyPage, setHistoryPage] = useState(1);
  const [historyPageSize, setHistoryPageSize] = useState(DEFAULT_PAGE_SIZE);

  const paginatedHistoryQuery = useQuery({
    queryKey: ["sentence-history-paginated", historyPage, historyPageSize],
    queryFn: () => listSentenceHistory(historyPage, historyPageSize) as Promise<PaginatedResult<SentenceHistoryItem>>,
    enabled: historyModalOpen,
  });

  const paginatedHistory = paginatedHistoryQuery.data?.list ?? [];
  const historyTotal = paginatedHistoryQuery.data?.total ?? 0;

  useEffect(() => {
    if (!latestResult && history.length > 0) {
      setLatestResult(history[0]);
    }
  }, [history, latestResult]);

  const sentenceMutation = useMutation({
    mutationFn: generateSentence,
    onSuccess: (data) => {
      setLatestResult(data);
      queryClient.invalidateQueries({ queryKey: ["sentence-history"] });
      queryClient.invalidateQueries({ queryKey: ["sentence-history-paginated"] });
      queryClient.invalidateQueries({ queryKey: ["workflow-runs"] });
      messageApi.success("生成成功");
    },
    onError: (error) => {
      if (error instanceof SentenceApiError && error.data) {
        setLatestResult(error.data);
      }
      queryClient.invalidateQueries({ queryKey: ["sentence-history"] });
      queryClient.invalidateQueries({ queryKey: ["sentence-history-paginated"] });
      queryClient.invalidateQueries({ queryKey: ["workflow-runs"] });
      messageApi.error(error instanceof Error ? error.message : "生成失败");
    },
  });

  const parsedWords = useMemo(() => parseWords(wordsInput), [wordsInput]);
  const currentStatus = sentenceMutation.isPending ? "running" : latestResult?.status;

  function handleGenerate() {
    if (parsedWords.length === 0) {
      messageApi.warning("请先输入单词");
      return;
    }
    sentenceMutation.mutate({ words: parsedWords });
  }

  function handleRefreshHistory() {
    sentenceHistoryQuery.refetch();
    queryClient.invalidateQueries({ queryKey: ["workflow-runs"] });
  }

  return (
    <>
      {contextHolder}
      <header className="content-header">
        <div>
          <h1>单词造句</h1>
          <p>React 提交任务，Go 调用 Python word-agent 生成句子</p>
        </div>
        <div className="content-actions">
          <Button icon={<HistoryOutlined />} onClick={() => {
            setHistoryPage(1);
            setHistoryModalOpen(true);
          }}>
            历史记录
          </Button>
          <Button icon={<ReloadOutlined />} onClick={() => setWordsInput("")}>
            清空输入
          </Button>
          <Button type="primary" icon={<SendOutlined />} onClick={handleGenerate} loading={sentenceMutation.isPending}>
            生成句子
          </Button>
        </div>
      </header>

      <section className="sentence-panel">
        <label className="sentence-input-field">
          <span>单词</span>
          <Input.TextArea
            rows={3}
            value={wordsInput}
            onChange={(event) => setWordsInput(event.target.value)}
            placeholder="brisk, anchor, harbor"
          />
        </label>
        <div className="word-preview">
          {parsedWords.length > 0 ? (
            parsedWords.map((word) => <span key={word}>{word}</span>)
          ) : (
            <em>待输入</em>
          )}
        </div>
      </section>

      <section className="sentence-result-panel">
        <div className="sentence-result-header">
          <strong>当前结果</strong>
          {currentStatus ? (
            <span className={`status-chip ${sentenceStatusClass(currentStatus)}`}>
              {currentStatus === "running" ? "生成中" : currentStatus === "success" ? "成功" : "失败"}
            </span>
          ) : null}
        </div>

        {sentenceMutation.isPending ? (
          <div className="sentence-loading">
            <Spin />
          </div>
        ) : latestResult ? (
          <>
            <p className="sentence-output">{latestResult.sentence || "未生成句子"}</p>
            <div className="sentence-explanation sentence-translation">
              <span>翻译</span>
              <p>{latestResult.translationZh || "暂无翻译"}</p>
            </div>
            <div className="sentence-explanation">
              <span>中文解释</span>
              <p>{latestResult.explanationZh || "暂无中文解释"}</p>
            </div>
            <div className="sentence-meta-grid">
              <div>
                <span>Run ID</span>
                <strong>{latestResult.runId}</strong>
              </div>
              <div>
                <span>Provider</span>
                <strong>{latestResult.providerLabel || "-"}</strong>
              </div>
              <div>
                <span>Model</span>
                <strong>{latestResult.model || "-"}</strong>
              </div>
              <div>
                <span>耗时</span>
                <strong>{latestResult.durationMs ? `${latestResult.durationMs}ms` : "-"}</strong>
              </div>
            </div>
          </>
        ) : (
          <div className="empty-inline">点击“生成句子”后，这里展示本次结果；下方表格展示历史记录。</div>
        )}
      </section>

      <section className="sentence-history-panel">
        <div className="sentence-result-header">
          <strong>最近 5 条造句记录</strong>
          <Button size="small" onClick={handleRefreshHistory} loading={sentenceHistoryQuery.isFetching}>
            刷新
          </Button>
        </div>
        <div className="sentence-table-scroll">
          <table className="sentence-table">
            <thead>
              <tr>
                <th>时间</th>
                <th>单词</th>
                <th>生成句子</th>
                <th>翻译</th>
                <th>模型</th>
                <th>耗时</th>
              </tr>
            </thead>
            <tbody>
              {sentenceHistoryQuery.isLoading ? (
                <tr>
                  <td colSpan={6} className="empty-table-cell">
                    加载中...
                  </td>
                </tr>
              ) : history.length > 0 ? (
                history.map((item) => (
                  <tr key={`${item.runId}-${item.createdAt}`}>
                    <td>{formatTime(item.createdAt)}</td>
                    <td>
                      <button
                        type="button"
                        className="word-detail-link"
                        onClick={() => setDetailItem(item)}
                        title="查看造句详情"
                      >
                        {item.words.join(", ")}
                      </button>
                    </td>
                    <td>
                      <Tooltip title={item.sentence} placement="topLeft">
                        <span className="table-ellipsis-text">{item.sentence}</span>
                      </Tooltip>
                    </td>
                    <td>
                      <Tooltip title={item.translationZh || "-"} placement="topLeft">
                        <span className="table-ellipsis-text">{item.translationZh || "-"}</span>
                      </Tooltip>
                    </td>
                    <td>{item.model}</td>
                    <td>{item.durationMs}ms</td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td colSpan={6} className="empty-table-cell">
                    暂无记录
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </section>

      <Modal
        title="造句详情"
        open={Boolean(detailItem)}
        onCancel={() => setDetailItem(null)}
        footer={null}
        width={760}
      >
        {detailItem ? (
          <div className="sentence-detail-modal">
            <div className="detail-words">
              {detailItem.words.map((word) => (
                <span key={word}>{word}</span>
              ))}
            </div>

            <div className="detail-block">
              <span>生成句子</span>
              <p>{detailItem.sentence || "-"}</p>
            </div>

            <div className="detail-block">
              <span>翻译</span>
              <p>{detailItem.translationZh || "暂无翻译"}</p>
            </div>

            <div className="detail-block">
              <span>中文解释</span>
              <p>{detailItem.explanationZh || "暂无中文解释"}</p>
            </div>

            <div className="detail-meta-grid">
              <div>
                <span>生成时间</span>
                <strong>{formatTime(detailItem.createdAt)}</strong>
              </div>
              <div>
                <span>Run ID</span>
                <strong>{detailItem.runId}</strong>
              </div>
              <div>
                <span>Provider</span>
                <strong>{detailItem.providerLabel || "-"}</strong>
              </div>
              <div>
                <span>Model</span>
                <strong>{detailItem.model || "-"}</strong>
              </div>
              <div>
                <span>状态</span>
                <strong>{detailItem.status === "success" ? "成功" : "失败"}</strong>
              </div>
              <div>
                <span>耗时</span>
                <strong>{detailItem.durationMs}ms</strong>
              </div>
            </div>
          </div>
        ) : null}
      </Modal>

      <Modal
        title="造句历史记录"
        open={historyModalOpen}
        onCancel={() => setHistoryModalOpen(false)}
        footer={null}
        width="100vw"
        style={{ top: 0, margin: 0, padding: 0, maxWidth: '100vw' }}
        className="history-fullscreen-modal"
        destroyOnHidden
      >
        <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
          <div className="sentence-table-scroll" style={{ flex: 1, overflowY: 'auto' }}>
            <table className="sentence-table" style={{ width: '100%', minWidth: '980px' }}>
              <thead>
                <tr>
                  <th>时间</th>
                  <th>单词</th>
                  <th>生成句子</th>
                  <th>翻译</th>
                  <th>模型</th>
                  <th>耗时</th>
                </tr>
              </thead>
              <tbody>
                {paginatedHistoryQuery.isLoading ? (
                  <tr>
                    <td colSpan={6} className="empty-table-cell">
                      加载中...
                    </td>
                  </tr>
                ) : paginatedHistory.length > 0 ? (
                  paginatedHistory.map((item) => (
                    <tr key={`${item.runId}-${item.createdAt}`}>
                      <td>{formatTime(item.createdAt)}</td>
                      <td>
                        <button
                          type="button"
                          className="word-detail-link"
                          onClick={() => {
                            setDetailItem(item);
                          }}
                          title="查看造句详情"
                        >
                          {item.words.join(", ")}
                        </button>
                      </td>
                      <td>
                        <Tooltip title={item.sentence} placement="topLeft">
                          <span className="table-ellipsis-text">{item.sentence}</span>
                        </Tooltip>
                      </td>
                      <td>
                        <Tooltip title={item.translationZh || "-"} placement="topLeft">
                          <span className="table-ellipsis-text">{item.translationZh || "-"}</span>
                        </Tooltip>
                      </td>
                      <td>{item.model}</td>
                      <td>{item.durationMs}ms</td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={6} className="empty-table-cell">
                      暂无记录
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
          {historyTotal > 0 && (
            <div style={{ marginTop: 16, display: 'flex', justifyContent: 'flex-end', alignItems: 'center' }}>
              <AppPagination
                current={historyPage}
                pageSize={historyPageSize}
                total={historyTotal}
                onChange={(page, pageSize) => {
                  setHistoryPage(page);
                  setHistoryPageSize(pageSize);
                }}
                totalLabel="条记录"
              />
            </div>
          )}
        </div>
      </Modal>
    </>
  );
}

function dateStartMs(value: string) {
  return value ? new Date(`${value}T00:00:00`).getTime() : undefined;
}

function dateEndMs(value: string) {
  return value ? new Date(`${value}T23:59:59.999`).getTime() : undefined;
}

function RecordsPage() {
  const [keywordInput, setKeywordInput] = useState("");
  const [startDateInput, setStartDateInput] = useState("");
  const [endDateInput, setEndDateInput] = useState("");
  const [filters, setFilters] = useState({
    keyword: "",
    startDate: "",
    endDate: "",
  });
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const runsQuery = useQuery({
    queryKey: ["workflow-runs", filters, page, pageSize],
    queryFn: () =>
      listWorkflowRuns({
        keyword: filters.keyword,
        page,
        pageSize,
        startedAtFrom: dateStartMs(filters.startDate),
        startedAtTo: dateEndMs(filters.endDate),
      }),
    refetchInterval: 10000,
  });

  const runs = runsQuery.data?.list ?? [];
  const total = runsQuery.data?.total ?? 0;

  function handleQuery() {
    setFilters({
      keyword: keywordInput.trim(),
      startDate: startDateInput,
      endDate: endDateInput,
    });
    setPage(1);
  }

  return (
    <>
      <header className="content-header">
        <div>
          <h1>执行业务记录</h1>
          <p>记录 Go 调用各业务流程后的执行状态、步骤和耗时</p>
        </div>
        <div className="content-actions">
          <Button icon={<CloudSyncOutlined />}>数据同步</Button>
          <Button icon={<ReloadOutlined />} onClick={() => runsQuery.refetch()} loading={runsQuery.isFetching}>
            刷新
          </Button>
        </div>
      </header>

      <section className="filter-panel">
        <div className="filter-field date-field">
          <label>执行日期</label>
          <div className="date-range">
            <input type="date" value={startDateInput} onChange={(event) => setStartDateInput(event.target.value)} />
            <span>→</span>
            <input type="date" value={endDateInput} onChange={(event) => setEndDateInput(event.target.value)} />
          </div>
        </div>
        <div className="filter-field word-field">
          <label>业务 / Run ID</label>
          <Input
            allowClear
            prefix={<SearchOutlined />}
            placeholder="输入业务类型、Run ID 或步骤名称"
            value={keywordInput}
            onChange={(event) => setKeywordInput(event.target.value)}
            onPressEnter={handleQuery}
          />
        </div>
        <Button className="query-button" type="primary" onClick={handleQuery}>
          查询
        </Button>
      </section>

      {runsQuery.isLoading ? (
        <div className="loading-area">
          <Spin />
        </div>
      ) : (
        <>
          <TrackerTable runs={runs} />
          <div className="table-pagination">
            <AppPagination
              current={page}
              pageSize={pageSize}
              total={total}
              onChange={(nextPage, nextPageSize) => {
                setPage(nextPage);
                setPageSize(nextPageSize);
              }}
              totalLabel="条记录"
            />
          </div>
        </>
      )}
    </>
  );
}

function ClozeResultsPage() {
  const [userKeyword, setUserKeyword] = useState("");
  const [resultKeyword, setResultKeyword] = useState("");
  const [userPage, setUserPage] = useState(1);
  const [userPageSize, setUserPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [resultPage, setResultPage] = useState(1);
  const [resultPageSize, setResultPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [userPickerOpen, setUserPickerOpen] = useState(false);
  const [selectedUser, setSelectedUser] = useState<ClozeResultUserSummary | null>(null);

  const usersQuery = useQuery({
    queryKey: ["cloze-result-users", userKeyword, userPage, userPageSize],
    queryFn: () =>
      listClozeResultUsers({
        keyword: userKeyword.trim(),
        page: userPage,
        pageSize: userPageSize,
      }),
    enabled: userPickerOpen,
  });

  const resultsQuery = useQuery({
    queryKey: ["cloze-result-items", selectedUser?.userId, resultKeyword, resultPage, resultPageSize],
    queryFn: () =>
      listClozeResultItems({
        userId: selectedUser?.userId,
        keyword: resultKeyword.trim(),
        page: resultPage,
        pageSize: resultPageSize,
      }),
  });

  const users = usersQuery.data?.list ?? [];
  const userTotal = usersQuery.data?.total ?? 0;
  const results = resultsQuery.data?.list ?? [];
  const resultTotal = resultsQuery.data?.total ?? 0;
  const resultResetKey = [
    selectedUser?.userId ?? "all",
    resultKeyword.trim(),
    resultPage,
    resultPageSize,
  ].join(":");

  function handleSelectUser(user: ClozeResultUserSummary) {
    setSelectedUser(user);
    setResultPage(1);
    setUserPickerOpen(false);
  }

  function handleClearUser() {
    setSelectedUser(null);
    setResultPage(1);
  }

  function handleRefresh() {
    resultsQuery.refetch();
    if (userPickerOpen) {
      usersQuery.refetch();
    }
  }

  return (
    <>
      <header className="content-header">
        <div>
          <h1>用户造句结果</h1>
          <p>按用户查看外部错题触发后生成的句子和挖空结果</p>
        </div>
        <div className="content-actions">
          <Button
            icon={<ReloadOutlined />}
            onClick={handleRefresh}
            loading={usersQuery.isFetching || resultsQuery.isFetching}
          >
            刷新
          </Button>
        </div>
      </header>

      <section className="cloze-filter-panel">
        <label className="filter-field">
          <span>用户</span>
          <div className="cloze-selected-user">
            <button type="button" className="cloze-selected-user-button" onClick={() => setUserPickerOpen(true)}>
              {selectedUser ? selectedUser.userName || `用户 ${selectedUser.userId}` : "全部用户"}
            </button>
            {selectedUser ? (
              <Button onClick={handleClearUser}>
                清除
              </Button>
            ) : null}
          </div>
        </label>
        <label className="filter-field">
          <span>内容</span>
          <Input
            allowClear
            prefix={<SearchOutlined />}
            placeholder="搜索单词、句子或翻译"
            value={resultKeyword}
            onChange={(event) => {
              setResultKeyword(event.target.value);
              setResultPage(1);
            }}
          />
        </label>
      </section>

      <section className="cloze-results-panel">
        <section className="cloze-result-detail-panel">
          <div className="cloze-detail-title">
            <div>
              <strong>造句结果列表</strong>
              <span>{selectedUser ? `当前用户：${selectedUser.userName || `用户 ${selectedUser.userId}`}` : "当前用户：全部用户"}</span>
            </div>
            <span>{resultTotal} 条结果</span>
          </div>

          <ClozeResultTable
            items={results}
            loading={resultsQuery.isLoading}
            page={resultPage}
            pageSize={resultPageSize}
            resetKey={resultResetKey}
          />

          {resultTotal > 0 ? (
            <div className="cloze-result-pagination">
              <AppPagination
                current={resultPage}
                pageSize={resultPageSize}
                total={resultTotal}
                onChange={(page, pageSize) => {
                  setResultPage(page);
                  setResultPageSize(pageSize);
                }}
                totalLabel="条结果"
              />
            </div>
          ) : null}
        </section>
      </section>

      <Modal
        title="选择用户"
        open={userPickerOpen}
        onCancel={() => setUserPickerOpen(false)}
        footer={null}
        width={720}
      >
        <div className="cloze-user-picker">
          <Input
            allowClear
            prefix={<SearchOutlined />}
            placeholder="搜索用户或 ID"
            value={userKeyword}
            onChange={(event) => {
              setUserKeyword(event.target.value);
              setUserPage(1);
            }}
          />
          <div className="cloze-user-picker-list">
            {usersQuery.isLoading ? (
              <div className="cloze-loading">
                <Spin />
              </div>
            ) : users.length > 0 ? (
              users.map((user) => (
                <button
                  type="button"
                  key={user.userId}
                  className={`cloze-user-row ${selectedUser?.userId === user.userId ? "active" : ""}`}
                  onClick={() => handleSelectUser(user)}
                >
                  <span className="cloze-user-name">{user.userName || `用户 ${user.userId}`}</span>
                  <span className="cloze-user-meta">ID {user.userId}</span>
                  <span className="cloze-user-count">{user.totalCount}</span>
                  <span className="cloze-user-time">{user.totalCount > 0 ? formatTime(user.latestTime) : "暂无结果"}</span>
                </button>
              ))
            ) : usersQuery.error instanceof Error ? (
              <div className="empty-inline">{usersQuery.error.message}</div>
            ) : (
              <div className="empty-inline">暂无用户</div>
            )}
          </div>
          <AppPagination
            className="cloze-user-pagination"
            current={userPage}
            pageSize={userPageSize}
            total={userTotal}
            onChange={(page, pageSize) => {
              setUserPage(page);
              setUserPageSize(pageSize);
            }}
            totalLabel="位用户"
          />
        </div>
      </Modal>

    </>
  );
}

function formatFullTime(value?: string) {
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
  return `${year}-${month}-${day} ${hour}:${minute}`;
}

function formatSeconds(seconds?: number) {
  if (!seconds) {
    return "-";
  }
  if (seconds < 60) {
    return `${seconds}s`;
  }
  return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
}

function formatAnswerTime(ms?: number) {
  if (!ms) {
    return "-";
  }
  if (ms < 1000) {
    return `${ms}ms`;
  }
  return `${(ms / 1000).toFixed(1)}s`;
}

function trainingDifficultyLabel(round: UserTrainingRound) {
  if (round.mode === "match") {
    const opponent = round.opponentName || "未知对手";
    return `匹配 · ${opponent}${round.resultLabel ? ` · ${round.resultLabel}` : ""}`;
  }
  if (round.trainingDifficultyLevel === "rank_current") {
    return "段位难度";
  }
  const groupLabel = TRAINING_DIFFICULTY_GROUP_LABELS[round.trainingDifficultyGroup];
  const levelLabel = TRAINING_DIFFICULTY_LEVEL_LABELS[round.trainingDifficultyLevel];
  if (groupLabel && levelLabel) {
    return `${groupLabel} · ${levelLabel}`;
  }
  return levelLabel || groupLabel || "段位难度";
}

function answerText(
  detail: Pick<UserTrainingAnswerDetail | UserWrongWordHistory, "option1" | "option2" | "option3" | "option4">,
  index?: number | null,
) {
  if (!index) {
    return "未作答";
  }
  const options: Record<number, string> = {
    1: detail.option1,
    2: detail.option2,
    3: detail.option3,
    4: detail.option4,
  };
  return options[index] || "-";
}

function wrongWordDifficultyLabel(item: Pick<UserWrongWordItem, "latestMode" | "latestGroup" | "latestLevel">) {
  if (item.latestMode === "match") {
    return "匹配答题";
  }
  if (item.latestMode === "cloze_review") {
    return "挖空复习";
  }
  if (item.latestLevel === "rank_current") {
    return "段位难度";
  }
  const groupLabel = TRAINING_DIFFICULTY_GROUP_LABELS[item.latestGroup];
  const levelLabel = TRAINING_DIFFICULTY_LEVEL_LABELS[item.latestLevel];
  if (groupLabel && levelLabel) {
    return `${groupLabel} · ${levelLabel}`;
  }
  return levelLabel || groupLabel || "段位难度";
}

function wrongHistoryDifficultyLabel(history: UserWrongWordHistory) {
  return wrongWordDifficultyLabel({
    latestMode: history.mode,
    latestGroup: history.trainingDifficultyGroup,
    latestLevel: history.trainingDifficultyLevel,
  });
}

type MasteredWordStatusFilter = "" | "learning" | "mastered";

interface MasteredWordsPanelProps {
  items: UserMasteredWordItem[];
  loading: boolean;
  errorMessage?: string;
  page: number;
  pageSize: number;
  total: number;
  showUser: boolean;
  emptyText: string;
  onPageChange: (page: number, pageSize: number) => void;
}

function masteryStatusLabel(status: string) {
  if (status === "mastered") {
    return "已掌握";
  }
  return "复习中";
}

function masteryStageLabel(stage: number, status: string) {
  if (status === "mastered") {
    return "3/3";
  }
  const normalizedStage = Math.max(0, Math.min(stage || 0, 2));
  return `${normalizedStage}/3`;
}

function masteryReviewLabel(item: UserMasteredWordItem) {
  if (item.status === "mastered") {
    return item.masteredTime ? `掌握 ${formatFullTime(item.masteredTime)}` : "已掌握";
  }
  if (item.nextReviewTime) {
    return `复习 ${formatFullTime(item.nextReviewTime)}`;
  }
  return `再答对 ${Math.max(1, 3 - Math.max(0, Math.min(item.stage || 0, 2)))} 次掌握`;
}

function MasteredStatusTabs({
  status,
  onChange,
}: {
  status: MasteredWordStatusFilter;
  onChange: (status: MasteredWordStatusFilter) => void;
}) {
  return (
    <div className="user-results-mode-tabs mastery-filter-tabs">
      <button type="button" className={status === "" ? "active" : ""} onClick={() => onChange("")}>
        全部
      </button>
      <button type="button" className={status === "learning" ? "active" : ""} onClick={() => onChange("learning")}>
        复习中
      </button>
      <button type="button" className={status === "mastered" ? "active" : ""} onClick={() => onChange("mastered")}>
        已掌握
      </button>
    </div>
  );
}

function MasteredWordsPanel({
  items,
  loading,
  errorMessage,
  page,
  pageSize,
  total,
  showUser,
  emptyText,
  onPageChange,
}: MasteredWordsPanelProps) {
  return (
    <>
      <div className="wrong-table-panel mastered-table-panel">
        {loading ? (
          <div className="empty-table-cell">
            <Spin />
          </div>
        ) : errorMessage ? (
          <div className="empty-table-cell">{errorMessage}</div>
        ) : items.length === 0 ? (
          <div className="empty-table-cell">{emptyText}</div>
        ) : (
          <>
            <div className="wrong-table-head mastered-table-head">
              <span>序号</span>
              <span>单词</span>
              <span>{showUser ? "用户 / 释义" : "释义"}</span>
              <span>状态</span>
              <span>进度</span>
              <span>答对</span>
              <span>难度</span>
              <span>词库</span>
              <span>最近答对</span>
              <span>下次 / 掌握</span>
            </div>
            <div className="wrong-table-body">
              {items.map((item, index) => (
                <article className={`wrong-table-card mastered-table-card ${item.status === "mastered" ? "mastered" : "learning"}`} key={`${item.userId}-${item.wordId}`}>
                  <div className="wrong-table-row mastered-table-row">
                    <span>#{(page - 1) * pageSize + index + 1}</span>
                    <strong>{item.wordContent || "-"}</strong>
                    <Tooltip title={item.correctMeaning || "-"} placement="topLeft">
                      <span>{showUser ? `${item.userName} · ${item.correctMeaning || "-"}` : item.correctMeaning || "-"}</span>
                    </Tooltip>
                    <span className={`mastery-status-pill ${item.status === "mastered" ? "mastered" : "learning"}`}>
                      {masteryStatusLabel(item.status)}
                    </span>
                    <span>{masteryStageLabel(item.stage, item.status)}</span>
                    <span>{item.correctCount} 次</span>
                    <span>{item.wordDifficulty || "-"}</span>
                    <Tooltip title={item.libraryName || item.libraryMeaning || "-"} placement="topLeft">
                      <span>{item.libraryMeaning || item.libraryName || "-"}</span>
                    </Tooltip>
                    <span>{formatFullTime(item.lastCorrectTime || undefined)}</span>
                    <span>{masteryReviewLabel(item)}</span>
                  </div>
                </article>
              ))}
            </div>
          </>
        )}
      </div>

      <div className="table-pagination">
        <AppPagination current={page} pageSize={pageSize} total={total} onChange={onPageChange} totalLabel="个掌握记录" />
      </div>
    </>
  );
}

function UserMasteredWordsPage() {
  const [keyword, setKeyword] = useState("");
  const [status, setStatus] = useState<MasteredWordStatusFilter>("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);

  const masteredWordsQuery = useQuery({
    queryKey: ["app-user-mastered-words", keyword, status, page, pageSize],
    queryFn: () =>
      listUserMasteredWords({
        keyword: keyword.trim(),
        status,
        page,
        pageSize,
      }),
  });

  const items: UserMasteredWordItem[] = masteredWordsQuery.data?.list ?? [];
  const total = masteredWordsQuery.data?.total ?? 0;
  const currentPageMastered = items.filter((item) => item.status === "mastered").length;
  const currentPageLearning = items.filter((item) => item.status !== "mastered").length;
  const uniqueUserCount = new Set(items.map((item) => item.userId)).size;

  function handleSearchChange(value: string) {
    setKeyword(value);
    setPage(1);
  }

  function handleStatusChange(nextStatus: MasteredWordStatusFilter) {
    setStatus(nextStatus);
    setPage(1);
  }

  return (
    <>
      <header className="page-header">
        <div>
          <h1>用户已掌握单词</h1>
          <p>查看单人训练、匹配和挖空练习产生的掌握进度</p>
        </div>
        <Button icon={<ReloadOutlined />} onClick={() => masteredWordsQuery.refetch()} loading={masteredWordsQuery.isFetching}>
          刷新
        </Button>
      </header>

      <section className="user-wrong-summary">
        <div>
          <span>掌握记录</span>
          <strong>{total}</strong>
        </div>
        <div>
          <span>当前页用户</span>
          <strong>{uniqueUserCount}</strong>
        </div>
        <div>
          <span>当前页已掌握 / 复习中</span>
          <strong>
            {currentPageMastered}/{currentPageLearning}
          </strong>
        </div>
      </section>

      <section className="panel compact-panel">
        <div className="table-toolbar mastered-toolbar">
          <Input
            prefix={<SearchOutlined />}
            placeholder="搜索用户、昵称、用户 ID、单词或释义"
            value={keyword}
            onChange={(event) => handleSearchChange(event.target.value)}
            allowClear
          />
          <MasteredStatusTabs status={status} onChange={handleStatusChange} />
          <span>{total} 个掌握记录</span>
        </div>

        <MasteredWordsPanel
          items={items}
          loading={masteredWordsQuery.isLoading}
          errorMessage={masteredWordsQuery.isError ? masteredWordsQuery.error.message : undefined}
          page={page}
          pageSize={pageSize}
          total={total}
          showUser
          emptyText="暂无已掌握单词记录"
          onPageChange={(nextPage, nextPageSize) => {
            setPage(nextPage);
            setPageSize(nextPageSize);
          }}
        />
      </section>
    </>
  );
}

function UserWrongWordsPage() {
  const [keyword, setKeyword] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [expandedKeys, setExpandedKeys] = useState<Set<string>>(() => new Set());

  const wrongWordsQuery = useQuery({
    queryKey: ["app-user-wrong-words", keyword, page, pageSize],
    queryFn: () =>
      listUserWrongWords({
        keyword: keyword.trim(),
        page,
        pageSize,
      }),
  });

  const items: UserWrongWordItem[] = wrongWordsQuery.data?.list ?? [];
  const total = wrongWordsQuery.data?.total ?? 0;
  const totalWrongCount = items.reduce((sum, item) => sum + item.wrongCount, 0);
  const uniqueUserCount = new Set(items.map((item) => item.userId)).size;

  function toggleWrongWord(item: UserWrongWordItem) {
    const key = `${item.userId}-${item.wordContent}`;
    setExpandedKeys((current) => {
      const next = new Set(current);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  }

  function handleSearchChange(value: string) {
    setKeyword(value);
    setPage(1);
  }

  return (
    <>
      <header className="page-header">
        <div>
          <h1>用户错题集</h1>
          <p>按用户和单词展示未完成复习的错词，展开后查看最近普通答题明细</p>
        </div>
        <Button icon={<ReloadOutlined />} onClick={() => wrongWordsQuery.refetch()} loading={wrongWordsQuery.isFetching}>
          刷新
        </Button>
      </header>

      <section className="user-wrong-summary">
        <div>
          <span>待复习错词</span>
          <strong>{total}</strong>
        </div>
        <div>
          <span>当前页用户</span>
          <strong>{uniqueUserCount}</strong>
        </div>
        <div>
          <span>当前页错误次数</span>
          <strong>{totalWrongCount}</strong>
        </div>
      </section>

      <section className="panel compact-panel">
        <div className="table-toolbar">
          <Input
            prefix={<SearchOutlined />}
            placeholder="搜索用户、昵称、用户 ID 或单词"
            value={keyword}
            onChange={(event) => handleSearchChange(event.target.value)}
            allowClear
          />
          <span>{total} 个未完成复习错词</span>
        </div>

        <div className="user-wrong-list">
          {wrongWordsQuery.isLoading ? (
            <div className="empty-table-cell">
              <Spin />
            </div>
          ) : wrongWordsQuery.isError ? (
            <div className="empty-table-cell">{wrongWordsQuery.error.message}</div>
          ) : items.length === 0 ? (
            <div className="empty-table-cell">暂无未完成复习错词</div>
          ) : (
            items.map((item, index) => {
              const key = `${item.userId}-${item.wordContent}`;
              const expanded = expandedKeys.has(key);
              const accuracy = item.totalAttempts > 0 ? Math.round(((item.totalAttempts - item.wrongCount) / item.totalAttempts) * 100) : 0;
              return (
                <article className="user-wrong-card" key={key}>
                  <button type="button" className="user-wrong-row" onClick={() => toggleWrongWord(item)}>
                    <span>#{(page - 1) * pageSize + index + 1}</span>
                    <strong>{item.wordContent}</strong>
                    <span>{item.userName}</span>
                    <span>错误 {item.wrongCount} 次</span>
                    <span>总答 {item.totalAttempts} 次</span>
                    <span>正确率 {accuracy}%</span>
                    <span>均难 {item.avgDifficulty}</span>
                    <span>{wrongWordDifficultyLabel(item)}</span>
                    <span>{formatTime(item.lastWrongTime)}</span>
                    <span className="expand-text">{expanded ? "收起" : "展开"}</span>
                  </button>

                  {expanded ? (
                    <div className="user-wrong-detail">
                      <div className="user-wrong-history-grid user-wrong-history-head">
                        <span>时间</span>
                        <span>模式/难度</span>
                        <span>词难度</span>
                        <span>耗时</span>
                        <span>正确答案</span>
                      </div>
                      {item.recentHistories.map((history) => (
                        <div className="user-wrong-history-grid user-wrong-history-row" key={history.detailId}>
                          <span>{formatTime(history.startTime)}</span>
                          <span>{wrongHistoryDifficultyLabel(history)}</span>
                          <span>{history.wordDifficulty}</span>
                          <span>{formatAnswerTime(history.answerTimeMs)}</span>
                          <span>{answerText(history, history.correctAnswerIndex)}</span>
                        </div>
                      ))}
                      {item.recentHistories.length === 0 ? (
                        <div className="empty-table-cell">暂无普通答题明细，可在用户造句错题集中查看挖空历史</div>
                      ) : null}
                    </div>
                  ) : null}
                </article>
              );
            })
          )}
        </div>

        <div className="table-pagination">
          <AppPagination
            current={page}
            pageSize={pageSize}
            total={total}
            onChange={(nextPage, nextPageSize) => {
              setPage(nextPage);
              setPageSize(nextPageSize);
            }}
            totalLabel="个错词"
          />
        </div>
      </section>
    </>
  );
}

function UserClozeWrongWordsPage() {
  const [keyword, setKeyword] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [expandedKeys, setExpandedKeys] = useState<Set<string>>(() => new Set());

  const wrongWordsQuery = useQuery({
    queryKey: ["app-user-cloze-wrong-words", keyword, page, pageSize],
    queryFn: () =>
      listUserClozeWrongWords({
        keyword: keyword.trim(),
        page,
        pageSize,
      }),
  });

  const items: UserClozeWrongItem[] = wrongWordsQuery.data?.list ?? [];
  const total = wrongWordsQuery.data?.total ?? 0;
  const totalWrongCount = items.reduce((sum, item) => sum + item.wrongCount, 0);
  const uniqueUserCount = new Set(items.map((item) => item.userId)).size;

  function toggleWrongWord(item: UserClozeWrongItem) {
    const key = `${item.userId}-${item.clozeItemId}`;
    setExpandedKeys((current) => {
      const next = new Set(current);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  }

  function handleSearchChange(value: string) {
    setKeyword(value);
    setPage(1);
  }

  return (
    <>
      <header className="page-header">
        <div>
          <h1>用户造句错题集</h1>
          <p>按用户和造句题聚合答错记录，展开后查看最近答错明细</p>
        </div>
        <Button icon={<ReloadOutlined />} onClick={() => wrongWordsQuery.refetch()} loading={wrongWordsQuery.isFetching}>
          刷新
        </Button>
      </header>

      <section className="user-wrong-summary">
        <div>
          <span>错题句子</span>
          <strong>{total}</strong>
        </div>
        <div>
          <span>当前页用户</span>
          <strong>{uniqueUserCount}</strong>
        </div>
        <div>
          <span>当前页错误次数</span>
          <strong>{totalWrongCount}</strong>
        </div>
      </section>

      <section className="panel compact-panel">
        <div className="table-toolbar">
          <Input
            prefix={<SearchOutlined />}
            placeholder="搜索用户、昵称、用户 ID、单词或句子"
            value={keyword}
            onChange={(event) => handleSearchChange(event.target.value)}
            allowClear
          />
          <span>{total} 个造句错题</span>
        </div>

        <div className="user-wrong-list">
          {wrongWordsQuery.isLoading ? (
            <div className="empty-table-cell">
              <Spin />
            </div>
          ) : wrongWordsQuery.isError ? (
            <div className="empty-table-cell">{wrongWordsQuery.error.message}</div>
          ) : items.length === 0 ? (
            <div className="empty-table-cell">暂无造句错题记录</div>
          ) : (
            items.map((item, index) => {
              const key = `${item.userId}-${item.clozeItemId}`;
              const expanded = expandedKeys.has(key);
              const accuracy =
                item.totalAttempts > 0 ? Math.max(0, Math.round(((item.totalAttempts - item.wrongCount) / item.totalAttempts) * 100)) : 0;
              const sentenceText = item.sentence || item.clozeSentence || "-";
              return (
                <article className="user-wrong-card" key={key}>
                  <button type="button" className="user-wrong-row user-cloze-wrong-row" onClick={() => toggleWrongWord(item)}>
                    <span>#{(page - 1) * pageSize + index + 1}</span>
                    <Tooltip title={sentenceText} placement="topLeft">
                      <strong>{sentenceText}</strong>
                    </Tooltip>
                    <span>{item.userName}</span>
                    <span>{clozeWrongWordsLabel(item)}</span>
                    <span>错误 {item.wrongCount} 次</span>
                    <span>总答 {item.totalAttempts} 次</span>
                    <span>正确率 {accuracy}%</span>
                    <span>{clozeSourceName(item.source)}</span>
                    <span>{formatTime(item.lastWrongTime)}</span>
                    <span className="expand-text">{expanded ? "收起" : "展开"}</span>
                  </button>

                  {expanded ? (
                    <div className="user-wrong-detail">
                      <div className="user-cloze-context">
                        <div>
                          <span>原句</span>
                          <p>{item.sentence || "-"}</p>
                        </div>
                        <div>
                          <span>挖空句</span>
                          <p>{item.clozeSentence || "-"}</p>
                        </div>
                        <div>
                          <span>翻译</span>
                          <p>{item.translationZh || "-"}</p>
                        </div>
                      </div>
                      <div className="user-cloze-history-grid user-wrong-history-head">
                        <span>时间</span>
                        <span>尝试</span>
                        <span>耗时</span>
                        <span>正确答案</span>
                      </div>
                      {item.recentHistories.map((history) => (
                          <div className="user-cloze-history-grid user-wrong-history-row" key={history.recordId}>
                            <span>{formatTime(history.createTime)}</span>
                            <span>第 {history.attemptNo} 次</span>
                            <span>{formatAnswerTime(history.costMs)}</span>
                            <span>{clozeAnswerText(history.expectedWords, "-")}</span>
                          </div>
                      ))}
                    </div>
                  ) : null}
                </article>
              );
            })
          )}
        </div>

        <div className="table-pagination">
          <AppPagination
            current={page}
            pageSize={pageSize}
            total={total}
            onChange={(nextPage, nextPageSize) => {
              setPage(nextPage);
              setPageSize(nextPageSize);
              setExpandedKeys(new Set());
            }}
            totalLabel="个造句错题"
          />
        </div>
      </section>
    </>
  );
}

function UserListPage() {
  const [keyword, setKeyword] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [resultUser, setResultUser] = useState<AppUserItem | null>(null);
  const [resultMode, setResultMode] = useState<"solo_training" | "match">("solo_training");
  const [expandedRounds, setExpandedRounds] = useState<Set<number>>(() => new Set());
  const [clozeUser, setClozeUser] = useState<AppUserItem | null>(null);
  const [clozePage, setClozePage] = useState(1);
  const [clozePageSize, setClozePageSize] = useState(DEFAULT_PAGE_SIZE);
  const [clozeDetailItem, setClozeDetailItem] = useState<ClozeResultItem | null>(null);
  const [wrongUser, setWrongUser] = useState<AppUserItem | null>(null);
  const [wrongPage, setWrongPage] = useState(1);
  const [wrongPageSize, setWrongPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [expandedWrongKeys, setExpandedWrongKeys] = useState<Set<string>>(() => new Set());
  const [clozeWrongUser, setClozeWrongUser] = useState<AppUserItem | null>(null);
  const [clozeWrongPage, setClozeWrongPage] = useState(1);
  const [clozeWrongPageSize, setClozeWrongPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [expandedClozeWrongKeys, setExpandedClozeWrongKeys] = useState<Set<string>>(() => new Set());
  const [masteredUser, setMasteredUser] = useState<AppUserItem | null>(null);
  const [masteredStatus, setMasteredStatus] = useState<MasteredWordStatusFilter>("");
  const [masteredPage, setMasteredPage] = useState(1);
  const [masteredPageSize, setMasteredPageSize] = useState(DEFAULT_PAGE_SIZE);

  const usersQuery = useQuery({
    queryKey: ["app-users", keyword, page, pageSize],
    queryFn: () =>
      listUsers({
        keyword: keyword.trim(),
        page,
        pageSize,
      }),
  });

  const users: AppUserItem[] = usersQuery.data?.list ?? [];
  const total = usersQuery.data?.total ?? 0;
  const trainingResultsQuery = useQuery({
    queryKey: ["app-user-training-results", resultUser?.id, resultMode],
    queryFn: () => listUserTrainingResults(resultUser?.id ?? 0, resultMode),
    enabled: Boolean(resultUser),
  });
  const trainingRounds: UserTrainingRound[] = trainingResultsQuery.data ?? [];
  const trainingAnswerCount = trainingRounds.reduce((sum, round) => sum + round.details.length, 0);
  const trainingCorrectCount = trainingRounds.reduce((sum, round) => sum + round.correctCount, 0);
  const trainingScore = trainingRounds.reduce((sum, round) => sum + round.score, 0);
  const clozeResultsQuery = useQuery({
    queryKey: ["app-user-cloze-results", clozeUser?.id, clozePage, clozePageSize],
    queryFn: () =>
      listClozeResultItems({
        userId: clozeUser?.id,
        page: clozePage,
        pageSize: clozePageSize,
      }),
    enabled: Boolean(clozeUser),
  });
  const clozeResults: ClozeResultItem[] = clozeResultsQuery.data?.list ?? [];
  const clozeTotal = clozeResultsQuery.data?.total ?? 0;
  const wrongWordsQuery = useQuery({
    queryKey: ["app-user-wrong-words-modal", wrongUser?.id, wrongPage, wrongPageSize],
    queryFn: () =>
      listUserWrongWords({
        userId: wrongUser?.id,
        page: wrongPage,
        pageSize: wrongPageSize,
      }),
    enabled: Boolean(wrongUser),
  });
  const wrongWords: UserWrongWordItem[] = wrongWordsQuery.data?.list ?? [];
  const wrongTotal = wrongWordsQuery.data?.total ?? 0;
  const wrongTotalCount = wrongWords.reduce((sum, item) => sum + item.wrongCount, 0);
  const clozeWrongWordsQuery = useQuery({
    queryKey: ["app-user-cloze-wrong-words-modal", clozeWrongUser?.id, clozeWrongPage, clozeWrongPageSize],
    queryFn: () =>
      listUserClozeWrongWords({
        userId: clozeWrongUser?.id,
        page: clozeWrongPage,
        pageSize: clozeWrongPageSize,
      }),
    enabled: Boolean(clozeWrongUser),
  });
  const clozeWrongWords: UserClozeWrongItem[] = clozeWrongWordsQuery.data?.list ?? [];
  const clozeWrongTotal = clozeWrongWordsQuery.data?.total ?? 0;
  const clozeWrongTotalCount = clozeWrongWords.reduce((sum, item) => sum + item.wrongCount, 0);
  const masteredWordsQuery = useQuery({
    queryKey: ["app-user-mastered-words-modal", masteredUser?.id, masteredStatus, masteredPage, masteredPageSize],
    queryFn: () =>
      listUserMasteredWords({
        userId: masteredUser?.id,
        status: masteredStatus,
        page: masteredPage,
        pageSize: masteredPageSize,
      }),
    enabled: Boolean(masteredUser),
  });
  const masteredWords: UserMasteredWordItem[] = masteredWordsQuery.data?.list ?? [];
  const masteredTotal = masteredWordsQuery.data?.total ?? 0;
  const masteredPageDone = masteredWords.filter((item) => item.status === "mastered").length;
  const masteredPageLearning = masteredWords.filter((item) => item.status !== "mastered").length;

  function winRate(wins: number, games: number) {
    if (!games) {
      return "0%";
    }
    return `${Math.round((wins / games) * 100)}%`;
  }

  function openResults(user: AppUserItem) {
    setResultUser(user);
    setResultMode("solo_training");
    setExpandedRounds(new Set());
  }

  function openClozeResults(user: AppUserItem) {
    setClozeUser(user);
    setClozePage(1);
    setClozeDetailItem(null);
  }

  function openWrongWords(user: AppUserItem) {
    setWrongUser(user);
    setWrongPage(1);
    setExpandedWrongKeys(new Set());
  }

  function openClozeWrongWords(user: AppUserItem) {
    setClozeWrongUser(user);
    setClozeWrongPage(1);
    setExpandedClozeWrongKeys(new Set());
  }

  function openMasteredWords(user: AppUserItem) {
    setMasteredUser(user);
    setMasteredStatus("");
    setMasteredPage(1);
  }

  function toggleRound(recordId: number) {
    setExpandedRounds((current) => {
      const next = new Set(current);
      if (next.has(recordId)) {
        next.delete(recordId);
      } else {
        next.add(recordId);
      }
      return next;
    });
  }

  function toggleModalWrongWord(item: UserWrongWordItem) {
    const key = `${item.userId}-${item.wordContent}`;
    setExpandedWrongKeys((current) => {
      const next = new Set(current);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  }

  function toggleModalClozeWrongWord(item: UserClozeWrongItem) {
    const key = `${item.userId}-${item.clozeItemId}`;
    setExpandedClozeWrongKeys((current) => {
      const next = new Set(current);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  }

  return (
    <>
      <header className="content-header">
        <div>
          <h1>用户列表</h1>
          <p>查看英语单词训练业务用户的基础信息、PK 数据和单人训练数据</p>
        </div>
        <div className="content-actions">
          <Button icon={<ReloadOutlined />} onClick={() => usersQuery.refetch()} loading={usersQuery.isFetching}>
            刷新
          </Button>
        </div>
      </header>

      <section className="table-card">
        <div className="user-list-toolbar">
          <Input
            allowClear
            prefix={<SearchOutlined />}
            placeholder="搜索用户 ID、用户名或昵称"
            value={keyword}
            onChange={(event) => {
              setKeyword(event.target.value);
              setPage(1);
            }}
          />
          <span>共 {total} 位用户</span>
        </div>

        <div className="table-scroll">
          <table className="data-table user-list-table">
            <thead>
              <tr>
                <th>序号</th>
                <th>用户名</th>
                <th>PK 段位</th>
                <th>PK 胜率</th>
                <th>训练等级</th>
                <th>训练胜率</th>
                <th>连胜</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {usersQuery.isLoading ? (
                <tr>
                  <td colSpan={8} className="empty-table-cell">
                    <Spin />
                  </td>
                </tr>
              ) : usersQuery.isError ? (
                <tr>
                  <td colSpan={8} className="empty-table-cell">
                    {usersQuery.error.message}
                  </td>
                </tr>
              ) : users.length === 0 ? (
                <tr>
                  <td colSpan={8} className="empty-table-cell">
                    暂无用户
                  </td>
                </tr>
              ) : (
                users.map((user, index) => (
                  <tr key={user.id}>
                    <td>{(page - 1) * pageSize + index + 1}</td>
                    <td>{user.username}</td>
                    <td>
                      <div className="user-stat-cell">
                        <strong>Lv. {user.rank}</strong>
                        <span>{user.exp} 经验</span>
                      </div>
                    </td>
                    <td>
                      <div className="user-stat-cell">
                        <strong>{winRate(user.totalWins, user.totalGames)}</strong>
                        <span>
                          {user.totalWins}/{user.totalGames}
                        </span>
                      </div>
                    </td>
                    <td>
                      <div className="user-stat-cell">
                        <strong>Lv. {user.trainingRank}</strong>
                        <span>{user.trainingExp} 经验</span>
                      </div>
                    </td>
                    <td>
                      <div className="user-stat-cell">
                        <strong>{winRate(user.trainingTotalWins, user.trainingTotalGames)}</strong>
                        <span>
                          {user.trainingTotalWins}/{user.trainingTotalGames}
                        </span>
                      </div>
                    </td>
                    <td>{user.currentWinStreak}</td>
                    <td>
                      <div className="user-action-buttons">
                        <Button size="small" icon={<TableOutlined />} onClick={() => openResults(user)}>
                          答题结果
                        </Button>
                        <Button size="small" icon={<HistoryOutlined />} onClick={() => openWrongWords(user)}>
                          错题集
                        </Button>
                        <Button size="small" icon={<BookOutlined />} onClick={() => openClozeResults(user)}>
                          造句结果
                        </Button>
                        <Button size="small" icon={<HistoryOutlined />} onClick={() => openClozeWrongWords(user)}>
                          造句错题
                        </Button>
                        <Button size="small" icon={<CheckCircleOutlined />} onClick={() => openMasteredWords(user)}>
                          已掌握
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        <div className="table-pagination">
          <AppPagination
            current={page}
            pageSize={pageSize}
            total={total}
            onChange={(nextPage, nextPageSize) => {
              setPage(nextPage);
              setPageSize(nextPageSize);
            }}
            totalLabel="位用户"
          />
        </div>
      </section>

      <Modal
        title={`${resultUser?.nickname || resultUser?.username || "用户"} · 答题结果`}
        open={Boolean(resultUser)}
        onCancel={() => setResultUser(null)}
        footer={null}
        width="100vw"
        className="fullscreen-results-modal"
      >
        <section className="user-results-modal">
          <div className="user-results-mode-tabs">
            <button
              type="button"
              className={resultMode === "solo_training" ? "active" : ""}
              onClick={() => {
                setResultMode("solo_training");
                setExpandedRounds(new Set());
              }}
            >
              单人训练
            </button>
            <button
              type="button"
              className={resultMode === "match" ? "active" : ""}
              onClick={() => {
                setResultMode("match");
                setExpandedRounds(new Set());
              }}
            >
              匹配答题
            </button>
          </div>

          <div className="user-results-summary">
            <div>
              <span>{resultMode === "match" ? "匹配轮次" : "训练轮次"}</span>
              <strong>{trainingRounds.length}</strong>
            </div>
            <div>
              <span>答题数</span>
              <strong>{trainingAnswerCount}</strong>
            </div>
            <div>
              <span>正确</span>
              <strong>{trainingCorrectCount}</strong>
            </div>
            <div>
              <span>准确率</span>
              <strong>{trainingAnswerCount ? Math.round((trainingCorrectCount / trainingAnswerCount) * 100) : 0}%</strong>
            </div>
            <div>
              <span>总得分</span>
              <strong>{trainingScore}</strong>
            </div>
          </div>

          {trainingResultsQuery.isLoading ? (
            <div className="empty-table-cell">
              <Spin />
            </div>
          ) : trainingResultsQuery.isError ? (
            <div className="empty-table-cell">{trainingResultsQuery.error.message}</div>
          ) : trainingRounds.length === 0 ? (
            <div className="empty-table-cell">
              今天暂无{resultMode === "match" ? "匹配答题" : "单人训练"}结果
            </div>
          ) : (
            <div className="user-results-rounds">
              {trainingRounds.map((round, index) => {
                const expanded = expandedRounds.has(round.recordId);
                const accuracy = round.totalCount ? Math.round((round.correctCount / round.totalCount) * 100) : 0;
                return (
                  <article
                    key={round.recordId}
                    className={`user-result-round ${round.correctCount === round.totalCount ? "correct" : "wrong"}`}
                  >
                    <button type="button" className="user-result-summary-row" onClick={() => toggleRound(round.recordId)}>
                      <span>第 {index + 1} 轮</span>
                      <strong>{formatFullTime(round.startTime)}</strong>
                      <span>{trainingDifficultyLabel(round)}</span>
                      <span>单词 {round.totalCount}</span>
                      <span>正确 {round.correctCount}/{round.totalCount}</span>
                      <span>准确率 {accuracy}%</span>
                      <span>得分 {round.score}</span>
                      <span>耗时 {formatSeconds(round.durationSeconds)}</span>
                      <span className="expand-text">{expanded ? "收起" : "展开"}</span>
                    </button>

                    {expanded ? (
                      <div className="user-result-detail">
                        <div className="user-result-word-grid user-result-word-head">
                          <span>序号</span>
                          <span>单词</span>
                          <span>难度</span>
                          <span>结果</span>
                          <span>得分</span>
                          <span>耗时</span>
                          <span>正确答案</span>
                        </div>
                        {round.details.map((detail, detailIndex) => (
                          <div
                            key={detail.id}
                            className={`user-result-word-grid user-result-word-row ${
                              detail.isCorrect === 1 ? "correct" : "wrong"
                            }`}
                          >
                            <span>{detailIndex + 1}</span>
                            <strong>{detail.wordContent}</strong>
                            <span>{detail.wordDifficulty || "-"}</span>
                            <span>{detail.isCorrect === 1 ? "答对" : "答错"}</span>
                            <span>{detail.score || 0}</span>
                            <span>{formatAnswerTime(detail.answerTimeMs)}</span>
                            <span>{answerText(detail, detail.correctAnswerIndex)}</span>
                          </div>
                        ))}
                      </div>
                    ) : null}
                  </article>
                );
              })}
            </div>
          )}
        </section>
      </Modal>

      <Modal
        title={`${wrongUser?.nickname || wrongUser?.username || "用户"} · 错题集`}
        open={Boolean(wrongUser)}
        onCancel={() => setWrongUser(null)}
        footer={null}
        width="100vw"
        className="fullscreen-results-modal"
      >
        <section className="user-results-modal">
          <section className="user-wrong-summary">
            <div>
              <span>错题条目</span>
              <strong>{wrongTotal}</strong>
            </div>
            <div>
              <span>当前页错误次数</span>
              <strong>{wrongTotalCount}</strong>
            </div>
            <div>
              <span>用户</span>
              <strong>{wrongUser?.username || "-"}</strong>
            </div>
          </section>

          <div className="wrong-table-panel">
            {wrongWordsQuery.isLoading ? (
              <div className="empty-table-cell">
                <Spin />
              </div>
            ) : wrongWordsQuery.isError ? (
              <div className="empty-table-cell">{wrongWordsQuery.error.message}</div>
            ) : wrongWords.length === 0 ? (
              <div className="empty-table-cell">暂无错题记录</div>
            ) : (
              <>
                <div className="wrong-table-head">
                  <span>序号</span>
                  <span>单词</span>
                  <span>错误</span>
                  <span>总答</span>
                  <span>正确率</span>
                  <span>均难度</span>
                  <span>最近难度</span>
                  <span>最近时间</span>
                  <span>操作</span>
                </div>
                <div className="wrong-table-body">
                  {wrongWords.map((item, index) => {
                    const key = `${item.userId}-${item.wordContent}`;
                    const expanded = expandedWrongKeys.has(key);
                    const accuracy =
                      item.totalAttempts > 0 ? Math.round(((item.totalAttempts - item.wrongCount) / item.totalAttempts) * 100) : 0;
                    return (
                      <article className="wrong-table-card" key={key}>
                        <button type="button" className="wrong-table-row" onClick={() => toggleModalWrongWord(item)}>
                          <span>#{(wrongPage - 1) * wrongPageSize + index + 1}</span>
                          <strong>{item.wordContent}</strong>
                          <span className="wrong-count-pill">{item.wrongCount} 次</span>
                          <span>{item.totalAttempts} 次</span>
                          <span>{accuracy}%</span>
                          <span>{item.avgDifficulty}</span>
                          <span>{wrongWordDifficultyLabel(item)}</span>
                          <span>{formatTime(item.lastWrongTime)}</span>
                          <span className="expand-text">{expanded ? "收起" : "展开"}</span>
                        </button>

                        {expanded ? (
                          <div className="wrong-table-detail">
                            <div className="wrong-detail-head">
                              <span>时间</span>
                              <span>模式/难度</span>
                              <span>词难度</span>
                              <span>耗时</span>
                              <span>正确答案</span>
                            </div>
                            {item.recentHistories.map((history) => (
                              <div className="wrong-detail-row" key={history.detailId}>
                                <span>{formatTime(history.startTime)}</span>
                                <span>{wrongHistoryDifficultyLabel(history)}</span>
                                <span>{history.wordDifficulty}</span>
                                <span>{formatAnswerTime(history.answerTimeMs)}</span>
                                <span>{answerText(history, history.correctAnswerIndex)}</span>
                              </div>
                            ))}
                          </div>
                        ) : null}
                      </article>
                    );
                  })}
                </div>
              </>
            )}
          </div>

          <div className="table-pagination">
            <AppPagination
              current={wrongPage}
              pageSize={wrongPageSize}
              total={wrongTotal}
              onChange={(nextPage, nextPageSize) => {
                setWrongPage(nextPage);
                setWrongPageSize(nextPageSize);
                setExpandedWrongKeys(new Set());
              }}
              totalLabel="个错词"
            />
          </div>
        </section>
      </Modal>

      <Modal
        title={`${clozeWrongUser?.nickname || clozeWrongUser?.username || "用户"} · 造句错题集`}
        open={Boolean(clozeWrongUser)}
        onCancel={() => setClozeWrongUser(null)}
        footer={null}
        width="100vw"
        className="fullscreen-results-modal"
      >
        <section className="user-results-modal">
          <section className="user-wrong-summary">
            <div>
              <span>错题句子</span>
              <strong>{clozeWrongTotal}</strong>
            </div>
            <div>
              <span>当前页错误次数</span>
              <strong>{clozeWrongTotalCount}</strong>
            </div>
            <div>
              <span>用户</span>
              <strong>{clozeWrongUser?.username || "-"}</strong>
            </div>
          </section>

          <div className="wrong-table-panel cloze-wrong-table-panel">
            {clozeWrongWordsQuery.isLoading ? (
              <div className="empty-table-cell">
                <Spin />
              </div>
            ) : clozeWrongWordsQuery.isError ? (
              <div className="empty-table-cell">{clozeWrongWordsQuery.error.message}</div>
            ) : clozeWrongWords.length === 0 ? (
              <div className="empty-table-cell">暂无造句错题记录</div>
            ) : (
              <>
                <div className="wrong-table-head cloze-wrong-table-head">
                  <span>序号</span>
                  <span>造句</span>
                  <span>目标词</span>
                  <span>错误</span>
                  <span>总答</span>
                  <span>正确率</span>
                  <span>来源</span>
                  <span>最近时间</span>
                  <span>操作</span>
                </div>
                <div className="wrong-table-body">
                  {clozeWrongWords.map((item, index) => {
                    const key = `${item.userId}-${item.clozeItemId}`;
                    const expanded = expandedClozeWrongKeys.has(key);
                    const accuracy =
                      item.totalAttempts > 0
                        ? Math.max(0, Math.round(((item.totalAttempts - item.wrongCount) / item.totalAttempts) * 100))
                        : 0;
                    const sentenceText = item.sentence || item.clozeSentence || "-";
                    return (
                      <article className="wrong-table-card" key={key}>
                        <button type="button" className="wrong-table-row cloze-wrong-table-row" onClick={() => toggleModalClozeWrongWord(item)}>
                          <span>#{(clozeWrongPage - 1) * clozeWrongPageSize + index + 1}</span>
                          <Tooltip title={sentenceText} placement="topLeft">
                            <strong>{sentenceText}</strong>
                          </Tooltip>
                          <span>{clozeWrongWordsLabel(item)}</span>
                          <span className="wrong-count-pill">{item.wrongCount} 次</span>
                          <span>{item.totalAttempts} 次</span>
                          <span>{accuracy}%</span>
                          <span>{clozeSourceName(item.source)}</span>
                          <span>{formatTime(item.lastWrongTime)}</span>
                          <span className="expand-text">{expanded ? "收起" : "展开"}</span>
                        </button>

                        {expanded ? (
                          <div className="wrong-table-detail">
                            <div className="wrong-detail-head cloze-wrong-detail-grid">
                              <span>时间</span>
                              <span>尝试</span>
                              <span>耗时</span>
                              <span>正确答案</span>
                            </div>
                            {item.recentHistories.map((history) => (
                                <div className="wrong-detail-row cloze-wrong-detail-grid" key={history.recordId}>
                                  <span>{formatTime(history.createTime)}</span>
                                  <span>第 {history.attemptNo} 次</span>
                                  <span>{formatAnswerTime(history.costMs)}</span>
                                  <span>{clozeAnswerText(history.expectedWords, "-")}</span>
                                </div>
                            ))}
                          </div>
                        ) : null}
                      </article>
                    );
                  })}
                </div>
              </>
            )}
          </div>

          <div className="table-pagination">
            <AppPagination
              current={clozeWrongPage}
              pageSize={clozeWrongPageSize}
              total={clozeWrongTotal}
              onChange={(nextPage, nextPageSize) => {
                setClozeWrongPage(nextPage);
                setClozeWrongPageSize(nextPageSize);
                setExpandedClozeWrongKeys(new Set());
              }}
              totalLabel="个造句错题"
            />
          </div>
        </section>
      </Modal>

      <Modal
        title={`${masteredUser?.nickname || masteredUser?.username || "用户"} · 已掌握单词`}
        open={Boolean(masteredUser)}
        onCancel={() => setMasteredUser(null)}
        footer={null}
        width="100vw"
        className="fullscreen-results-modal"
      >
        <section className="user-results-modal">
          <section className="user-wrong-summary">
            <div>
              <span>掌握记录</span>
              <strong>{masteredTotal}</strong>
            </div>
            <div>
              <span>当前页已掌握</span>
              <strong>{masteredPageDone}</strong>
            </div>
            <div>
              <span>当前页复习中</span>
              <strong>{masteredPageLearning}</strong>
            </div>
          </section>

          <div className="mastered-modal-toolbar">
            <MasteredStatusTabs
              status={masteredStatus}
              onChange={(nextStatus) => {
                setMasteredStatus(nextStatus);
                setMasteredPage(1);
              }}
            />
          </div>

          <MasteredWordsPanel
            items={masteredWords}
            loading={masteredWordsQuery.isLoading}
            errorMessage={masteredWordsQuery.isError ? masteredWordsQuery.error.message : undefined}
            page={masteredPage}
            pageSize={masteredPageSize}
            total={masteredTotal}
            showUser={false}
            emptyText="暂无已掌握单词记录"
            onPageChange={(nextPage, nextPageSize) => {
              setMasteredPage(nextPage);
              setMasteredPageSize(nextPageSize);
            }}
          />
        </section>
      </Modal>

      <Modal
        title={`${clozeUser?.nickname || clozeUser?.username || "用户"} · 造句结果`}
        open={Boolean(clozeUser)}
        onCancel={() => setClozeUser(null)}
        footer={null}
        width="100vw"
        className="fullscreen-results-modal"
      >
        <section className="user-results-modal">
          <div className="user-list-toolbar">
            <span>共 {clozeTotal} 条造句结果</span>
          </div>

          <div className="table-scroll">
            <table className="data-table cloze-result-table">
              <thead>
                <tr>
                  <th>ID</th>
                  <th>单词</th>
                  <th>来源</th>
                  <th>模型</th>
                  <th>生成句子</th>
                  <th>挖空句</th>
                  <th>生成时间</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {clozeResultsQuery.isLoading ? (
                  <tr>
                    <td colSpan={8} className="empty-table-cell">
                      <Spin />
                    </td>
                  </tr>
                ) : clozeResultsQuery.isError ? (
                  <tr>
                    <td colSpan={8} className="empty-table-cell">
                      {clozeResultsQuery.error.message}
                    </td>
                  </tr>
                ) : clozeResults.length === 0 ? (
                  <tr>
                    <td colSpan={8} className="empty-table-cell">
                      暂无造句结果
                    </td>
                  </tr>
                ) : (
                  clozeResults.map((item) => (
                    <tr key={item.id}>
                      <td>{item.id}</td>
                      <td>
                        <button type="button" className="cloze-word-button" onClick={() => setClozeDetailItem(item)}>
                          {(item.words.length > 0 ? item.words : [item.word]).join(", ")}
                        </button>
                      </td>
                      <td>{clozeResultSourceName(item)}</td>
                      <td>{item.model || "-"}</td>
                      <td>
                        <Tooltip title={item.sentence || "-"} placement="topLeft">
                          <span className="table-ellipsis-text">{item.sentence || "-"}</span>
                        </Tooltip>
                      </td>
                      <td>
                        <Tooltip title={item.clozeSentence || "-"} placement="topLeft">
                          <span className="table-ellipsis-text">{item.clozeSentence || "-"}</span>
                        </Tooltip>
                      </td>
                      <td>{formatTime(item.createTime)}</td>
                      <td>
                        <button type="button" className="table-text-action" onClick={() => setClozeDetailItem(item)}>
                          详情
                        </button>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          <div className="table-pagination">
            <AppPagination
              current={clozePage}
              pageSize={clozePageSize}
              total={clozeTotal}
              onChange={(nextPage, nextPageSize) => {
                setClozePage(nextPage);
                setClozePageSize(nextPageSize);
              }}
              totalLabel="条造句结果"
            />
          </div>
        </section>
      </Modal>

      <Modal
        title="造句结果详情"
        open={Boolean(clozeDetailItem)}
        onCancel={() => setClozeDetailItem(null)}
        footer={null}
        width={900}
      >
        {clozeDetailItem ? (
          <div className="sentence-detail-modal">
            <div className="detail-words">
              {(clozeDetailItem.words.length > 0 ? clozeDetailItem.words : [clozeDetailItem.word]).map((word) => (
                <span key={word}>{word}</span>
              ))}
            </div>
            <div className="detail-block">
              <span>生成句子</span>
              <p>{clozeDetailItem.sentence || "-"}</p>
            </div>
            <div className="detail-block">
              <span>挖空句</span>
              <p>{clozeDetailItem.clozeSentence || "-"}</p>
            </div>
            <div className="detail-block">
              <span>翻译</span>
              <p>{clozeDetailItem.translationZh || "暂无翻译"}</p>
            </div>
            <div className="detail-block">
              <span>中文解释</span>
              <p>{clozeDetailItem.explanationZh || "暂无中文解释"}</p>
            </div>
            <div className="detail-meta-grid">
              <div>
                <span>用户</span>
                <strong>{clozeDetailItem.userName || `用户 ${clozeDetailItem.userId}`}</strong>
              </div>
              <div>
                <span>来源</span>
                <strong>{clozeResultSourceName(clozeDetailItem)}</strong>
              </div>
              <div>
                <span>生成时间</span>
                <strong>{formatTime(clozeDetailItem.createTime)}</strong>
              </div>
              <div>
                <span>Provider</span>
                <strong>{clozeDetailItem.providerLabel || "-"}</strong>
              </div>
              <div>
                <span>Model</span>
                <strong>{clozeDetailItem.model || "-"}</strong>
              </div>
            </div>
          </div>
        ) : null}
      </Modal>
    </>
  );
}

function WordLibraryPage() {
  const [messageApi, contextHolder] = message.useMessage();
  const [libraryKeyword, setLibraryKeyword] = useState("");
  const [wordKeyword, setWordKeyword] = useState("");
  const [libraryPage, setLibraryPage] = useState(1);
  const [libraryPageSize, setLibraryPageSize] = useState(1000);
  const [wordPage, setWordPage] = useState(1);
  const [wordPageSize, setWordPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [selectedLibrary, setSelectedLibrary] = useState<WordLibraryItem | null>(null);
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(() => new Set());
  const [wordSortBy, setWordSortBy] = useState<WordSortBy | undefined>();
  const [wordSortOrder, setWordSortOrder] = useState<SortOrder>("asc");
  const [playingWordID, setPlayingWordID] = useState<number | null>(null);
  const [loadingAudioWordID, setLoadingAudioWordID] = useState<number | null>(null);
  const audioRef = useRef<HTMLAudioElement | null>(null);

  const librariesQuery = useQuery({
    queryKey: ["word-libraries", libraryKeyword, libraryPage, libraryPageSize],
    queryFn: () =>
      listWordLibraries({
        keyword: libraryKeyword.trim(),
        page: libraryPage,
        pageSize: libraryPageSize,
      }),
  });

  const libraries = librariesQuery.data?.list ?? [];
  const libraryTotal = librariesQuery.data?.total ?? 0;
  const libraryGroups = useMemo(() => groupWordLibraries(libraries), [libraries]);

  useEffect(() => {
    if (!selectedLibrary && libraries.length > 0) {
      setSelectedLibrary(libraries[0]);
    }
    if (selectedLibrary && libraries.length > 0 && !libraries.some((library) => library.id === selectedLibrary.id)) {
      setSelectedLibrary(libraries[0]);
      setWordPage(1);
    }
  }, [libraries, selectedLibrary]);

  useEffect(() => {
    if (libraryGroups.length === 0) {
      setExpandedGroups(new Set());
      return;
    }

    const selectedGroupKey = selectedLibrary ? libraryGroupTitle(selectedLibrary) : libraryGroups[0].key;
    setExpandedGroups((current) => {
      if (current.size > 0 && current.has(selectedGroupKey)) {
        return current;
      }
      const next = new Set(current);
      next.add(selectedGroupKey);
      return next;
    });
  }, [libraryGroups, selectedLibrary]);

  const wordsQuery = useQuery({
    queryKey: ["word-library-words", selectedLibrary?.id, wordKeyword, wordPage, wordPageSize, wordSortBy, wordSortOrder],
    queryFn: () =>
      listWordLibraryWords({
        libraryId: selectedLibrary?.id ?? 0,
        keyword: wordKeyword.trim(),
        page: wordPage,
        pageSize: wordPageSize,
        sortBy: wordSortBy,
        sortOrder: wordSortBy ? wordSortOrder : undefined,
      }),
    enabled: Boolean(selectedLibrary),
  });

  const words = wordsQuery.data?.list ?? [];
  const wordTotal = wordsQuery.data?.total ?? 0;

  useEffect(() => {
    return () => {
      const audio = audioRef.current;
      if (audio) {
        audio.pause();
        audio.src = "";
      }
      audioRef.current = null;
    };
  }, []);

  function stopWordAudio() {
    const audio = audioRef.current;
    if (audio) {
      audio.pause();
      audio.currentTime = 0;
      audio.src = "";
    }
    audioRef.current = null;
    setPlayingWordID(null);
    setLoadingAudioWordID(null);
  }

  async function handlePlayWordSentence(item: WordLibraryWordItem) {
    const audioURL = playableBestSentenceAudioURL(item.bestSentenceTtsStatus, item.bestSentenceTtsObjectUrl);
    if (!audioURL) {
      return;
    }
    if (playingWordID === item.id || loadingAudioWordID === item.id) {
      stopWordAudio();
      return;
    }

    stopWordAudio();
    const audio = new Audio(audioURL);
    audio.preload = "metadata";
    audioRef.current = audio;
    setLoadingAudioWordID(item.id);
    audio.addEventListener("playing", () => {
      if (audioRef.current === audio) {
        setLoadingAudioWordID(null);
        setPlayingWordID(item.id);
      }
    });
    audio.addEventListener("ended", () => {
      if (audioRef.current === audio) {
        stopWordAudio();
      }
    });
    audio.addEventListener("error", () => {
      if (audioRef.current === audio) {
        stopWordAudio();
        messageApi.error(`无法播放 ${item.word} 的例句语音`);
      }
    });
    try {
      await audio.play();
    } catch {
      if (audioRef.current === audio) {
        stopWordAudio();
        messageApi.error(`无法播放 ${item.word} 的例句语音`);
      }
    }
  }

  function handleRefresh() {
    librariesQuery.refetch();
    if (selectedLibrary) {
      wordsQuery.refetch();
    }
  }

  function handleSelectLibrary(library: WordLibraryItem) {
    stopWordAudio();
    setSelectedLibrary(library);
    setWordKeyword("");
    setWordPage(1);
  }

  function handleWordSort(sortBy: WordSortBy, sortOrder: SortOrder) {
    setWordSortBy(sortBy);
    setWordSortOrder(sortOrder);
    setWordPage(1);
  }

  function toggleLibraryGroup(groupKey: string) {
    setExpandedGroups((current) => {
      const next = new Set(current);
      if (next.has(groupKey)) {
        next.delete(groupKey);
      } else {
        next.add(groupKey);
      }
      return next;
    });
  }

  function renderSortableHeader(label: string, sortBy: WordSortBy) {
    const ascActive = wordSortBy === sortBy && wordSortOrder === "asc";
    const descActive = wordSortBy === sortBy && wordSortOrder === "desc";
    return (
      <span className="sortable-header">
        <span>{label}</span>
        <span className="sort-buttons">
          <button
            type="button"
            className={ascActive ? "active" : ""}
            aria-label={`${label}升序`}
            onClick={() => handleWordSort(sortBy, "asc")}
          >
            <CaretUpOutlined />
          </button>
          <button
            type="button"
            className={descActive ? "active" : ""}
            aria-label={`${label}降序`}
            onClick={() => handleWordSort(sortBy, "desc")}
          >
            <CaretDownOutlined />
          </button>
        </span>
      </span>
    );
  }

  return (
    <>
      {contextHolder}
      <header className="content-header">
        <div>
          <h1>词库单词管理</h1>
          <p>按词库查看 word_library 与 word 的一对多关系</p>
        </div>
        <div className="content-actions">
          <Button icon={<ReloadOutlined />} onClick={handleRefresh} loading={librariesQuery.isFetching || wordsQuery.isFetching}>
            刷新
          </Button>
        </div>
      </header>

      <section className="library-workspace">
        <aside className="library-list-panel">
          <div className="library-panel-header">
            <div>
              <strong>词库</strong>
              <span>{libraryTotal} 个词库</span>
            </div>
          </div>
          <Input
            allowClear
            prefix={<SearchOutlined />}
            placeholder="搜索名称、中文说明或 ID"
            value={libraryKeyword}
            onChange={(event) => {
              setLibraryKeyword(event.target.value);
              setLibraryPage(1);
            }}
          />

          <div className="library-tree">
            {librariesQuery.isLoading ? (
              <div className="cloze-loading">
                <Spin />
              </div>
            ) : libraryGroups.length > 0 ? (
              libraryGroups.map((group) => {
                const expanded = expandedGroups.has(group.key);
                return (
                  <section className="library-group" key={group.key}>
                    <button type="button" className="library-group-row" onClick={() => toggleLibraryGroup(group.key)}>
                      {expanded ? <DownOutlined /> : <RightOutlined />}
                      <span className="library-group-name">{group.title}</span>
                      <span className="library-group-meta">
                        {group.libraries.length} 册 · {group.totalWordCount} 词
                      </span>
                    </button>
                    {expanded ? (
                      <div className="library-children">
                        {group.libraries.map((library) => (
                          <button
                            type="button"
                            key={library.id}
                            className={`library-row ${selectedLibrary?.id === library.id ? "active" : ""}`}
                            onClick={() => handleSelectLibrary(library)}
                          >
                            <span className="library-name">{libraryChildTitle(library)}</span>
                            <span className="library-code">{library.libraryName}</span>
                            <span className="library-meta">{library.wordCount} 词</span>
                          </button>
                        ))}
                      </div>
                    ) : null}
                  </section>
                );
              })
            ) : librariesQuery.error instanceof Error ? (
              <div className="empty-inline">{librariesQuery.error.message}</div>
            ) : (
              <div className="empty-inline">暂无词库</div>
            )}
          </div>
        </aside>

        <section className="library-word-panel">
          <div className="library-detail-title">
            <div>
              <strong>{selectedLibrary?.libraryMeaning || selectedLibrary?.libraryName || "请选择词库"}</strong>
              <span>{selectedLibrary ? `${selectedLibrary.libraryName} · ${selectedLibrary.wordCount} 词` : "左侧选择词库后查看单词"}</span>
            </div>
            <span>{wordTotal} 条单词</span>
          </div>

          <section className="library-word-filter">
            <Input
              allowClear
              prefix={<SearchOutlined />}
              placeholder="搜索英文单词、中文释义、短语或例句"
              value={wordKeyword}
              onChange={(event) => {
                setWordKeyword(event.target.value);
                setWordPage(1);
              }}
              disabled={!selectedLibrary}
            />
          </section>

          <div className="sentence-table-scroll">
            <table className="sentence-table word-library-table">
              <thead>
                <tr>
                  <th>序号</th>
                  <th>单词</th>
                  <th>中文释义</th>
                  <th>{renderSortableHeader("难度", "difficulty")}</th>
                  <th>{renderSortableHeader("频率", "frequency")}</th>
                  <th>例句</th>
                </tr>
              </thead>
              <tbody>
                {!selectedLibrary ? (
                  <tr>
                    <td colSpan={6} className="empty-table-cell">
                      请先选择词库
                    </td>
                  </tr>
                ) : wordsQuery.isLoading ? (
                  <tr>
                    <td colSpan={6} className="empty-table-cell">
                      加载中...
                    </td>
                  </tr>
                ) : words.length > 0 ? (
                  words.map((item, index) => {
                    const hasAudio = item.bestSentenceTtsStatus === "success" && Boolean(item.bestSentenceTtsObjectUrl);
                    const isPlaying = playingWordID === item.id;
                    const isAudioLoading = loadingAudioWordID === item.id;
                    return (
                    <tr key={item.id}>
                      <td>{(wordPage - 1) * wordPageSize + index + 1}</td>
                      <td className="strong">
                        <Tooltip title={item.word} placement="topLeft">
                          <span className="table-ellipsis-text word-cell-text">{item.word}</span>
                        </Tooltip>
                      </td>
                      <td>
                        <Tooltip title={item.meaning} placement="topLeft">
                          <span className="table-ellipsis-text">{item.meaning}</span>
                        </Tooltip>
                      </td>
                      <td>{item.difficulty}</td>
                      <td>{item.frequency}</td>
                      <td>
                        <span className="word-library-sentence-cell">
                          <Tooltip title={item.sentence || "-"} placement="topLeft">
                            <span className="table-ellipsis-text">{item.sentence || "-"}</span>
                          </Tooltip>
                          <Tooltip title={hasAudio ? (isPlaying || isAudioLoading ? "停止播放" : "播放例句语音") : "暂无可播放语音"}>
                            <Button
                              aria-label={`${isPlaying || isAudioLoading ? "停止" : "播放"} ${item.word} 的例句语音`}
                              className={`word-audio-button ${isPlaying ? "playing" : ""}`}
                              disabled={!hasAudio}
                              icon={isPlaying ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
                              loading={isAudioLoading}
                              size="small"
                              type="text"
                              onClick={() => handlePlayWordSentence(item)}
                            />
                          </Tooltip>
                        </span>
                      </td>
                    </tr>
                    );
                  })
                ) : wordsQuery.error instanceof Error ? (
                  <tr>
                    <td colSpan={6} className="empty-table-cell">
                      {wordsQuery.error.message}
                    </td>
                  </tr>
                ) : (
                  <tr>
                    <td colSpan={6} className="empty-table-cell">
                      暂无单词
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          {selectedLibrary && wordTotal > 0 ? (
            <div className="cloze-result-pagination">
              <AppPagination
                current={wordPage}
                pageSize={wordPageSize}
                total={wordTotal}
                onChange={(page, pageSize) => {
                  stopWordAudio();
                  setWordPage(page);
                  setWordPageSize(pageSize);
                }}
                totalLabel="条单词"
              />
            </div>
          ) : null}
        </section>
      </section>
    </>
  );
}

function WordCleanPage() {
  const [messageApi, contextHolder] = message.useMessage();
  const queryClient = useQueryClient();
  const [keyword, setKeyword] = useState("");
  const [selectedPepDifficulty, setSelectedPepDifficulty] = useState<number | undefined>();
  const [selectedSourceGroup, setSelectedSourceGroup] = useState<string | undefined>();
  const [selectedDifficultyRange, setSelectedDifficultyRange] = useState<string | undefined>();
  const [expandedPepGroups, setExpandedPepGroups] = useState<Set<string>>(
    () => new Set([...PEP_DIFFICULTY_GROUPS, ...WORD_CLEAN_EXTRA_GROUPS].map((group) => group.title)),
  );
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [sortBy, setSortBy] = useState<WordCleanSortBy | undefined>();
  const [sortOrder, setSortOrder] = useState<SortOrder>("asc");
  const [sentenceWord, setSentenceWord] = useState<WordCleanItem | null>(null);
  const [playingCleanWordAudioTarget, setPlayingCleanWordAudioTarget] = useState<CleanWordAudioTarget | null>(null);
  const [loadingCleanWordAudioTarget, setLoadingCleanWordAudioTarget] = useState<CleanWordAudioTarget | null>(null);
  const cleanWordAudioRef = useRef<HTMLAudioElement | null>(null);

  const cleanWordsQuery = useQuery({
    queryKey: [
      "word-clean",
      keyword,
      selectedPepDifficulty,
      selectedSourceGroup,
      selectedDifficultyRange,
      page,
      pageSize,
      sortBy,
      sortOrder,
    ],
    queryFn: () => {
      const difficultyRange = WORD_CLEAN_DIFFICULTY_RANGES.find((range) => range.value === selectedDifficultyRange);
      return listWordCleanWords({
        keyword: keyword.trim(),
        pepDifficulty: selectedPepDifficulty,
        sourceGroup: selectedSourceGroup,
        difficultyMin: difficultyRange?.min,
        difficultyMax: difficultyRange?.max,
        page,
        pageSize,
        sortBy,
        sortOrder: sortBy ? sortOrder : undefined,
      });
    },
  });

  const cleanWordSentencesQuery = useQuery({
    queryKey: ["word-clean-sentences", sentenceWord?.id],
    queryFn: () => listWordCleanSentences(sentenceWord?.id ?? 0),
    enabled: Boolean(sentenceWord),
  });

  const words: WordCleanItem[] = cleanWordsQuery.data?.list ?? [];
  const total = cleanWordsQuery.data?.total ?? 0;
  const wordSentences: WordCleanSentenceItem[] = cleanWordSentencesQuery.data ?? [];

  useEffect(() => {
    return () => {
      const audio = cleanWordAudioRef.current;
      if (audio) {
        audio.pause();
        audio.src = "";
      }
      cleanWordAudioRef.current = null;
    };
  }, []);

  useEffect(() => {
    stopCleanWordAudio();
  }, [keyword, selectedPepDifficulty, selectedSourceGroup, selectedDifficultyRange, page, pageSize, sortBy, sortOrder]);

  function stopCleanWordAudio() {
    const audio = cleanWordAudioRef.current;
    if (audio) {
      audio.pause();
      audio.currentTime = 0;
      audio.src = "";
    }
    cleanWordAudioRef.current = null;
    setPlayingCleanWordAudioTarget(null);
    setLoadingCleanWordAudioTarget(null);
  }

  async function handlePlayCleanWordAudio(
    item: WordCleanItem,
    target: CleanWordAudioTarget,
    audioURL: string | null,
    errorMessage: string,
  ) {
    if (!audioURL) {
      return;
    }
    if (playingCleanWordAudioTarget === target || loadingCleanWordAudioTarget === target) {
      stopCleanWordAudio();
      return;
    }

    stopCleanWordAudio();
    const audio = new Audio(audioURL);
    audio.preload = "metadata";
    cleanWordAudioRef.current = audio;
    setLoadingCleanWordAudioTarget(target);
    audio.addEventListener("playing", () => {
      if (cleanWordAudioRef.current === audio) {
        setLoadingCleanWordAudioTarget(null);
        setPlayingCleanWordAudioTarget(target);
      }
    });
    audio.addEventListener("ended", () => {
      if (cleanWordAudioRef.current === audio) {
        stopCleanWordAudio();
      }
    });
    audio.addEventListener("error", () => {
      if (cleanWordAudioRef.current === audio) {
        stopCleanWordAudio();
        messageApi.error(errorMessage);
      }
    });
    try {
      await audio.play();
    } catch {
      if (cleanWordAudioRef.current === audio) {
        stopCleanWordAudio();
        messageApi.error(errorMessage);
      }
    }
  }

  const scoreSentencesMutation = useMutation({
    mutationFn: ({ item, overwrite }: { item: WordCleanItem; overwrite: boolean }) =>
      scoreWordCleanSentences({
        wordCleanIds: [item.id],
        limit: 50,
        overwrite,
      }),
    onSuccess: (result, variables) => {
      messageApi.success(result.message || `已评分 ${result.scoredCount} 条`);
      const bestItem = (result.bestItems ?? []).find((item) => item.wordCleanId === variables.item.id);
      if (bestItem) {
        setSentenceWord((current) =>
          current?.id === variables.item.id
            ? {
                ...current,
                bestSentenceId: bestItem.id,
                bestSourceSentenceId: bestItem.sourceSentenceId,
                bestSourceModelName: bestItem.sourceModelName,
                bestSentence: bestItem.sentence,
                bestSentenceTranslation: bestItem.sentenceTranslation,
                bestSentenceScore: bestItem.score,
                bestSentenceScoreReason: bestItem.scoreReason,
                bestSentenceScoreModelName: bestItem.scoreModelName,
                bestSentenceScoredAt: bestItem.scoredAt ?? null,
                bestSentenceTtsStatus: bestItem.ttsStatus,
                bestSentenceTtsBucket: bestItem.ttsBucket,
                bestSentenceTtsObjectKey: bestItem.ttsObjectKey,
                bestSentenceTtsObjectUrl: bestItem.ttsObjectUrl,
              }
            : current,
        );
      }
      queryClient.invalidateQueries({ queryKey: ["word-clean"] });
      queryClient.invalidateQueries({ queryKey: ["word-clean-sentences", variables.item.id] });
    },
    onError: (error) => {
      messageApi.error(error instanceof Error ? error.message : "评分失败");
    },
  });

  function scoreWordCleanItem(item: WordCleanItem, overwrite: boolean) {
    scoreSentencesMutation.mutate({ item, overwrite });
  }

  function handleScoreWordCleanItem(item: WordCleanItem) {
    if (typeof item.bestSentenceScore === "number") {
      Modal.confirm({
        title: "已有评分结果",
        content: `当前最佳句子 ${item.bestSentenceScore} 分，来自 ${item.bestSourceModelName || "未知模型"}。是否再次评分？`,
        okText: "再次评分",
        cancelText: "取消",
        onOk: () => scoreWordCleanItem(item, true),
      });
      return;
    }
    scoreWordCleanItem(item, false);
  }

  function handleSort(nextSortBy: WordCleanSortBy, nextSortOrder: SortOrder) {
    setSortBy(nextSortBy);
    setSortOrder(nextSortOrder);
    setPage(1);
  }

  function renderSortableHeader(label: string, nextSortBy: WordCleanSortBy) {
    const ascActive = sortBy === nextSortBy && sortOrder === "asc";
    const descActive = sortBy === nextSortBy && sortOrder === "desc";
    return (
      <span className="sortable-header">
        <span>{label}</span>
        <span className="sort-buttons">
          <button
            type="button"
            className={ascActive ? "active" : ""}
            aria-label={`${label}升序`}
            onClick={() => handleSort(nextSortBy, "asc")}
          >
            <CaretUpOutlined />
          </button>
          <button
            type="button"
            className={descActive ? "active" : ""}
            aria-label={`${label}降序`}
            onClick={() => handleSort(nextSortBy, "desc")}
          >
            <CaretDownOutlined />
          </button>
        </span>
      </span>
    );
  }

  function togglePepGroup(groupTitle: string) {
    setExpandedPepGroups((current) => {
      const next = new Set(current);
      if (next.has(groupTitle)) {
        next.delete(groupTitle);
      } else {
        next.add(groupTitle);
      }
      return next;
    });
  }

  return (
    <>
      {contextHolder}
      <header className="content-header">
        <div>
          <h1>去重单词表</h1>
          <p>查看 word_clean 中去重、去词组后的单词练习基础数据</p>
        </div>
        <div className="content-actions">
          <Button
            icon={<ReloadOutlined />}
            onClick={() => {
              stopCleanWordAudio();
              cleanWordsQuery.refetch();
            }}
            loading={cleanWordsQuery.isFetching}
          >
            刷新
          </Button>
        </div>
      </header>

      <section className="clean-word-panel">
        <div className="library-detail-title">
          <div>
            <strong>单词基础表</strong>
            <span>只保留单词、中文释义、难度、频率和例句</span>
          </div>
          <span>{total} 条单词</span>
        </div>

        <section className="word-clean-layout">
          <aside className="word-clean-level-panel">
            <div className="word-clean-level-groups">
              {PEP_DIFFICULTY_GROUPS.map((group) => (
                <section className="word-clean-level-group" key={group.title}>
                  <button
                    type="button"
                    className="word-clean-level-group-toggle"
                    aria-expanded={expandedPepGroups.has(group.title)}
                    onClick={() => togglePepGroup(group.title)}
                  >
                    {expandedPepGroups.has(group.title) ? <DownOutlined /> : <RightOutlined />}
                    <span>{group.title}</span>
                  </button>
                  {expandedPepGroups.has(group.title) ? (
                    <div className="word-clean-level-items">
                      {group.items.map((item) => (
                        <button
                          type="button"
                          className={selectedPepDifficulty === item.value ? "active" : ""}
                          key={item.value}
                          onClick={() => {
                            setSelectedPepDifficulty(item.value);
                            setSelectedSourceGroup(undefined);
                            setSortBy("pepDifficulty");
                            setSortOrder("asc");
                            setPage(1);
                          }}
                        >
                          <span>{item.label}</span>
                          <small>{item.count}词</small>
                        </button>
                      ))}
                    </div>
                  ) : null}
                </section>
              ))}
              {WORD_CLEAN_EXTRA_GROUPS.map((group) => (
                <section className="word-clean-level-group" key={group.title}>
                  <button
                    type="button"
                    className="word-clean-level-group-toggle"
                    aria-expanded={expandedPepGroups.has(group.title)}
                    onClick={() => togglePepGroup(group.title)}
                  >
                    {expandedPepGroups.has(group.title) ? <DownOutlined /> : <RightOutlined />}
                    <span>{group.title}</span>
                  </button>
                  {expandedPepGroups.has(group.title) ? (
                    <div className="word-clean-level-items">
                      {group.items.map((item) => (
                        <button
                          type="button"
                          className={selectedSourceGroup === item.value ? "active" : ""}
                          key={item.value}
                          onClick={() => {
                            setSelectedSourceGroup(item.value);
                            setSelectedPepDifficulty(undefined);
                            setSortBy("sourceDifficulty");
                            setSortOrder("asc");
                            setPage(1);
                          }}
                        >
                          <span>{item.label}</span>
                          <small>{item.count}词</small>
                        </button>
                      ))}
                    </div>
                  ) : null}
                </section>
              ))}
            </div>
          </aside>

          <section className="word-clean-main">
            <section className="library-word-filter">
              <Input
                allowClear
                prefix={<SearchOutlined />}
                placeholder="搜索英文单词、中文释义、例句或人教标签"
                value={keyword}
                onChange={(event) => {
                  setKeyword(event.target.value);
                  setPage(1);
                }}
              />
              <Select
                allowClear
                className="difficulty-range-select"
                placeholder="难度阶梯"
                value={selectedDifficultyRange}
                onChange={(value) => {
                  setSelectedDifficultyRange(value);
                  setPage(1);
                }}
                options={WORD_CLEAN_DIFFICULTY_RANGES.map((range) => ({
                  value: range.value,
                  label: `${range.label} · ${range.count}词`,
                }))}
              />
            </section>

            <div className="sentence-table-scroll">
              <table className="sentence-table word-library-table word-clean-table">
            <thead>
              <tr>
                <th>序号</th>
                <th>单词</th>
                <th>中文释义</th>
                <th>{renderSortableHeader("难度", "difficulty")}</th>
                <th>{renderSortableHeader("频率", "frequency")}</th>
                <th>{renderSortableHeader("来源难度", "sourceDifficulty")}</th>
                <th>例句</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {cleanWordsQuery.isLoading ? (
                <tr>
                  <td colSpan={8} className="empty-table-cell">
                    加载中...
                  </td>
                </tr>
              ) : words.length > 0 ? (
                words.map((item, index) => {
                  const displaySentence = item.bestSentence || item.sentence || "-";
                  const wordAudioURL = playableTTSAudioURL(item.wordTtsStatus, item.wordTtsObjectUrl);
                  const sentenceAudioURL = playableBestSentenceAudioURL(
                    item.bestSentenceTtsStatus,
                    item.bestSentenceTtsObjectUrl,
                  );
                  const wordAudioTarget = `word:${item.id}` as const;
                  const sentenceAudioTarget = `sentence:${item.id}` as const;
                  const isWordAudioPlaying = playingCleanWordAudioTarget === wordAudioTarget;
                  const isWordAudioLoading = loadingCleanWordAudioTarget === wordAudioTarget;
                  const isSentenceAudioPlaying = playingCleanWordAudioTarget === sentenceAudioTarget;
                  const isSentenceAudioLoading = loadingCleanWordAudioTarget === sentenceAudioTarget;
                  const scoreTooltip =
                    typeof item.bestSentenceScore === "number"
                      ? `${item.bestSentenceScore} 分 · ${item.bestSourceModelName || "未知模型"} · ${formatTime(
                          item.bestSentenceScoredAt ?? undefined,
                        )}${item.bestSentenceScoreReason ? ` · ${item.bestSentenceScoreReason}` : ""}`
                      : "尚未评分";
                  const isScoring =
                    scoreSentencesMutation.isPending && scoreSentencesMutation.variables?.item.id === item.id;
                  return (
                    <tr key={item.id}>
                      <td>{(page - 1) * pageSize + index + 1}</td>
                      <td className="strong">
                        <span className="word-cell-content">
                          <Tooltip title={item.word} placement="topLeft">
                            <span className="table-ellipsis-text word-cell-text">{item.word}</span>
                          </Tooltip>
                          <Tooltip
                            title={
                              wordAudioURL
                                ? isWordAudioPlaying || isWordAudioLoading
                                  ? "停止播放"
                                  : "播放单词发音"
                                : "暂无可播放发音"
                            }
                          >
                            <Button
                              aria-label={`${isWordAudioPlaying || isWordAudioLoading ? "停止" : "播放"} ${item.word} 的单词发音`}
                              className={`word-audio-button ${isWordAudioPlaying ? "playing" : ""}`}
                              disabled={!wordAudioURL}
                              icon={isWordAudioPlaying ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
                              loading={isWordAudioLoading}
                              size="small"
                              type="text"
                              onClick={() =>
                                handlePlayCleanWordAudio(
                                  item,
                                  wordAudioTarget,
                                  wordAudioURL,
                                  `无法播放 ${item.word} 的单词发音`,
                                )
                              }
                            />
                          </Tooltip>
                          <Tooltip title="查看大模型造句结果">
                            <Button
                              aria-label={`查看 ${item.word} 的大模型造句结果`}
                              className="word-sentence-button"
                              icon={<HistoryOutlined />}
                              size="small"
                              type="text"
                              onClick={() => setSentenceWord(item)}
                            />
                          </Tooltip>
                        </span>
                      </td>
                      <td>
                        <Tooltip title={item.meaning} placement="topLeft">
                          <span className="table-ellipsis-text">{item.meaning}</span>
                        </Tooltip>
                      </td>
                      <td>{item.difficulty}</td>
                      <td>{item.frequency}</td>
                      <td>
                        <Tooltip title={item.sourceLabel || "-"} placement="topLeft">
                          <span className="table-ellipsis-text">
                            {item.sourceDifficulty ? `${item.sourceDifficulty}. ${item.sourceLabel}` : "-"}
                          </span>
                        </Tooltip>
                      </td>
                      <td>
                        <span className="word-clean-sentence-cell">
                          <Tooltip title={displaySentence} placement="topLeft">
                            <span className="table-ellipsis-text">{displaySentence}</span>
                          </Tooltip>
                          <Tooltip
                            title={
                              sentenceAudioURL
                                ? isSentenceAudioPlaying || isSentenceAudioLoading
                                  ? "停止播放"
                                  : "播放例句语音"
                                : "暂无可播放语音"
                            }
                          >
                            <Button
                              aria-label={`${isSentenceAudioPlaying || isSentenceAudioLoading ? "停止" : "播放"} ${item.word} 的例句语音`}
                              className={`word-audio-button ${isSentenceAudioPlaying ? "playing" : ""}`}
                              disabled={!sentenceAudioURL}
                              icon={isSentenceAudioPlaying ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
                              loading={isSentenceAudioLoading}
                              size="small"
                              type="text"
                              onClick={() =>
                                handlePlayCleanWordAudio(
                                  item,
                                  sentenceAudioTarget,
                                  sentenceAudioURL,
                                  `无法播放 ${item.word} 的例句语音`,
                                )
                              }
                            />
                          </Tooltip>
                          {typeof item.bestSentenceScore === "number" ? (
                            <Tooltip title={scoreTooltip}>
                              <span className="best-sentence-score">{item.bestSentenceScore}</span>
                            </Tooltip>
                          ) : null}
                        </span>
                      </td>
                      <td>
                        <span className="word-clean-actions">
                          <Button
                            size="small"
                            type="link"
                            loading={isScoring}
                            onClick={() => handleScoreWordCleanItem(item)}
                          >
                            评分
                          </Button>
                          <Button size="small" type="link" onClick={() => setSentenceWord(item)}>
                            结果
                          </Button>
                        </span>
                      </td>
                    </tr>
                  );
                })
              ) : cleanWordsQuery.error instanceof Error ? (
                <tr>
                  <td colSpan={8} className="empty-table-cell">
                    {cleanWordsQuery.error.message}
                  </td>
                </tr>
              ) : (
                <tr>
                  <td colSpan={8} className="empty-table-cell">
                    暂无单词
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        {total > 0 ? (
          <div className="cloze-result-pagination">
            <AppPagination
              current={page}
              pageSize={pageSize}
              total={total}
              onChange={(nextPage, nextPageSize) => {
                setPage(nextPage);
                setPageSize(nextPageSize);
              }}
              totalLabel="条单词"
            />
          </div>
        ) : null}
          </section>
        </section>
      </section>

      <Modal
        destroyOnClose
        footer={null}
        open={Boolean(sentenceWord)}
        title="大模型造句结果"
        width={760}
        onCancel={() => setSentenceWord(null)}
      >
        <section className="word-sentence-modal">
          <div className="word-sentence-summary">
            <strong>{sentenceWord?.word}</strong>
            <span>{sentenceWord?.meaning || "-"}</span>
          </div>
          <div className="word-sentence-actions">
            <Button
              icon={<CloudSyncOutlined />}
              loading={scoreSentencesMutation.isPending && scoreSentencesMutation.variables?.item.id === sentenceWord?.id}
              disabled={!sentenceWord || wordSentences.length === 0}
              onClick={() => {
                if (sentenceWord) {
                  handleScoreWordCleanItem(sentenceWord);
                }
              }}
            >
              评分当前单词
            </Button>
          </div>
          {cleanWordSentencesQuery.isLoading ? (
            <div className="word-sentence-loading">
              <Spin />
            </div>
          ) : cleanWordSentencesQuery.error instanceof Error ? (
            <div className="empty-table-cell">{cleanWordSentencesQuery.error.message}</div>
          ) : wordSentences.length > 0 ? (
            <div className="word-sentence-list">
              {wordSentences.map((item) => (
                <article className="word-sentence-row" key={item.id}>
                  <div className="word-sentence-row-header">
                    <span>{item.modelName}</span>
                    {typeof item.score === "number" ? (
                      <Tooltip
                        title={
                          item.scoreReason
                            ? `${item.scoreReason}${item.scoreModelName ? ` · ${item.scoreModelName}` : ""}`
                            : "暂无评分说明"
                        }
                      >
                        <strong>{item.score}</strong>
                      </Tooltip>
                    ) : (
                      <small>未评分</small>
                    )}
                  </div>
                  <p>{item.sentence || "-"}</p>
                  <p className="word-sentence-translation">{item.sentenceTranslation || "-"}</p>
                  {item.scoreReason ? <p className="word-sentence-score-reason">{item.scoreReason}</p> : null}
                </article>
              ))}
            </div>
          ) : (
            <div className="empty-table-cell">暂无造句结果</div>
          )}
        </section>
      </Modal>
    </>
  );
}


function TTSConfigPage() {
  const [messageApi, contextHolder] = message.useMessage();
  const queryClient = useQueryClient();
  const configQuery = useQuery({
    queryKey: ["tts-config"],
    queryFn: getTTSConfig,
  });
  const [draft, setDraft] = useState<TTSConfig | null>(null);
  const [selectedIndex, setSelectedIndex] = useState(0);

  useEffect(() => {
    if (!configQuery.data) {
      return;
    }
    if (configQuery.data.providers.length > 0) {
      setDraft(configQuery.data);
    } else {
      const provider = createDefaultTTSProvider();
      setDraft({ active: provider.id, providers: [provider] });
    }
    setSelectedIndex(0);
  }, [configQuery.data]);

  useEffect(() => {
    if (draft && selectedIndex >= draft.providers.length) {
      setSelectedIndex(Math.max(0, draft.providers.length - 1));
    }
  }, [draft, selectedIndex]);

  const saveMutation = useMutation({
    mutationFn: saveTTSConfig,
    onSuccess: (data) => {
      queryClient.setQueryData(["tts-config"], data);
      setDraft(data);
      const activeIndex = data.providers.findIndex((provider) => provider.id === data.active);
      setSelectedIndex(activeIndex >= 0 ? activeIndex : 0);
      messageApi.success("TTS 配置保存成功");
    },
    onError: (error) => {
      messageApi.error(error instanceof Error ? error.message : "TTS 配置保存失败");
    },
  });

  const selectedProvider = draft?.providers[selectedIndex];

  function updateSelectedProvider(patch: Partial<TTSProviderConfig>) {
    if (!draft || !selectedProvider) {
      return;
    }
    if (patch.enabled === false && draft.active === selectedProvider.id) {
      messageApi.warning("请先将其他启用配置设为默认，再停用当前配置");
      return;
    }

    const previousID = selectedProvider.id;
    const providers = draft.providers.map((provider, index) =>
      index === selectedIndex ? { ...provider, ...patch } : provider,
    );
    const active = patch.id !== undefined && draft.active === previousID ? patch.id : draft.active;
    setDraft({ active, providers });
  }

  function handleAddProvider() {
    const provider = createDefaultTTSProvider(draft?.providers.map((item) => item.id));
    const nextIndex = draft?.providers.length ?? 0;
    setDraft((current) => {
      if (!current) {
        return { active: provider.id, providers: [provider] };
      }
      return {
        active: current.active || provider.id,
        providers: [...current.providers, provider],
      };
    });
    setSelectedIndex(nextIndex);
  }

  function handleDeleteProvider() {
    if (!draft || !selectedProvider) {
      return;
    }
    if (draft.providers.length <= 1) {
      messageApi.warning("请至少保留一个 TTS 配置");
      return;
    }
    if (draft.active === selectedProvider.id) {
      messageApi.warning("请先将其他启用配置设为默认，再删除当前配置");
      return;
    }

    const providers = draft.providers.filter((_, index) => index !== selectedIndex);
    setDraft({ ...draft, providers });
    setSelectedIndex(Math.max(0, selectedIndex - 1));
  }

  function handleSetActive() {
    if (!draft || !selectedProvider?.id.trim()) {
      messageApi.warning("请先填写配置 ID");
      return;
    }
    const providers = draft.providers.map((provider, index) =>
      index === selectedIndex ? { ...provider, enabled: true } : provider,
    );
    setDraft({ active: selectedProvider.id, providers });
  }

  function handleSave() {
    if (!draft) {
      return;
    }
    const error = validateTTSConfig(draft);
    if (error) {
      messageApi.warning(error);
      return;
    }
    saveMutation.mutate(draft);
  }

  function useDefaultDraft() {
    const provider = createDefaultTTSProvider();
    setDraft({ active: provider.id, providers: [provider] });
    setSelectedIndex(0);
  }

  const showLoading = configQuery.isLoading && !draft;

  return (
    <>
      {contextHolder}
      <header className="content-header">
        <div>
          <h1>TTS 模型配置</h1>
          <p>配置 Word Agent 生成 Xiaomi MiMo TTS 语音时使用的连接信息</p>
        </div>
        <div className="content-actions">
          <Button icon={<PlusOutlined />} onClick={handleAddProvider}>
            新增配置
          </Button>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => configQuery.refetch()}
            loading={configQuery.isFetching && !saveMutation.isPending}
          >
            刷新
          </Button>
          <Button type="primary" icon={<SaveOutlined />} onClick={handleSave} loading={saveMutation.isPending}>
            保存配置
          </Button>
        </div>
      </header>

      {showLoading ? (
        <div className="loading-area">
          <Spin />
        </div>
      ) : !draft ? (
        <div className="empty-state">
          <strong>暂时无法读取 TTS 配置</strong>
          <span>{configQuery.error instanceof Error ? configQuery.error.message : "请确认 Go server 已启动"}</span>
          <Button onClick={useDefaultDraft}>使用 Xiaomi MiMo 默认草稿</Button>
        </div>
      ) : (
        <section className="config-panel">
          <aside className="provider-list-panel">
            <div className="provider-list-title">
              <span>配置列表</span>
              <em>{draft.providers.length} 项</em>
            </div>
            <div className="provider-buttons">
              {draft.providers.map((provider, index) => (
                <button
                  type="button"
                  key={`${provider.id || "empty"}-${index}`}
                  className={`provider-row ${index === selectedIndex ? "active" : ""}`}
                  onClick={() => setSelectedIndex(index)}
                >
                  <span className="provider-name">{provider.label || provider.id || "未命名配置"}</span>
                  <span className="provider-meta">{provider.model || provider.type}</span>
                  {draft.active === provider.id && provider.id ? <CheckCircleOutlined /> : null}
                </button>
              ))}
            </div>
          </aside>

          <section className="provider-form-panel">
            {selectedProvider ? (
              <>
                <div className="provider-form-title">
                  <div>
                    <strong>{selectedProvider.label || selectedProvider.id || "未命名配置"}</strong>
                    <span>{draft.active === selectedProvider.id ? "默认 TTS 配置" : "可切换 TTS 配置"}</span>
                  </div>
                  <Button icon={<CheckCircleOutlined />} onClick={handleSetActive}>
                    设为默认
                  </Button>
                </div>

                <div className="config-grid">
                  <label className="config-field">
                    <span>配置 ID</span>
                    <Input
                      placeholder="xiaomi-mimo-tts"
                      value={selectedProvider.id}
                      onChange={(event) => updateSelectedProvider({ id: event.target.value })}
                    />
                  </label>
                  <label className="config-field">
                    <span>显示名称</span>
                    <Input
                      placeholder="Xiaomi MiMo TTS"
                      value={selectedProvider.label}
                      onChange={(event) => updateSelectedProvider({ label: event.target.value })}
                    />
                  </label>
                  <label className="config-field">
                    <span>接口类型</span>
                    <Select
                      options={[{ label: "Xiaomi MiMo TTS", value: "mimo-tts" }]}
                      value={selectedProvider.type}
                      onChange={(value) => updateSelectedProvider({ type: value })}
                    />
                  </label>
                  <label className="config-field">
                    <span>启用配置</span>
                    <Switch
                      checked={selectedProvider.enabled}
                      checkedChildren="启用"
                      unCheckedChildren="停用"
                      onChange={(enabled) => updateSelectedProvider({ enabled })}
                    />
                  </label>
                  <label className="config-field wide">
                    <span>Base URL</span>
                    <Input
                      placeholder="https://api.xiaomimimo.com/v1"
                      value={selectedProvider.base_url}
                      onChange={(event) => updateSelectedProvider({ base_url: event.target.value })}
                    />
                  </label>
                  <label className="config-field wide">
                    <span>
                      API Key
                      {selectedProvider.api_key_configured ? " · API Key 已配置" : ""}
                    </span>
                    <Input.Password
                      autoComplete="new-password"
                      placeholder={selectedProvider.api_key_configured ? "留空表示保持现有 API Key" : "请输入 API Key"}
                      value={selectedProvider.api_key}
                      onChange={(event) => updateSelectedProvider({ api_key: event.target.value })}
                    />
                  </label>
                  <label className="config-field">
                    <span>模型名称</span>
                    <Input
                      placeholder="mimo-v2.5-tts"
                      value={selectedProvider.model}
                      onChange={(event) => updateSelectedProvider({ model: event.target.value })}
                    />
                  </label>
                  <label className="config-field">
                    <span>默认音色</span>
                    <Input
                      placeholder="Chloe"
                      value={selectedProvider.voice}
                      onChange={(event) => updateSelectedProvider({ voice: event.target.value })}
                    />
                  </label>
                </div>

                <div className="config-footer">
                  <span>保存后写入独立 TTS 配置表，Word Agent 下一次生成语音时立即读取。</span>
                  <Button danger icon={<DeleteOutlined />} onClick={handleDeleteProvider}>
                    删除配置
                  </Button>
                </div>
              </>
            ) : (
              <div className="empty-state">
                <strong>还没有 TTS 配置</strong>
                <Button icon={<PlusOutlined />} onClick={handleAddProvider}>
                  新增配置
                </Button>
              </div>
            )}
          </section>
        </section>
      )}
    </>
  );
}

interface TrackerAppProps {
  authenticated: boolean;
  onLogin: () => void;
  onLogout: () => void;
}

function TrackerApp({ authenticated, onLogin, onLogout }: TrackerAppProps) {
  const [activePage, setActivePage] = useState<PageKey>("runs");
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [openNavGroups, setOpenNavGroups] = useState({
    wordAgent: true,
    users: true,
    library: true,
  });

  function toggleNavGroup(group: keyof typeof openNavGroups) {
    setOpenNavGroups((current) => ({
      ...current,
      [group]: !current[group],
    }));
  }

  function renderPage() {
    if (activePage === "runs") {
      return <RecordsPage />;
    }
    if (activePage === "sentence") {
      return <SentenceTaskPage />;
    }
    if (activePage === "cloze-results") {
      return <ClozeResultsPage />;
    }
    if (activePage === "ai-config") {
      return <ExecutionConfigPage authenticated={authenticated} />;
    }
    if (activePage === "tts-config") {
      return <TTSConfigPage />;
    }
    if (activePage === "users") {
      return <UserListPage />;
    }
    if (activePage === "user-wrong-words") {
      return <UserWrongWordsPage />;
    }
    if (activePage === "user-cloze-wrong-words") {
      return <UserClozeWrongWordsPage />;
    }
    if (activePage === "user-mastered-words") {
      return <UserMasteredWordsPage />;
    }
    if (activePage === "word-library") {
      return <WordLibraryPage />;
    }
    if (activePage === "word-clean") {
      return <WordCleanPage />;
    }
    return <RecordsPage />;
  }

  return (
    <main className={`admin-shell ${sidebarCollapsed ? "sidebar-collapsed" : ""}`}>
      <aside className="sidebar">
        <div className="brand">
          <div className="brand-mark">WA</div>
          <div className="brand-copy">
            <strong>Word Agent</strong>
            <span>数据追踪后台</span>
          </div>
          <button
            type="button"
            className="sidebar-toggle"
            aria-label={sidebarCollapsed ? "展开菜单" : "收起菜单"}
            onClick={() => setSidebarCollapsed((collapsed) => !collapsed)}
          >
            {sidebarCollapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
          </button>
        </div>

        <nav className={`nav-group ${openNavGroups.wordAgent ? "open" : "closed"}`}>
          <button
            type="button"
            className="nav-parent"
            aria-expanded={openNavGroups.wordAgent}
            onClick={() => toggleNavGroup("wordAgent")}
          >
            <DatabaseOutlined />
            <span>Word Agent</span>
            <DownOutlined />
          </button>
          {openNavGroups.wordAgent ? (
            <>
              <button
                type="button"
                className={`nav-item ${activePage === "runs" ? "active" : ""}`}
                onClick={() => setActivePage("runs")}
              >
                <TableOutlined />
                <span>执行记录</span>
              </button>
              <button
                type="button"
                className={`nav-item ${activePage === "sentence" ? "active" : ""}`}
                onClick={() => setActivePage("sentence")}
              >
                <EditOutlined />
                <span>造句任务</span>
              </button>
              <button
                type="button"
                className={`nav-item ${activePage === "cloze-results" ? "active" : ""}`}
                onClick={() => setActivePage("cloze-results")}
              >
                <UserOutlined />
                <span>用户造句结果</span>
              </button>
              <button
                type="button"
                className={`nav-item ${activePage === "ai-config" ? "active" : ""}`}
                onClick={() => setActivePage("ai-config")}
              >
                <SettingOutlined />
                <span>模型配置</span>
              </button>
              <button
                type="button"
                className={`nav-item ${activePage === "tts-config" ? "active" : ""}`}
                onClick={() => setActivePage("tts-config")}
              >
                <AudioOutlined />
                <span>TTS 模型配置</span>
              </button>
            </>
          ) : null}
        </nav>

        <nav className={`nav-group ${openNavGroups.users ? "open" : "closed"}`}>
          <button
            type="button"
            className="nav-parent"
            aria-expanded={openNavGroups.users}
            onClick={() => toggleNavGroup("users")}
          >
            <UserOutlined />
            <span>用户管理</span>
            <DownOutlined />
          </button>
          {openNavGroups.users ? (
            <>
              <button
                type="button"
                className={`nav-item ${activePage === "users" ? "active" : ""}`}
                onClick={() => setActivePage("users")}
              >
                <TableOutlined />
                <span>用户列表</span>
              </button>
              <button
                type="button"
                className={`nav-item ${activePage === "user-wrong-words" ? "active" : ""}`}
                onClick={() => setActivePage("user-wrong-words")}
              >
                <HistoryOutlined />
                <span>用户错题集</span>
              </button>
              <button
                type="button"
                className={`nav-item ${activePage === "user-cloze-wrong-words" ? "active" : ""}`}
                onClick={() => setActivePage("user-cloze-wrong-words")}
              >
                <BookOutlined />
                <span>用户造句错题集</span>
              </button>
              <button
                type="button"
                className={`nav-item ${activePage === "user-mastered-words" ? "active" : ""}`}
                onClick={() => setActivePage("user-mastered-words")}
              >
                <CheckCircleOutlined />
                <span>用户已掌握单词</span>
              </button>
            </>
          ) : null}
        </nav>

        <nav className={`nav-group ${openNavGroups.library ? "open" : "closed"}`}>
          <button
            type="button"
            className="nav-parent"
            aria-expanded={openNavGroups.library}
            onClick={() => toggleNavGroup("library")}
          >
            <BookOutlined />
            <span>词库管理</span>
            <DownOutlined />
          </button>
          {openNavGroups.library ? (
            <>
              <button
                type="button"
                className={`nav-item ${activePage === "word-library" ? "active" : ""}`}
                onClick={() => setActivePage("word-library")}
              >
                <TableOutlined />
                <span>词库单词管理</span>
              </button>
              <button
                type="button"
                className={`nav-item ${activePage === "word-clean" ? "active" : ""}`}
                onClick={() => setActivePage("word-clean")}
              >
                <TableOutlined />
                <span>去重单词表</span>
              </button>
            </>
          ) : null}
        </nav>

        <div className="sidebar-session">
          <button type="button" className="session-button" onClick={authenticated ? onLogout : onLogin}>
            {authenticated ? <LogoutOutlined /> : <LoginOutlined />}
            <span>{authenticated ? "退出登录" : "管理员登录"}</span>
          </button>
        </div>
      </aside>

      <section className="content-area">
        <div className="content-card">
          {renderPage()}
        </div>
      </section>
    </main>
  );
}

interface LoginPageProps {
  onSuccess: () => void;
  onCancel: () => void;
}

function LoginPage({ onSuccess, onCancel }: LoginPageProps) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const mountedRef = useRef(true);
  const submitSequenceRef = useRef(0);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      submitSequenceRef.current += 1;
      cancelPendingLoginAttempt();
    };
  }, []);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!username.trim() || !password) {
      setError("请输入用户名和密码");
      return;
    }
    const submitSequence = submitSequenceRef.current + 1;
    submitSequenceRef.current = submitSequence;
    setSubmitting(true);
    setError("");
    try {
      await login(username.trim(), password);
      if (!mountedRef.current || submitSequenceRef.current !== submitSequence) {
        return;
      }
      setPassword("");
      onSuccess();
    } catch (nextError) {
      if (
        nextError instanceof LoginAttemptCancelledError
        || !mountedRef.current
        || submitSequenceRef.current !== submitSequence
      ) {
        return;
      }
      setError(nextError instanceof Error ? nextError.message : "登录失败");
    } finally {
      if (mountedRef.current && submitSequenceRef.current === submitSequence) {
        setSubmitting(false);
      }
    }
  }

  function handleCancel() {
    submitSequenceRef.current += 1;
    cancelPendingLoginAttempt();
    onCancel();
  }

  return (
    <div className="login-overlay" role="dialog" aria-modal="true" aria-label="管理员登录">
      <main className="login-shell">
        <form className="login-card" onSubmit={handleSubmit}>
          <div className="login-mark"><LockOutlined /></div>
          <h1>管理员登录</h1>
          <p>登录后可保存模型与 TTS 配置；只读数据无需登录。</p>
          <label>
            <span>用户名</span>
            <Input autoComplete="username" value={username} onChange={(event) => setUsername(event.target.value)} />
          </label>
          <label>
            <span>密码</span>
            <Input.Password
              autoComplete="current-password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
            />
          </label>
          {error ? <div className="login-error">{error}</div> : null}
          <Button type="primary" htmlType="submit" loading={submitting} block>
            登录
          </Button>
          <Button type="text" onClick={handleCancel} block>
            继续只读浏览
          </Button>
        </form>
      </main>
    </div>
  );
}

export default function App() {
  const queryClient = useQueryClient();
  const [authenticated, setAuthenticated] = useState(() => Boolean(getAuthToken()));
  const [loginRequired, setLoginRequired] = useState(false);

  useEffect(() => {
    const requireLogin = () => {
      queryClient.removeQueries({ queryKey: ["execution-config"] });
      setAuthenticated(false);
      setLoginRequired(true);
    };
    authEvents.addEventListener(AUTH_REQUIRED_EVENT, requireLogin);
    return () => authEvents.removeEventListener(AUTH_REQUIRED_EVENT, requireLogin);
  }, [queryClient]);

  function handleLogout() {
    clearAuthSession();
    queryClient.removeQueries({ queryKey: ["execution-config"] });
    setAuthenticated(false);
  }

  return (
    <ConfigProvider
      theme={{
        algorithm: theme.defaultAlgorithm,
        token: {
          borderRadius: 6,
          colorPrimary: "#4354e8",
          fontFamily:
            "Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, Segoe UI, sans-serif",
        },
        components: {
          Button: {
            controlHeight: 32,
          },
        },
      }}
    >
      <>
        <TrackerApp
          authenticated={authenticated}
          onLogin={() => setLoginRequired(true)}
          onLogout={handleLogout}
        />
        {loginRequired && (
            <LoginPage
              onSuccess={() => {
                queryClient.removeQueries({ queryKey: ["execution-config"] });
                setAuthenticated(true);
              setLoginRequired(false);
            }}
            onCancel={() => setLoginRequired(false)}
          />
        )}
      </>
    </ConfigProvider>
  );
}
