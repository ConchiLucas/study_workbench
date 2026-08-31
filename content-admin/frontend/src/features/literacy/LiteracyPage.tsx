import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useMemo, useState } from 'react'
import {
  listLiteracy,
  patchLiteracyChar,
  speechAudioURL,
  syncLiteracy,
} from '../../api/literacy'
import type { LiteracyChar } from '../../api/literacyTypes'
import {
  buildGroupQuestions,
} from './quizPreview'
import { QuizQuestionRow } from './QuizQuestionRow'

type ViewMode = 'groups' | 'table'
type FilterMode = '' | 'true' | 'false'

async function playSpeech(kpId: number, speechAudioUrl?: string) {
  const res = await fetch(speechAudioURL(kpId, speechAudioUrl))
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(typeof body.error === 'string' ? body.error : `读音失败 ${res.status}`)
  }
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const audio = new Audio(url)
  audio.onended = () => URL.revokeObjectURL(url)
  await audio.play()
}

export function LiteracyPage() {
  const queryClient = useQueryClient()
  const [view, setView] = useState<ViewMode>('groups')
  const [filter, setFilter] = useState<FilterMode>('')
  const [q, setQ] = useState('')
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})
  const [busyKp, setBusyKp] = useState<number | null>(null)
  const [speakingKp, setSpeakingKp] = useState<number | null>(null)
  const [speechError, setSpeechError] = useState('')
  const [detailGroup, setDetailGroup] = useState<{
    moduleName: string
    chars: LiteracyChar[]
  } | null>(null)

  const listQuery = useQuery({
    queryKey: ['literacy', 'chars', view, filter],
    queryFn: () => listLiteracy({ view, needsSenseImage: filter }),
  })

  const syncMutation = useMutation({
    mutationFn: syncLiteracy,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['literacy'] }),
  })

  const overrideMutation = useMutation({
    mutationFn: ({ kpId, value }: { kpId: number; value: boolean }) =>
      patchLiteracyChar(kpId, value),
    onMutate: ({ kpId }) => setBusyKp(kpId),
    onSettled: () => {
      setBusyKp(null)
      void queryClient.invalidateQueries({ queryKey: ['literacy'] })
    },
  })

  const onPlaySpeech = async (char: LiteracyChar) => {
    setSpeechError('')
    setSpeakingKp(char.kpId)
    try {
      await playSpeech(char.kpId, char.speechAudioUrl)
    } catch (e) {
      setSpeechError(e instanceof Error ? e.message : '读音失败')
    } finally {
      setSpeakingKp(null)
    }
  }

  const chars = useMemo(() => {
    const raw =
      view === 'table'
        ? listQuery.data?.chars ?? []
        : (listQuery.data?.groups ?? []).flatMap((g) => g.chars)
    if (!q.trim()) return raw
    return raw.filter((c) => c.charText.includes(q.trim()))
  }, [listQuery.data, view, q])

  const error =
    speechError ||
    (listQuery.error instanceof Error && listQuery.error.message) ||
    (syncMutation.error instanceof Error && syncMutation.error.message) ||
    (overrideMutation.error instanceof Error && overrideMutation.error.message) ||
    ''

  return (
    <section className="literacy-page" aria-label="识字素材">
      <div className="page-heading">
        <div>
          <p className="eyebrow">CONTENT / LITERACY</p>
          <h1>识字素材</h1>
          <p className="page-description">
            字图 / 义图 / 读音由后台生成；本页用于浏览、试听与标记是否要义图。
          </p>
        </div>
        <div className="heading-actions">
          <button
            type="button"
            className="refresh-button"
            onClick={() => syncMutation.mutate()}
            disabled={syncMutation.isPending}
          >
            {syncMutation.isPending ? '同步中…' : '从题库同步'}
          </button>
        </div>
      </div>

      <div className="literacy-toolbar">
        <div className="segmented" role="group" aria-label="视图">
          <button type="button" className={view === 'groups' ? 'active' : ''} onClick={() => setView('groups')}>
            按组
          </button>
          <button type="button" className={view === 'table' ? 'active' : ''} onClick={() => setView('table')}>
            表格
          </button>
        </div>
        <div className="segmented" role="group" aria-label="义图筛选">
          <button type="button" className={filter === '' ? 'active' : ''} onClick={() => setFilter('')}>
            全部
          </button>
          <button type="button" className={filter === 'true' ? 'active' : ''} onClick={() => setFilter('true')}>
            要义图
          </button>
          <button type="button" className={filter === 'false' ? 'active' : ''} onClick={() => setFilter('false')}>
            不要义图
          </button>
        </div>
        <input
          className="search-input"
          placeholder="搜索单字"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <span className="muted">{listQuery.data ? `共 ${listQuery.data.total} 字` : ''}</span>
      </div>

      {error ? <div className="error-panel" role="alert">{error}</div> : null}
      {syncMutation.data ? (
        <div className="info-panel">已同步 {syncMutation.data.upserted} / {syncMutation.data.total} 字</div>
      ) : null}

      {listQuery.isLoading ? (
        <div className="loading-panel">加载中…</div>
      ) : !listQuery.data || listQuery.data.total === 0 ? (
        <div className="empty-panel">暂无数据。请先点「从题库同步」。</div>
      ) : view === 'groups' ? (
        <div className="group-list">
          {(listQuery.data.groups ?? [])
            .map((g) => ({
              ...g,
              chars: g.chars.filter((c) => !q.trim() || c.charText.includes(q.trim())),
            }))
            .filter((g) => g.chars.length > 0)
            .map((g) => {
              const closed = collapsed[g.moduleCode]
              return (
                <section key={g.moduleCode} className="literacy-group">
                  <div className="group-header-row">
                    <button
                      type="button"
                      className="group-header"
                      onClick={() =>
                        setCollapsed((prev) => ({ ...prev, [g.moduleCode]: !prev[g.moduleCode] }))
                      }
                    >
                      <span>{g.moduleName}</span>
                      <span className="muted">{g.chars.length} 字 · {closed ? '展开' : '收起'}</span>
                    </button>
                    <button
                      type="button"
                      className="refresh-button group-preview-link"
                      onClick={() =>
                        setDetailGroup({ moduleName: g.moduleName, chars: g.chars })
                      }
                    >
                      题目
                    </button>
                  </div>
                  {!closed ? (
                    <div className="char-grid">
                      {g.chars.map((c) => (
                        <CharCard
                          key={c.kpId}
                          char={c}
                          busy={busyKp === c.kpId}
                          speaking={speakingKp === c.kpId}
                          onPlaySpeech={() => void onPlaySpeech(c)}
                          onToggleSense={() =>
                            overrideMutation.mutate({
                              kpId: c.kpId,
                              value: !c.effectiveNeedsSenseImage,
                            })
                          }
                        />
                      ))}
                    </div>
                  ) : null}
                </section>
              )
            })}
        </div>
      ) : (
        <div className="table-wrap">
          <table className="literacy-table">
            <thead>
              <tr>
                <th>字</th>
                <th>组别</th>
                <th>义图标记</th>
                <th>字图</th>
                <th>义图</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {chars.map((c) => (
                <tr key={c.kpId}>
                  <td className="char-cell">{c.charText}</td>
                  <td>{c.moduleName}</td>
                  <td>{c.effectiveNeedsSenseImage ? '要义图' : '不要义图'}</td>
                  <td>
                    {c.glyphImageUrl ? (
                      <img className="glyph-thumb" src={c.glyphImageUrl} alt={c.charText} />
                    ) : (
                      '—'
                    )}
                  </td>
                  <td>
                    {c.senseImageUrl ? (
                      <img className="glyph-thumb" src={c.senseImageUrl} alt={`${c.charText}义图`} />
                    ) : (
                      '—'
                    )}
                  </td>
                  <td className="table-actions">
                    <button
                      type="button"
                      className="mini-btn"
                      disabled={speakingKp === c.kpId}
                      onClick={() => void onPlaySpeech(c)}
                    >
                      {speakingKp === c.kpId ? '…' : c.speechAudioUrl ? '读音✓' : '读音'}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {detailGroup ? (
        <GroupQuizSheet
          moduleName={detailGroup.moduleName}
          chars={detailGroup.chars}
          onClose={() => setDetailGroup(null)}
        />
      ) : null}
    </section>
  )
}

function GroupQuizSheet({
  moduleName,
  chars,
  onClose,
}: {
  moduleName: string
  chars: LiteracyChar[]
  onClose: () => void
}) {
  const questions = useMemo(() => buildGroupQuestions(chars), [chars])
  const [activeId, setActiveId] = useState<string | null>(null)
  const active = questions.find((q) => q.id === activeId) ?? null

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return
      if (activeId) setActiveId(null)
      else onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [activeId, onClose])

  return (
    <div className="fullscreen-sheet" role="dialog" aria-modal="true" aria-labelledby="quiz-sheet-title">
      <header className="fullscreen-sheet-header">
        <div>
          {active ? (
            <button type="button" className="sheet-back" onClick={() => setActiveId(null)}>
              ← 返回列表
            </button>
          ) : null}
          <h2 id="quiz-sheet-title">
            {moduleName}
            {active ? ` · ${active.target.charText} · ${active.title}` : ' · 题目列表'}
          </h2>
          {!active ? (
            <p className="muted">两类题预览 · 看字选义 / 看义选字 · 左题干 / 右两列选项 · 点击进入详情试答</p>
          ) : null}
        </div>
        <button type="button" className="refresh-button group-preview-link" onClick={onClose}>
          关闭
        </button>
      </header>

      <div className="fullscreen-sheet-body">
        {active ? (
          <QuizQuestionRow question={active} mode="detail" />
        ) : (
          <div className="quiz-list">
            {questions.map((q) => (
              <QuizQuestionRow key={q.id} question={q} mode="list" onOpen={() => setActiveId(q.id)} />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

function CharCard({
  char,
  busy,
  speaking,
  onPlaySpeech,
  onToggleSense,
}: {
  char: LiteracyChar
  busy: boolean
  speaking: boolean
  onPlaySpeech: () => void
  onToggleSense: () => void
}) {
  return (
    <article className={`char-card${char.effectiveNeedsSenseImage ? ' needs-sense' : ''}`}>
      {char.glyphImageUrl ? (
        <img className="glyph-preview" src={char.glyphImageUrl} alt={char.charText} />
      ) : (
        <div className="char-glyph">{char.charText}</div>
      )}
      {char.senseImageUrl ? (
        <img className="sense-preview" src={char.senseImageUrl} alt={`${char.charText}义图`} />
      ) : char.effectiveNeedsSenseImage ? (
        <div className="sense-placeholder">义图未生成</div>
      ) : null}
      <button
        type="button"
        className={`sense-tag${char.effectiveNeedsSenseImage ? ' yes' : ' no'}`}
        disabled={busy}
        title="点击切换要/不要义图"
        onClick={onToggleSense}
      >
        {char.effectiveNeedsSenseImage ? '要义图' : '不要义图'}
        {char.speechAudioUrl ? ' · 有读音' : ''}
      </button>
      <div className="card-actions">
        <button type="button" className="mini-btn" disabled={speaking} onClick={onPlaySpeech}>
          {speaking ? '…' : char.speechAudioUrl ? '读音✓' : '读音'}
        </button>
      </div>
    </article>
  )
}
