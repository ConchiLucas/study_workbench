import { DownOutlined, RightOutlined } from "@ant-design/icons";
import { Tooltip } from "antd";
import { Fragment, useEffect, useState } from "react";

import {
  clozeResultSourceLabel,
  formatSourceAnswerTime,
  sourceWordCount,
  sourceWordDifficultyLabel,
  sourceWordEntryModeLabel,
  sourceWordsForDisplay,
  sourceWordTraceLabel,
} from "../features/clozeResultPresentation";
import { formatTime } from "../lib/format";
import type { ClozeResultItem, ClozeSourceWord } from "../types/clozeResult";

interface ClozeResultTableProps {
  items: ClozeResultItem[];
  loading: boolean;
  page: number;
  pageSize: number;
  resetKey: string;
}

function sourceTraceClass(word: ClozeSourceWord) {
  if (word.traceStatus === "historical") {
    return "historical";
  }
  if (word.traceStatus !== "available") {
    return "missing";
  }
  return "available";
}

function sourceWrongTime(value: string | null) {
  return value ? formatTime(value) : "-";
}

export default function ClozeResultTable({
  items,
  loading,
  page,
  pageSize,
  resetKey,
}: ClozeResultTableProps) {
  const [expandedIds, setExpandedIds] = useState<Set<number>>(() => new Set());

  useEffect(() => {
    setExpandedIds(new Set());
  }, [resetKey]);

  function toggleExpanded(itemId: number) {
    setExpandedIds((current) => {
      const next = new Set(current);
      if (next.has(itemId)) {
        next.delete(itemId);
      } else {
        next.add(itemId);
      }
      return next;
    });
  }

  return (
    <div className="sentence-table-scroll cloze-sentence-table-scroll">
      <table className="sentence-table cloze-sentence-main-table">
        <thead>
          <tr>
            <th>#</th>
            <th>时间</th>
            <th>用户</th>
            <th>句子 / 翻译</th>
            <th>来源单词</th>
            <th>来源</th>
            <th>模型</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          {loading ? (
            <tr>
              <td colSpan={8} className="empty-table-cell">
                加载中...
              </td>
            </tr>
          ) : items.length === 0 ? (
            <tr>
              <td colSpan={8} className="empty-table-cell">
                暂无生成结果
              </td>
            </tr>
          ) : (
            items.map((item, index) => {
              const expanded = expandedIds.has(item.id);
              const sourceWords = sourceWordsForDisplay(item);
              return (
                <Fragment key={item.id}>
                  <tr
                    className={`cloze-sentence-main-row ${expanded ? "expanded" : ""}`}
                    onClick={() => toggleExpanded(item.id)}
                  >
                    <td className="cloze-sentence-sequence">
                      #{(page - 1) * pageSize + index + 1}
                    </td>
                    <td>{formatTime(item.createTime)}</td>
                    <td>{item.userName || `用户 ${item.userId}`}</td>
                    <td>
                      <div className="cloze-sentence-cell">
                        <Tooltip title={item.sentence || "-"} placement="topLeft">
                          <strong>{item.sentence || "-"}</strong>
                        </Tooltip>
                        <Tooltip title={item.translationZh || "暂无翻译"} placement="topLeft">
                          <span>{item.translationZh || "暂无翻译"}</span>
                        </Tooltip>
                      </div>
                    </td>
                    <td>
                      <span className="cloze-source-count">
                        {sourceWordCount(item)} 个词
                      </span>
                    </td>
                    <td>{clozeResultSourceLabel(item)}</td>
                    <td>
                      <Tooltip title={item.model || "-"} placement="topLeft">
                        <span className="table-ellipsis-text">{item.model || "-"}</span>
                      </Tooltip>
                    </td>
                    <td>
                      <button
                        type="button"
                        className="cloze-expand-button"
                        aria-expanded={expanded}
                        aria-controls={`cloze-source-detail-${item.id}`}
                        onClick={(event) => {
                          event.stopPropagation();
                          toggleExpanded(item.id);
                        }}
                      >
                        {expanded ? <DownOutlined /> : <RightOutlined />}
                        <span>{expanded ? "收起" : "展开"}</span>
                      </button>
                    </td>
                  </tr>
                  {expanded ? (
                    <tr className="cloze-source-detail-row">
                      <td colSpan={8}>
                        <section
                          className="cloze-source-detail"
                          id={`cloze-source-detail-${item.id}`}
                          aria-label={`句子 ${item.id} 的来源单词`}
                        >
                          <div className="cloze-source-context">
                            <div>
                              <span>挖空句</span>
                              <p>{item.clozeSentence || "-"}</p>
                            </div>
                            <div>
                              <span>中文解释</span>
                              <p>{item.explanationZh || "暂无中文解释"}</p>
                            </div>
                          </div>

                          <div className="cloze-source-word-table-scroll">
                            <div className="cloze-source-word-grid cloze-source-word-head">
                              <span>来源单词</span>
                              <span>答错时间</span>
                              <span>入口 / 模式</span>
                              <span>词库 / 难度</span>
                              <span>词难度</span>
                              <span>耗时</span>
                              <span>正确答案</span>
                              <span>来源追溯</span>
                            </div>
                            {sourceWords.map((word, wordIndex) => (
                              <div
                                className={`cloze-source-word-grid cloze-source-word-row ${sourceTraceClass(word)}`}
                                key={`${item.id}-${word.sourceEventId || wordIndex}-${word.word}`}
                              >
                                <strong>{word.word || "-"}</strong>
                                <span>{sourceWrongTime(word.wrongTime)}</span>
                                <span>{sourceWordEntryModeLabel(word)}</span>
                                <span>{sourceWordDifficultyLabel(word)}</span>
                                <span>{word.wordDifficulty ?? "-"}</span>
                                <span>{formatSourceAnswerTime(word.answerTimeMs)}</span>
                                <span>{word.correctAnswer || "-"}</span>
                                <span className="cloze-source-trace">
                                  {sourceWordTraceLabel(word)}
                                </span>
                              </div>
                            ))}
                          </div>
                        </section>
                      </td>
                    </tr>
                  ) : null}
                </Fragment>
              );
            })
          )}
        </tbody>
      </table>
    </div>
  );
}
