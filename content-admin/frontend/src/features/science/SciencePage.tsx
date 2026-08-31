import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import {
  listScience,
  patchScienceItem,
  speechAudioURL,
  syncScience,
} from '../../api/science'
import type { ScienceItem } from '../../api/scienceTypes'

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

export function SciencePage() {
  const queryClient = useQueryClient()
  const [view, setView] = useState<ViewMode>('groups')
  const [filter, setFilter] = useState<FilterMode>('')
  const [q, setQ] = useState('')
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})
  const [busyKp, setBusyKp] = useState<number | null>(null)
  const [speakingKp, setSpeakingKp] = useState<number | null>(null)
  const [speechError, setSpeechError] = useState('')

  const listQuery = useQuery({
    queryKey: ['science', 'items', view, filter],
    queryFn: () => listScience({ view, needsSenseImage: filter }),
  })

  const syncMutation = useMutation({
    mutationFn: syncScience,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['science'] }),
  })

  const overrideMutation = useMutation({
    mutationFn: ({ kpId, value }: { kpId: number; value: boolean }) =>
      patchScienceItem(kpId, value),
    onMutate: ({ kpId }) => setBusyKp(kpId),
    onSettled: () => {
      setBusyKp(null)
      void queryClient.invalidateQueries({ queryKey: ['science'] })
    },
  })

  const onPlaySpeech = async (item: ScienceItem) => {
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

  const items = useMemo(() => {
    const raw =
      view === 'table'
        ? (listQuery.data?.items ?? [])
        : (listQuery.data?.groups ?? []).flatMap((g) => g.items)
    if (!q.trim()) return raw
    const needle = q.trim().toLowerCase()
    return raw.filter((item) => item.title.toLowerCase().includes(needle))
  }, [listQuery.data, view, q])

  const error =
    speechError ||
    (listQuery.error instanceof Error && listQuery.error.message) ||
    (syncMutation.error instanceof Error && syncMutation.error.message) ||
    (overrideMutation.error instanceof Error && overrideMutation.error.message) ||
    ''

  return (
    <section className="literacy-page" aria-label="科普素材">
      <div className="page-heading">
        <div>
          <p className="eyebrow">CONTENT / SCIENCE</p>
          <h1>科普素材</h1>
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
          placeholder="搜索知识点"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <span className="muted">{listQuery.data ? `共 ${listQuery.data.total} 项` : ''}</span>
      </div>

      {error ? <div className="error-panel" role="alert">{error}</div> : null}
      {syncMutation.data ? (
        <div className="info-panel">
          已同步 {syncMutation.data.upserted} / {syncMutation.data.total} 项
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
              items: g.items.filter(
                (item) => !q.trim() || item.title.toLowerCase().includes(q.trim().toLowerCase()),
              ),
            }))
            .filter((g) => g.items.length > 0)
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
                        {g.items.length} 项 · {closed ? '展开' : '收起'}
                      </span>
                    </button>
                  </div>
                  {!closed ? (
                    <div className="char-grid">
                      {g.items.map((item) => (
                        <ItemCard
                          key={item.kpId}
                          item={item}
                          busy={busyKp === item.kpId}
                          speaking={speakingKp === item.kpId}
                          onPlaySpeech={() => void onPlaySpeech(item)}
                          onToggleSense={() =>
                            overrideMutation.mutate({
                              kpId: item.kpId,
                              value: !item.effectiveNeedsSenseImage,
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
                <th>知识点</th>
                <th>组别</th>
                <th>义图标记</th>
                <th>字图</th>
                <th>义图</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <tr key={item.kpId}>
                  <td className="char-cell">{item.title}</td>
                  <td>{item.moduleName}</td>
                  <td>{item.effectiveNeedsSenseImage ? '要义图' : '不要义图'}</td>
                  <td>
                    {item.glyphImageUrl ? (
                      <img className="glyph-thumb" src={item.glyphImageUrl} alt={item.title} />
                    ) : (
                      '—'
                    )}
                  </td>
                  <td>
                    {item.senseImageUrl ? (
                      <img className="glyph-thumb" src={item.senseImageUrl} alt={`${item.title}义图`} />
                    ) : (
                      '—'
                    )}
                  </td>
                  <td className="table-actions">
                    <button
                      type="button"
                      className="mini-btn"
                      disabled={speakingKp === item.kpId}
                      onClick={() => void onPlaySpeech(item)}
                    >
                      {speakingKp === item.kpId ? '…' : item.speechAudioUrl ? '读音✓' : '读音'}
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

function ItemCard({
  item,
  busy,
  speaking,
  onPlaySpeech,
  onToggleSense,
}: {
  item: ScienceItem
  busy: boolean
  speaking: boolean
  onPlaySpeech: () => void
  onToggleSense: () => void
}) {
  return (
    <article className={`char-card${item.effectiveNeedsSenseImage ? ' needs-sense' : ''}`}>
      {item.glyphImageUrl ? (
        <img className="glyph-preview" src={item.glyphImageUrl} alt={item.title} />
      ) : (
        <div className="char-glyph">{item.title}</div>
      )}
      {item.senseImageUrl ? (
        <img className="sense-preview" src={item.senseImageUrl} alt={`${item.title}义图`} />
      ) : item.effectiveNeedsSenseImage ? (
        <div className="sense-placeholder">义图未生成</div>
      ) : null}
      <button
        type="button"
        className={`sense-tag${item.effectiveNeedsSenseImage ? ' yes' : ' no'}`}
        disabled={busy}
        title="点击切换要/不要义图"
        onClick={onToggleSense}
      >
        {item.effectiveNeedsSenseImage ? '要义图' : '不要义图'}
        {item.speechAudioUrl ? ' · 有读音' : ''}
      </button>
      <div className="card-actions">
        <button type="button" className="mini-btn" disabled={speaking} onClick={onPlaySpeech}>
          {speaking ? '…' : item.speechAudioUrl ? '读音✓' : '读音'}
        </button>
      </div>
    </article>
  )
}
