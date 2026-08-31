import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import {
  listEnglish,
  patchEnglishWord,
  speechAudioURL,
  syncEnglish,
} from '../../api/english'
import type { EnglishWord } from '../../api/englishTypes'

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

export function EnglishPage() {
  const queryClient = useQueryClient()
  const [view, setView] = useState<ViewMode>('groups')
  const [filter, setFilter] = useState<FilterMode>('')
  const [q, setQ] = useState('')
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})
  const [busyKp, setBusyKp] = useState<number | null>(null)
  const [speakingKp, setSpeakingKp] = useState<number | null>(null)
  const [speechError, setSpeechError] = useState('')

  const listQuery = useQuery({
    queryKey: ['english', 'words', view, filter],
    queryFn: () => listEnglish({ view, needsSenseImage: filter }),
  })

  const syncMutation = useMutation({
    mutationFn: syncEnglish,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['english'] }),
  })

  const overrideMutation = useMutation({
    mutationFn: ({ kpId, value }: { kpId: number; value: boolean }) =>
      patchEnglishWord(kpId, value),
    onMutate: ({ kpId }) => setBusyKp(kpId),
    onSettled: () => {
      setBusyKp(null)
      void queryClient.invalidateQueries({ queryKey: ['english'] })
    },
  })

  const onPlaySpeech = async (word: EnglishWord) => {
    setSpeechError('')
    setSpeakingKp(word.kpId)
    try {
      await playSpeech(word.kpId, word.speechAudioUrl)
    } catch (e) {
      setSpeechError(e instanceof Error ? e.message : '读音失败')
    } finally {
      setSpeakingKp(null)
    }
  }

  const words = useMemo(() => {
    const raw =
      view === 'table'
        ? (listQuery.data?.words ?? [])
        : (listQuery.data?.groups ?? []).flatMap((g) => g.words)
    if (!q.trim()) return raw
    const needle = q.trim().toLowerCase()
    return raw.filter((w) => w.wordText.toLowerCase().includes(needle))
  }, [listQuery.data, view, q])

  const error =
    speechError ||
    (listQuery.error instanceof Error && listQuery.error.message) ||
    (syncMutation.error instanceof Error && syncMutation.error.message) ||
    (overrideMutation.error instanceof Error && overrideMutation.error.message) ||
    ''

  return (
    <section className="literacy-page" aria-label="英语素材">
      <div className="page-heading">
        <div>
          <p className="eyebrow">CONTENT / ENGLISH</p>
          <h1>英语素材</h1>
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
          placeholder="搜索单词"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <span className="muted">{listQuery.data ? `共 ${listQuery.data.total} 词` : ''}</span>
      </div>

      {error ? <div className="error-panel" role="alert">{error}</div> : null}
      {syncMutation.data ? (
        <div className="info-panel">
          已同步 {syncMutation.data.upserted} / {syncMutation.data.total} 词
        </div>
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
              words: g.words.filter(
                (w) => !q.trim() || w.wordText.toLowerCase().includes(q.trim().toLowerCase()),
              ),
            }))
            .filter((g) => g.words.length > 0)
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
                      <span className="muted">
                        {g.words.length} 词 · {closed ? '展开' : '收起'}
                      </span>
                    </button>
                  </div>
                  {!closed ? (
                    <div className="char-grid">
                      {g.words.map((w) => (
                        <WordCard
                          key={w.kpId}
                          word={w}
                          busy={busyKp === w.kpId}
                          speaking={speakingKp === w.kpId}
                          onPlaySpeech={() => void onPlaySpeech(w)}
                          onToggleSense={() =>
                            overrideMutation.mutate({
                              kpId: w.kpId,
                              value: !w.effectiveNeedsSenseImage,
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
                <th>词</th>
                <th>组别</th>
                <th>义图标记</th>
                <th>字图</th>
                <th>义图</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {words.map((w) => (
                <tr key={w.kpId}>
                  <td className="char-cell">{w.wordText}</td>
                  <td>{w.moduleName}</td>
                  <td>{w.effectiveNeedsSenseImage ? '要义图' : '不要义图'}</td>
                  <td>
                    {w.glyphImageUrl ? (
                      <img className="glyph-thumb" src={w.glyphImageUrl} alt={w.wordText} />
                    ) : (
                      '—'
                    )}
                  </td>
                  <td>
                    {w.senseImageUrl ? (
                      <img className="glyph-thumb" src={w.senseImageUrl} alt={`${w.wordText}义图`} />
                    ) : (
                      '—'
                    )}
                  </td>
                  <td className="table-actions">
                    <button
                      type="button"
                      className="mini-btn"
                      disabled={speakingKp === w.kpId}
                      onClick={() => void onPlaySpeech(w)}
                    >
                      {speakingKp === w.kpId ? '…' : w.speechAudioUrl ? '读音✓' : '读音'}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}

function WordCard({
  word,
  busy,
  speaking,
  onPlaySpeech,
  onToggleSense,
}: {
  word: EnglishWord
  busy: boolean
  speaking: boolean
  onPlaySpeech: () => void
  onToggleSense: () => void
}) {
  return (
    <article className={`char-card${word.effectiveNeedsSenseImage ? ' needs-sense' : ''}`}>
      {word.glyphImageUrl ? (
        <img className="glyph-preview" src={word.glyphImageUrl} alt={word.wordText} />
      ) : (
        <div className="char-glyph">{word.wordText}</div>
      )}
      {word.senseImageUrl ? (
        <img className="sense-preview" src={word.senseImageUrl} alt={`${word.wordText}义图`} />
      ) : word.effectiveNeedsSenseImage ? (
        <div className="sense-placeholder">义图未生成</div>
      ) : null}
      <button
        type="button"
        className={`sense-tag${word.effectiveNeedsSenseImage ? ' yes' : ' no'}`}
        disabled={busy}
        title="点击切换要/不要义图"
        onClick={onToggleSense}
      >
        {word.effectiveNeedsSenseImage ? '要义图' : '不要义图'}
        {word.speechAudioUrl ? ' · 有读音' : ''}
      </button>
      <div className="card-actions">
        <button type="button" className="mini-btn" disabled={speaking} onClick={onPlaySpeech}>
          {speaking ? '…' : word.speechAudioUrl ? '读音✓' : '读音'}
        </button>
      </div>
    </article>
  )
}
