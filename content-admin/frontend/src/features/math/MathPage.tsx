import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useMemo, useState } from 'react'
import {
  batchGenerateGlyphs,
  batchGenerateSpeech,
  generateGlyph,
  glyphImageURL,
  listMath,
  regenerateSpeech,
  speechAudioURL,
  syncMath,
} from '../../api/math'
import type { MathItem } from '../../api/mathTypes'
import { buildMathGroupQuestions } from './mathQuizPreview'
import { MathQuizRow } from './MathQuizRow'

const KIND_LABEL: Record<string, string> = {
  add: '加法',
  sub: '减法',
}

function kindLabel(kind: string) {
  if (!kind) return '图形'
  return KIND_LABEL[kind] ?? kind
}

async function playSpeech(kpId: number, speechUrl?: string) {
  const res = await fetch(speechAudioURL(kpId, speechUrl))
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

export function MathPage() {
  const queryClient = useQueryClient()
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})
  const [busyGlyphModule, setBusyGlyphModule] = useState<string | null>(null)
  const [busySpeechModule, setBusySpeechModule] = useState<string | null>(null)
  const [busyGlyphKp, setBusyGlyphKp] = useState<number | null>(null)
  const [busySpeechKp, setBusySpeechKp] = useState<number | null>(null)
  const [speakingKp, setSpeakingKp] = useState<number | null>(null)
  const [speechError, setSpeechError] = useState('')
  const [detailGroup, setDetailGroup] = useState<{
    moduleName: string
    items: MathItem[]
  } | null>(null)

  const listQuery = useQuery({
    queryKey: ['math', 'items', 'groups'],
    queryFn: () => listMath({ view: 'groups' }),
  })

  const syncMutation = useMutation({
    mutationFn: syncMath,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['math'] }),
  })

  const glyphBatchMutation = useMutation({
    mutationFn: (moduleCode: string) => batchGenerateGlyphs(moduleCode),
    onMutate: (moduleCode) => setBusyGlyphModule(moduleCode),
    onSettled: () => {
      setBusyGlyphModule(null)
      void queryClient.invalidateQueries({ queryKey: ['math'] })
    },
  })

  const speechBatchMutation = useMutation({
    mutationFn: (moduleCode: string) => batchGenerateSpeech(moduleCode),
    onMutate: (moduleCode) => setBusySpeechModule(moduleCode),
    onSettled: () => {
      setBusySpeechModule(null)
      void queryClient.invalidateQueries({ queryKey: ['math'] })
    },
  })

  const glyphMutation = useMutation({
    mutationFn: generateGlyph,
    onMutate: (kpId) => setBusyGlyphKp(kpId),
    onSettled: () => {
      setBusyGlyphKp(null)
      void queryClient.invalidateQueries({ queryKey: ['math'] })
    },
  })

  const speechRegenMutation = useMutation({
    mutationFn: regenerateSpeech,
    onMutate: (kpId) => setBusySpeechKp(kpId),
    onSuccess: async (dto) => {
      setSpeechError('')
      setSpeakingKp(dto.kpId)
      try {
        await playSpeech(dto.kpId, dto.speechAudioUrl)
      } catch (e) {
        setSpeechError(e instanceof Error ? e.message : '读音失败')
      } finally {
        setSpeakingKp(null)
      }
    },
    onSettled: () => {
      setBusySpeechKp(null)
      void queryClient.invalidateQueries({ queryKey: ['math'] })
    },
  })

  const onPlaySpeech = async (item: MathItem) => {
    setSpeechError('')
    setSpeakingKp(item.kpId)
    try {
      await playSpeech(item.kpId, item.speechAudioUrl)
    } catch (e) {
      setSpeechError(e instanceof Error ? e.message : '读音失败')
    } finally {
      setSpeakingKp(null)
    }
  }

  const error =
    speechError ||
    (listQuery.error instanceof Error && listQuery.error.message) ||
    (syncMutation.error instanceof Error && syncMutation.error.message) ||
    (glyphBatchMutation.error instanceof Error && glyphBatchMutation.error.message) ||
    (speechBatchMutation.error instanceof Error && speechBatchMutation.error.message) ||
    (glyphMutation.error instanceof Error && glyphMutation.error.message) ||
    (speechRegenMutation.error instanceof Error && speechRegenMutation.error.message) ||
    ''

  return (
    <section className="literacy-page math-page" aria-label="算术素材">
      <div className="page-heading">
        <div>
          <p className="eyebrow">CONTENT / MATH</p>
          <h1>算术素材</h1>
          <p className="page-description">
            加减法淡蓝方格纸字图；认识图形画真实几何图形；题干 TTS 读音可按组批量生成。
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
        <span className="muted">{listQuery.data ? `共 ${listQuery.data.total} 题` : ''}</span>
      </div>

      {error ? <div className="error-panel" role="alert">{error}</div> : null}
      {syncMutation.data ? (
        <div className="info-panel">
          已同步 {syncMutation.data.upserted} / {syncMutation.data.total} 题
        </div>
      ) : null}
      {glyphBatchMutation.data ? (
        <div className="info-panel">
          批量字图：生成 {glyphBatchMutation.data.generated}，失败 {glyphBatchMutation.data.failed}
          {glyphBatchMutation.data.errors?.length
            ? ` · ${glyphBatchMutation.data.errors.join('；')}`
            : ''}
        </div>
      ) : null}
      {speechBatchMutation.data ? (
        <div className="info-panel">
          批量读音：生成 {speechBatchMutation.data.generated}，跳过 {speechBatchMutation.data.skipped}
          ，失败 {speechBatchMutation.data.failed}
          {speechBatchMutation.data.errors?.length
            ? ` · ${speechBatchMutation.data.errors.join('；')}`
            : ''}
        </div>
      ) : null}

      {listQuery.isLoading ? (
        <div className="loading-panel">加载中…</div>
      ) : !listQuery.data || listQuery.data.total === 0 ? (
        <div className="empty-panel">暂无数据。请先点「从题库同步」。</div>
      ) : (
        <div className="group-list">
          {(listQuery.data.groups ?? [])
            .filter((g) => g.moduleCode === 'add10' || g.moduleCode === 'sub10' || g.moduleCode === 'shape')
            .map((g) => {
            const closed = collapsed[g.moduleCode]
            const glyphBusy = busyGlyphModule === g.moduleCode
            const speechBusy = busySpeechModule === g.moduleCode
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
                      {g.items.length} 题 · {closed ? '展开' : '收起'}
                    </span>
                  </button>
                  <button
                    type="button"
                    className="mini-btn"
                    disabled={glyphBusy || glyphBatchMutation.isPending}
                    onClick={() => glyphBatchMutation.mutate(g.moduleCode)}
                  >
                    {glyphBusy ? '生成中…' : '生成本组字图'}
                  </button>
                  <button
                    type="button"
                    className="mini-btn"
                    disabled={speechBusy || speechBatchMutation.isPending}
                    onClick={() => speechBatchMutation.mutate(g.moduleCode)}
                  >
                    {speechBusy ? '生成中…' : '生成本组读音'}
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
                  <div className="char-grid math-grid">
                    {g.items.map((item) => (
                      <MathCard
                        key={item.kpId}
                        item={item}
                        busyGlyph={busyGlyphKp === item.kpId}
                        busySpeech={busySpeechKp === item.kpId}
                        speaking={speakingKp === item.kpId}
                        onGenerateGlyph={() => glyphMutation.mutate(item.kpId)}
                        onPlay={() => void onPlaySpeech(item)}
                        onRegenSpeech={() => speechRegenMutation.mutate(item.kpId)}
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
  items: MathItem[]
  onClose: () => void
}) {
  const questions = useMemo(() => buildMathGroupQuestions(items), [items])
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

  const typeHint =
    items[0]?.moduleCode === 'shape'
      ? '看名称选图形 / 看图形选名称'
      : '看算式选答案 / 看图选答案 · 四个连续数字选项'

  return (
    <div className="fullscreen-sheet" role="dialog" aria-modal="true" aria-labelledby="math-quiz-sheet-title">
      <header className="fullscreen-sheet-header">
        <div>
          {active ? (
            <button type="button" className="sheet-back" onClick={() => setActiveId(null)}>
              ← 返回列表
            </button>
          ) : null}
          <h2 id="math-quiz-sheet-title">
            {moduleName}
            {active ? ` · ${active.target.title} · ${active.title}` : ' · 题目列表'}
          </h2>
          {!active ? (
            <p className="muted">
              {typeHint} · 预览本组前 30 题 · 点击进入详情试答
            </p>
          ) : null}
        </div>
        <button type="button" className="refresh-button group-preview-link" onClick={onClose}>
          关闭
        </button>
      </header>
      <div className="fullscreen-sheet-body">
        {active ? (
          <MathQuizRow question={active} detail />
        ) : (
          <div className="quiz-list">
            {questions.map((q) => (
              <div
                key={q.id}
                className="quiz-list-item"
                role="button"
                tabIndex={0}
                onClick={() => setActiveId(q.id)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault()
                    setActiveId(q.id)
                  }
                }}
              >
                <MathQuizRow question={q} />
              </div>
            ))}
            {questions.length === 0 ? (
              <p className="muted">本组暂无法生成题目预览。</p>
            ) : null}
          </div>
        )}
      </div>
    </div>
  )
}

function MathCard({
  item,
  busyGlyph,
  busySpeech,
  speaking,
  onGenerateGlyph,
  onPlay,
  onRegenSpeech,
}: {
  item: MathItem
  busyGlyph: boolean
  busySpeech: boolean
  speaking: boolean
  onGenerateGlyph: () => void
  onPlay: () => void
  onRegenSpeech: () => void
}) {
  const hasSpeech = Boolean(item.speechAudioUrl)
  return (
    <article className="char-card math-card">
      {item.glyphImageUrl ? (
        <img
          className="glyph-preview"
          src={glyphImageURL(item.kpId, item.glyphImageUrl)}
          alt={item.title}
        />
      ) : (
        <div className="math-title" title={item.title}>
          {item.title}
        </div>
      )}
      <div className="char-meta">
        <span className="badge">{kindLabel(item.kind)}</span>
        <span className="badge muted-badge">难度 {item.difficulty}</span>
        {item.speechText ? (
          <span className="badge muted-badge" title={item.speechText}>
            {hasSpeech ? '有读音' : '无读音'}
          </span>
        ) : null}
      </div>
      <div className="card-actions">
        <button type="button" className="mini-btn" disabled={busyGlyph} onClick={onGenerateGlyph}>
          {busyGlyph ? '生成中…' : item.glyphImageUrl ? '重做字图' : '生成字图'}
        </button>
        <button
          type="button"
          className="mini-btn"
          disabled={speaking || (!hasSpeech && busySpeech)}
          onClick={onPlay}
        >
          {speaking ? '播放中…' : '读音'}
        </button>
        <button type="button" className="mini-btn" disabled={busySpeech} onClick={onRegenSpeech}>
          {busySpeech ? '生成中…' : '重新生成'}
        </button>
      </div>
    </article>
  )
}
