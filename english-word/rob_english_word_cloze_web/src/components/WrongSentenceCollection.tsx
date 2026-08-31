import { useEffect, useRef, useState, type KeyboardEvent } from "react";
import { ApiError, getWrongSentenceDetail, getWrongSentences } from "../lib/api";
import type {
  WrongSentenceDetail,
  WrongSentenceItem,
  WrongSentencePageResponse,
  WrongSentenceQuery,
} from "../types/cloze";
import { FullscreenCloseButton } from "./FullscreenCloseButton";

interface WrongSentenceCollectionProps {
  token: string;
  onClose: () => void;
  onAuthExpired: () => void;
}

const EMPTY_PAGE: WrongSentencePageResponse = {
  items: [],
  total: 0,
  current: 1,
  pages: 0,
  summary: { activeCount: 0, dueCount: 0, stage1Count: 0, stage2Count: 0, completedCount: 0 },
};

function formatDateTime(value?: string | null) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatCost(value?: number | null) {
  if (value === null || value === undefined) return "-";
  return value < 1000 ? `${value}ms` : `${(value / 1000).toFixed(1)}s`;
}

function stageLabel(item: WrongSentenceItem) {
  if (item.status === "completed") return "已完成";
  return ["立即复习", "7 天复习", "15 天复习", "完成"][item.reviewStage] || "立即复习";
}

function sourceLabel(source: WrongSentenceItem["practiceContext"]) {
  return source === "solo" ? "单独训练" : "开始答题";
}

