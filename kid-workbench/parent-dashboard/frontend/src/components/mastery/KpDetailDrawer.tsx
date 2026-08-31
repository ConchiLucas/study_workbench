import { useKpDetail, useMarkMastered, useUndoMark } from '../../api/dashboard'
import { useChildStore } from '../../store/childStore'
import { STATUS_STYLE } from '../../theme'
import { parseKpPayload, type KpContent } from '../../lib/kpPayload'

const SKILL_SHORT: Record<string, string> = {
  glyph_sense: '义',
  sense_char: '字',
  inword: '例',
  listen: '音',
}
const SKILL_FULL: Record<string, string> = {
  glyph_sense: '看字选义',
  sense_char: '看义选字',
  inword: '听例字选音',
  listen: '听单读选字母',
}

const DEFAULT_SKILLS: Record<string, { code: string; status: 'not_started'; accuracy: number; attempts: number }[]> = {
  literacy: [
    { code: 'glyph_sense', status: 'not_started', accuracy: 0, attempts: 0 },
    { code: 'sense_char', status: 'not_started', accuracy: 0, attempts: 0 },
  ],
  pinyin: [
    { code: 'inword', status: 'not_started', accuracy: 0, attempts: 0 },
    { code: 'listen', status: 'not_started', accuracy: 0, attempts: 0 },
  ],
}

