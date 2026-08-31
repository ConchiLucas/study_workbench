import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useMemo, useState } from 'react'
import { glyphImageURL, listPinyin, speechAudioURL, syncPinyin } from '../../api/pinyin'
import type { PinyinItem } from '../../api/pinyinTypes'
import { buildGroupQuestions } from './pinyinQuizPreview'
import { PinyinQuizRow } from './PinyinQuizRow'

async function playSpeech(kpId: number, kind: 'solo' | 'word', speechUrl?: string) {
  const res = await fetch(speechAudioURL(kpId, kind, speechUrl))
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

export function PinyinPage() {
  const queryClient = useQueryClient()
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})
  const [speakingKey, setSpeakingKey] = useState<string | null>(null)
  const [speechError, setSpeechError] = useState('')
  const [detailGroup, setDetailGroup] = useState<{
    moduleName: string
    items: PinyinItem[]
  } | null>(null)

  const listQuery = useQuery({
    queryKey: ['pinyin', 'items', 'groups'],
    queryFn: () => listPinyin({ view: 'groups' }),
  })

  const syncMutation = useMutation({
    mutationFn: syncPinyin,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['pinyin'] }),
  })

  const onPlaySpeech = async (item: PinyinItem, kind: 'solo' | 'word') => {
    setSpeechError('')
    const key = `${item.kpId}-${kind}`
    setSpeakingKey(key)
    try {
      const url = kind === 'solo' ? item.soloSpeechUrl : item.wordSpeechUrl
      await playSpeech(item.kpId, kind, url)
    } catch (e) {
      setSpeechError(e instanceof Error ? e.message : '读音失败')
    } finally {
      setSpeakingKey(null)
    }
  }

  const error =
    speechError ||
    (listQuery.error instanceof Error && listQuery.error.message) ||
    (syncMutation.error instanceof Error && syncMutation.error.message) ||
    ''

  return (
    <section className="literacy-page pinyin-page" aria-label="拼音素材">
      <div className="page-heading">
        <div>
          <p className="eyebrow">CONTENT / PINYIN</p>
          <h1>拼音素材</h1>
          <p className="page-description">
            从题库同步声母/韵母；字图与读音由后台生成；本页用于浏览与试听。
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
        <span className="muted">{listQuery.data ? `共 ${listQuery.data.total} 音` : ''}</span>
      </div>

      {error ? <div className="error-panel" role="alert">{error}</div> : null}
      {syncMutation.data ? (
        <div className="info-panel">
          已同步 {syncMutation.data.upserted} / {syncMutation.data.total} 音
        </div>
      ) : null}

      {listQuery.isLoading ? (
        <div className="loading-panel">加载中…</div>
      ) : !listQuery.data || listQuery.data.total === 0 ? (
        <div className="empty-panel">暂无数据。请先点「从题库同步」。</div>
      ) : (
        <div className="group-list">
          {(listQuery.data.groups ?? []).map((g) => {
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
                    <span className="muted">
                      {g.items.length} 音 · {closed ? '展开' : '收起'}
                    </span>
                  </button>
                  <button
                    type="button"
                    className="refresh-button group-preview-link"
                    onClick={() =>
                      setDetailGroup({ moduleName: g.moduleName, items: g.items })
                    }
                  >
                    题目
                  </button>
                </div>
                {closed ? null : (
                  <div className="char-grid">
                    {g.items.map((item) => (
                      <PinyinCard
                        key={item.kpId}
                        item={item}
                        speakingKey={speakingKey}
                        onPlay={(kind) => void onPlaySpeech(item, kind)}
                      />
                    ))}
                  </div>
                )}
              </section>
            )
          })}
        </div>
      )}

      {detailGroup ? (
        <GroupQuizSheet
          moduleName={detailGroup.moduleName}
          items={detailGroup.items}
          onClose={() => setDetailGroup(null)}
        />
      ) : null}
    </section>
  )
}

function GroupQuizSheet({
  moduleName,
  items,
  onClose,
}: {
  moduleName: string
  items: PinyinItem[]
  onClose: () => void
}) {
  const questions = useMemo(() => buildGroupQuestions(items), [items])
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
    <div className="fullscreen-sheet" role="dialog" aria-modal="true" aria-labelledby="pinyin-quiz-sheet-title">
      <header className="fullscreen-sheet-header">
        <div>
          {active ? (
            <button type="button" className="sheet-back" onClick={() => setActiveId(null)}>
              ← 返回列表
            </button>
          ) : null}
          <h2 id="pinyin-quiz-sheet-title">
            {moduleName}
            {active ? ` · ${active.target.letter} · ${active.title}` : ' · 题目列表'}
          </h2>
          {!active ? (
            <p className="muted">
              两类题预览 · 听例字选音 / 听单读选字母 · 左题干 / 右两列选项 · 点击进入详情试答
            </p>
          ) : null}
        </div>
        <button type="button" className="refresh-button group-preview-link" onClick={onClose}>
          关闭
        </button>
      </header>

      <div className="fullscreen-sheet-body">
        {active ? (
          <PinyinQuizRow question={active} mode="detail" />
        ) : questions.length === 0 ? (
          <div className="empty-panel">本组字母不足，暂无法生成四选一题目。</div>
        ) : (
          <div className="quiz-list">
            {questions.map((q) => (
              <PinyinQuizRow key={q.id} question={q} mode="list" onOpen={() => setActiveId(q.id)} />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

function PinyinCard({
  item,
  speakingKey,
  onPlay,
}: {
  item: PinyinItem
  speakingKey: string | null
  onPlay: (kind: 'solo' | 'word') => void
}) {
  const soloKey = `${item.kpId}-solo`
  const wordKey = `${item.kpId}-word`
  const hasSolo = Boolean(item.soloText)
  const hasWord = Boolean(item.wordText)

  return (
    <article className="char-card">
      {item.glyphImageUrl ? (
        <img
          className="glyph-preview"
          src={glyphImageURL(item.kpId, item.glyphImageUrl)}
          alt={item.letter}
        />
      ) : (
        <div className="char-glyph">{item.letter}</div>
      )}
      <div className="sense-tag">
        {hasSolo ? `单读 ${item.soloText}` : '无单读'}
        {hasWord ? ` · 例字 ${item.wordText}` : ''}
        {item.soloSpeechUrl || item.wordSpeechUrl ? ' · 有读音' : ''}
      </div>
      <div className="card-actions">
        <button
          type="button"
          className="mini-btn"
          disabled={!hasSolo || speakingKey === soloKey}
          onClick={() => onPlay('solo')}
        >
          {speakingKey === soloKey ? '…' : item.soloSpeechUrl ? '读单读✓' : '读单读'}
        </button>
        <button
          type="button"
          className="mini-btn"
          disabled={!hasWord || speakingKey === wordKey}
          onClick={() => onPlay('word')}
        >
          {speakingKey === wordKey ? '…' : item.wordSpeechUrl ? '读例字✓' : '读例字'}
        </button>
      </div>
    </article>
  )
}