export function WrongSentenceCollection({ token, onClose, onAuthExpired }: WrongSentenceCollectionProps) {
  const [status, setStatus] = useState<"active" | "completed">("active");
  const [source, setSource] = useState<"all" | "review" | "solo">("all");
  const [availability, setAvailability] = useState<"all" | "due" | "waiting">("all");
  const [sort, setSort] = useState<"nextReview" | "recent" | "wrongCount">("nextReview");
  const [keyword, setKeyword] = useState("");
  const [page, setPage] = useState(1);
  const [data, setData] = useState<WrongSentencePageResponse>(EMPTY_PAGE);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [expandedIds, setExpandedIds] = useState<Set<number>>(() => new Set());
  const [details, setDetails] = useState<Record<number, WrongSentenceDetail>>({});
  const [detailLoadingIds, setDetailLoadingIds] = useState<Set<number>>(() => new Set());
  const [detailErrors, setDetailErrors] = useState<Record<number, string>>({});
  const requestSequence = useRef(0);
  const dialogRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const previousFocus = document.activeElement instanceof HTMLElement
      ? document.activeElement
      : null;
    dialogRef.current?.focus();
    return () => previousFocus?.focus();
  }, []);

  const query: WrongSentenceQuery = {
    status,
    source,
    availability,
    sort,
    keyword: keyword.trim() || undefined,
    page,
    size: 20,
  };

  useEffect(() => {
    const sequence = ++requestSequence.current;
    setLoading(true);
    setLoadError("");
    void getWrongSentences(token, query)
      .then((response) => {
        if (sequence === requestSequence.current) setData(response);
      })
      .catch((error: unknown) => {
        if (sequence !== requestSequence.current) return;
        if (error instanceof ApiError && (error.status === 401 || error.status === 403)) {
          onAuthExpired();
          return;
        }
        setLoadError(error instanceof Error ? error.message : "错题集加载失败");
      })
      .finally(() => {
        if (sequence === requestSequence.current) setLoading(false);
      });
  }, [token, status, source, availability, sort, keyword, page]);

  function reloadList() {
    requestSequence.current += 1;
    setPage((current) => current);
    setLoading(true);
    setLoadError("");
    const sequence = requestSequence.current;
    void getWrongSentences(token, query)
      .then((response) => {
        if (sequence === requestSequence.current) setData(response);
      })
      .catch((error: unknown) => {
        if (error instanceof ApiError && (error.status === 401 || error.status === 403)) {
          onAuthExpired();
          return;
        }
        if (sequence === requestSequence.current) {
          setLoadError(error instanceof Error ? error.message : "错题集加载失败");
        }
      })
      .finally(() => {
        if (sequence === requestSequence.current) setLoading(false);
      });
  }

  function updateFilter(action: () => void) {
    setPage(1);
    setExpandedIds(new Set());
    action();
  }

  async function loadDetail(progressId: number, force = false) {
    if (details[progressId] && !force) return;
    setDetailLoadingIds((current) => new Set(current).add(progressId));
    setDetailErrors((current) => ({ ...current, [progressId]: "" }));
    try {
      const detail = await getWrongSentenceDetail(token, progressId);
      setDetails((current) => ({ ...current, [progressId]: detail }));
    } catch (error) {
      if (error instanceof ApiError && (error.status === 401 || error.status === 403)) {
        onAuthExpired();
        return;
      }
      setDetailErrors((current) => ({
        ...current,
        [progressId]: error instanceof Error ? error.message : "详情加载失败",
      }));
    } finally {
      setDetailLoadingIds((current) => {
        const next = new Set(current);
        next.delete(progressId);
        return next;
      });
    }
  }

  function toggleExpanded(progressId: number) {
    const isExpanded = expandedIds.has(progressId);
    setExpandedIds((current) => {
      const next = new Set(current);
      if (isExpanded) next.delete(progressId);
      else next.add(progressId);
      return next;
    });
    if (!isExpanded) void loadDetail(progressId);
  }

  function trapDialogFocus(event: KeyboardEvent<HTMLDivElement>) {
    if (event.key !== "Tab" || !dialogRef.current) return;
    const focusable = Array.from(dialogRef.current.querySelectorAll<HTMLElement>(
      'button:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])',
    ));
    if (focusable.length === 0) {
      event.preventDefault();
      dialogRef.current.focus();
      return;
    }
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (event.shiftKey && (document.activeElement === first || document.activeElement === dialogRef.current)) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }

  return (
    <div
      ref={dialogRef}
      className="wrong-sentence-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="wrong-sentence-title"
      tabIndex={-1}
      onKeyDown={trapDialogFocus}
    >
      <FullscreenCloseButton label="关闭错题集" onClose={onClose} />
      <header className="wrong-sentence-header">
        <div>
          <span className="launcher-kicker">WRONG SENTENCE COLLECTION</span>
          <h2 id="wrong-sentence-title">句子错题集</h2>
          <p>整句复习按立即、7 天、15 天三个阶段推进</p>
        </div>
        <button className="ghost-button wrong-refresh-button" type="button" onClick={reloadList} disabled={loading}>刷新</button>
      </header>

      <section className="wrong-summary-grid" aria-label="错题概览">
        <div><span>待复习</span><strong>{data.summary.activeCount}</strong></div>
        <div><span>现在到期</span><strong>{data.summary.dueCount}</strong></div>
        <div><span>7 天阶段</span><strong>{data.summary.stage1Count}</strong></div>
        <div><span>15 天阶段</span><strong>{data.summary.stage2Count}</strong></div>
        <div><span>已完成</span><strong>{data.summary.completedCount}</strong></div>
      </section>

      <div className="wrong-collection-tabs" role="tablist" aria-label="错题状态">
        <button role="tab" aria-selected={status === "active"} className={status === "active" ? "active" : ""} type="button" onClick={() => updateFilter(() => setStatus("active"))}>待复习</button>
        <button role="tab" aria-selected={status === "completed"} className={status === "completed" ? "active" : ""} type="button" onClick={() => updateFilter(() => setStatus("completed"))}>已完成</button>
      </div>

      <section className="wrong-filter-bar" aria-label="错题筛选">
        <label>来源
          <select value={source} onChange={(event) => updateFilter(() => setSource(event.target.value as typeof source))}>
            <option value="all">全部</option><option value="review">开始答题</option><option value="solo">单独训练</option>
          </select>
        </label>
        <label>复习时间
          <select value={availability} onChange={(event) => updateFilter(() => setAvailability(event.target.value as typeof availability))}>
            <option value="all">全部</option><option value="due">现在到期</option><option value="waiting">尚未到期</option>
          </select>
        </label>
        <label>排序
          <select value={sort} onChange={(event) => updateFilter(() => setSort(event.target.value as typeof sort))}>
            <option value="nextReview">下次复习</option><option value="recent">最近答错</option><option value="wrongCount">错误次数</option>
          </select>
        </label>
        <label className="wrong-keyword-field">搜索
          <input value={keyword} onChange={(event) => updateFilter(() => setKeyword(event.target.value))} placeholder="句子、翻译或目标词" />
        </label>
      </section>

      {loadError ? (
        <section className="wrong-collection-state error"><p>{loadError}</p><button className="ghost-button" type="button" onClick={reloadList}>重试</button></section>
      ) : loading ? (
        <section className="wrong-collection-state"><p>正在加载错题集…</p></section>
      ) : data.items.length === 0 ? (
        <section className="wrong-collection-state"><strong>{status === "active" ? "暂无待复习错题" : "暂无已完成错题"}</strong><p>句子中任意一个单词拼写错误后，会自动出现在这里。</p></section>
      ) : (
        <section className="wrong-sentence-table" aria-label="句子错题列表">
          <div className="wrong-sentence-table-head" aria-hidden="true">
            <span>最近答错</span><span>句子 / 翻译</span><span>错词</span><span>来源</span><span>复习阶段</span><span>下次复习</span><span>错误次数</span><span>操作</span>
          </div>
          <div className="wrong-sentence-table-body">
            {data.items.map((item) => {
              const expanded = expandedIds.has(item.progressId);
              const detail = details[item.progressId];
              return (
                <article className={`wrong-sentence-entry ${expanded ? "expanded" : ""}`} key={item.progressId}>
                  <div className="wrong-sentence-row">
                    <time data-label="最近答错">{formatDateTime(item.lastWrongTime)}</time>
                    <div className="wrong-sentence-copy" data-label="句子 / 翻译"><strong>{item.clozeSentence}</strong><span>{item.translationZh || "-"}</span></div>
                    <span data-label="错词" className="wrong-word-count">{item.wrongBlankCount} 个</span>
                    <span data-label="来源">{sourceLabel(item.practiceContext)}</span>
                    <span data-label="复习阶段" className={`wrong-stage-badge stage-${item.reviewStage}`}>{stageLabel(item)}</span>
                    <span data-label="下次复习">{item.status === "completed" ? "已完成" : formatDateTime(item.nextReviewTime)}</span>
                    <strong data-label="错误次数">{item.wrongCount}</strong>
                    <button className="wrong-expand-button" type="button" aria-expanded={expanded} onClick={() => toggleExpanded(item.progressId)}>{expanded ? "收起" : "展开"}</button>
                  </div>

                  {expanded ? (
                    <div className="wrong-sentence-detail">
                      {detailLoadingIds.has(item.progressId) ? <p>正在加载详情…</p> : null}
                      {detailErrors[item.progressId] ? <div className="wrong-detail-error"><span>{detailErrors[item.progressId]}</span><button type="button" onClick={() => void loadDetail(item.progressId, true)}>重试详情</button></div> : null}
                      {detail ? (
                        <>
                          <div className="wrong-detail-sentence"><span>完整句</span><strong>{detail.item.sentence}</strong><p>{detail.item.translationZh}</p></div>
                          <section className="wrong-blank-grid" aria-label="逐空复习状态">
                            {detail.blanks.map((blank) => (
                              <article key={blank.index} className={blank.lastCorrect ? "correct" : "wrong"}>
                                <span>第 {blank.index + 1} 空 · {blank.lastCorrect ? "最近正确" : "最近错误"}</span>
                                <strong>{blank.word}</strong>
                                <p>{blank.meaning || "暂无词义"}</p>
                                <small>{blank.wordReviewStage === null || blank.wordReviewStage === undefined ? "无独立单词进度" : `单词复习阶段 ${blank.wordReviewStage}`}</small>
                              </article>
                            ))}
                          </section>
                          <section className="wrong-review-track">
                            <h3>复习轨迹</h3>
                            <div>{detail.reviewStages.map((stage) => <span key={stage.stage} className={stage.state}><i />{stage.label}</span>)}</div>
                          </section>
                          <div className="wrong-detail-meta"><span>首次答错：{formatDateTime(detail.item.firstWrongTime)}</span><span>最近耗时：{formatCost(detail.item.lastCostMs)}</span><span>来源难度：{detail.item.difficultyLabel || "未标注"}</span></div>
                          <section className="wrong-attempts"><h3>最近复习</h3>{detail.attempts.map((attempt) => <div key={attempt.recordId}><time>{formatDateTime(attempt.answeredAt)}</time><strong className={attempt.correct ? "correct" : "wrong"}>{attempt.correct ? "整句正确" : attempt.actionType === "reveal" ? "查看答案" : "整句错误"}</strong><span>{sourceLabel(attempt.practiceContext)}</span><span>{formatCost(attempt.costMs)}</span></div>)}</section>
                        </>
                      ) : null}
                    </div>
                  ) : null}
                </article>
              );
            })}
          </div>
        </section>
      )}

      {data.pages > 1 ? <nav className="wrong-pagination" aria-label="错题分页"><button type="button" disabled={page <= 1 || loading} onClick={() => setPage((current) => Math.max(1, current - 1))}>上一页</button><span>{page} / {data.pages}</span><button type="button" disabled={page >= data.pages || loading} onClick={() => setPage((current) => current + 1)}>下一页</button></nav> : null}
    </div>
  );
}