export function KpDetailDrawer() {
  const { childId, kpDrawerId, closeKp } = useChildStore()
  const { data } = useKpDetail(childId, kpDrawerId)
  const mark = useMarkMastered(childId)
  const undo = useUndoMark(childId)

  if (kpDrawerId === null) return null

  const style = data ? STATUS_STYLE[data.status] : null
  const hasParentMark = data?.history.some((h) => h.source === 'parent_mark') ?? false
  const content = data ? parseKpPayload(data.payload) : { kind: 'unknown' as const }

  return (
    <>
      <div className="fixed inset-0 bg-slate-900/20 z-40" onClick={closeKp} />
      <div className="fixed right-0 top-0 h-full w-96 overflow-y-auto bg-white p-5 shadow-xl z-50">
        {!data ? <div className="text-sm text-slate-400">加载中…</div> : (
          <div className="space-y-5">
            <div className="flex items-start justify-between">
              <div>
                <div className="text-xs text-slate-400">
                  {data.subject_name} · {data.module_name}
                </div>
                <h3 className="text-xl font-semibold text-slate-700">{data.title}</h3>
              </div>
              <button className="text-slate-400" onClick={closeKp}>✕</button>
            </div>

            <KpContentBlock content={content} />

            <div className="rounded-xl2 p-4" style={{ backgroundColor: style!.bg }}>
              <div className="font-medium" style={{ color: style!.text }}>{style!.label}</div>
              <div className="mt-1 text-xs" style={{ color: style!.text }}>{style!.desc}</div>
            </div>

            {(data.subject_code === 'literacy' || data.subject_code === 'pinyin') && (
              <div>
                <div className="mb-2 text-sm font-medium text-slate-600">题型掌握</div>
                <div className="space-y-2">
                  {(data.skills ?? DEFAULT_SKILLS[data.subject_code] ?? []).map((sk) => {
                    const st = STATUS_STYLE[sk.status]
                    return (
                      <div key={sk.code} className="flex items-center gap-3 rounded-xl border border-slate-100 bg-slate-50 px-3 py-2">
                        <span className="grid h-9 w-9 place-items-center rounded-lg text-sm font-bold text-white"
                          style={{ backgroundColor: st.ring }}>
                          {SKILL_SHORT[sk.code] ?? sk.code}
                        </span>
                        <div className="min-w-0 flex-1">
                          <div className="text-sm font-medium text-slate-700">{SKILL_FULL[sk.code] ?? sk.code}</div>
                          <div className="text-[11px] text-slate-400">
                            练 {sk.attempts} 次 · 正确率 {Math.round(sk.accuracy * 100)}%
                          </div>
                        </div>
                        <span className="shrink-0 rounded-full px-2 py-0.5 text-[11px] font-semibold"
                          style={{ backgroundColor: st.bg, color: st.text }}>
                          {st.label}
                        </span>
                      </div>
                    )
                  })}
                </div>
              </div>
            )}

            {(data.subject_code === 'literacy' || data.subject_code === 'pinyin') && (
              <div className="text-sm font-medium text-slate-600">
                {data.subject_code === 'literacy' ? '字级汇总' : '音级汇总'}
              </div>
            )}

            <dl className="grid grid-cols-2 gap-3 text-sm">
              <Stat label="练习次数" value={String(data.attempts)} />
              <Stat label="正确率" value={`${Math.round(data.accuracy * 100)}%`} />
              <Stat label="当前连对" value={String(data.streak)} />
              <Stat label="历史最佳" value={String(data.best_streak)} />
              <Stat label="下次复习" value={data.due_at ? data.due_at.slice(0, 10) : '—'} />
              <Stat label="首次掌握" value={data.mastered_at ? data.mastered_at.slice(0, 10) : '—'} />
            </dl>

            <div className="flex gap-2">
              <button
                className="flex-1 rounded-xl2 bg-brand-500 py-2 text-white disabled:opacity-50"
                disabled={mark.isPending}
                onClick={() => mark.mutate(data.kp_id)}
              >
                标记为已学会
              </button>
              {hasParentMark && (
                <button
                  className="rounded-xl2 border border-slate-200 px-3 py-2 text-sm text-slate-500"
                  disabled={undo.isPending}
                  onClick={() => undo.mutate(data.kp_id)}
                >
                  撤销标记
                </button>
              )}
            </div>

            <div>
              <div className="mb-2 text-sm font-medium text-slate-600">
                作答记录（{data.history.length}）
              </div>
              <ul className="space-y-1 text-xs">
                {[...data.history].reverse().map((h, i) => (
                  <li key={i} className="flex items-center justify-between rounded-lg bg-slate-50 px-3 py-2">
                    <span className="flex items-center gap-2">
                      {(data.subject_code === 'literacy' || data.subject_code === 'pinyin') && (
                        h.source === 'parent_mark' ? (
                          <span className="rounded bg-brand-500 px-1.5 py-0.5 text-[10px] font-bold text-white">标</span>
                        ) : h.skill_code ? (
                          <span className="rounded bg-teal-600 px-1.5 py-0.5 text-[10px] font-bold text-white">
                            {SKILL_SHORT[h.skill_code] ?? h.skill_code}
                          </span>
                        ) : null
                      )}
                      <span>{h.at.slice(0, 16).replace('T', ' ')}</span>
                    </span>
                    <span className="flex items-center gap-2">
                      {h.source === 'parent_mark'
                        ? <em className="text-brand-700">家长标记</em>
                        : <span>{(h.cost_ms / 1000).toFixed(1)}s</span>}
                      <span>{h.is_correct ? '✅' : '❌'}</span>
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        )}
      </div>
    </>
  )
}

/** 古诗 / 科普 / 逻辑的正文；算术识字等没有可读 payload 时不渲染。 */
function KpContentBlock({ content }: { content: KpContent }) {
  switch (content.kind) {
    case 'poem':
      return (
        <div className="rounded-xl2 bg-gradient-to-b from-rose-50 to-amber-50 px-5 py-6 text-center">
          {content.author && (
            <div className="text-sm text-slate-500">〔{content.author}〕</div>
          )}
          <div className="mt-3 space-y-2 text-lg leading-relaxed text-slate-800">
            {content.lines.map((line, i) => (
              <p key={i}>{line}</p>
            ))}
          </div>
        </div>
      )
    case 'fact':
      return (
        <div className="rounded-xl2 bg-emerald-50 px-4 py-4">
          {content.emoji && <div className="mb-2 text-3xl">{content.emoji}</div>}
          <div className="text-sm text-slate-600">{content.q}</div>
          <div className="mt-2 text-base font-medium text-emerald-800">答：{content.a}</div>
        </div>
      )
    case 'logic':
      return (
        <div className="rounded-xl2 bg-violet-50 px-4 py-4">
          <div className="text-sm text-slate-600">{content.prompt}</div>
          {content.seq.length > 0 && (
            <div className="mt-3 flex flex-wrap gap-2 text-2xl">
              {content.seq.map((it, i) => (
                <span key={i}>{it}</span>
              ))}
              <span className="text-slate-400">？</span>
            </div>
          )}
          <div className="mt-2 text-sm text-violet-800">答：{content.answer}</div>
        </div>
      )
    case 'chengyu':
      return (
        <div className="rounded-xl2 bg-amber-50 px-4 py-4">
          {content.pinyin && (
            <div className="text-sm text-amber-700">{content.pinyin}</div>
          )}
          <div className="mt-2 text-base font-medium text-slate-800">{content.meaning}</div>
          {content.example && (
            <div className="mt-2 text-sm text-slate-600">例：{content.example}</div>
          )}
        </div>
      )
    case 'phrase':
      return (
        <div className="rounded-xl2 bg-sky-50 px-4 py-4">
          <div className="text-base font-medium text-sky-900">{content.zh}</div>
        </div>
      )
    default:
      return null
  }
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl2 bg-slate-50 px-3 py-2">
      <dt className="text-xs text-slate-400">{label}</dt>
      <dd className="font-medium text-slate-700">{value}</dd>
    </div>
  )
}
