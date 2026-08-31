import { useAttention } from '../api/dashboard'
import { useChildStore } from '../store/childStore'
import { STATUS_STYLE } from '../theme'

export function Attention() {
  const { childId, openKp } = useChildStore()
  const { data } = useAttention(childId, 30)
  if (!data) return <div className="text-sm text-slate-400">加载中…</div>

  if (data.length === 0) {
    return (
      <div className="rounded-xl2 bg-white p-10 text-center shadow-sm">
        <div className="text-4xl">🎉</div>
        <div className="mt-2 text-slate-600">当前没有需要特别关注的知识点</div>
      </div>
    )
  }

  return (
    <div className="rounded-xl2 bg-white p-5 shadow-sm">
      <h3 className="mb-1 font-semibold text-slate-700">需要关注（{data.length}）</h3>
      <p className="mb-4 text-xs text-slate-400">反复出错和到期该复习的知识点，建议今天陪练这些</p>
      <ul className="divide-y divide-slate-100">
        {data.map((it) => {
          const style = STATUS_STYLE[it.status]
          return (
            <li key={it.kp_id} className="flex items-center gap-4 py-3 flex-wrap">
              <span className="rounded-lg px-2 py-1 text-xs"
                    style={{ backgroundColor: style.bg, color: style.text }}>
                {style.label}
              </span>
              <button className="w-28 text-left font-medium text-slate-700 hover:text-brand-700"
                      onClick={() => openKp(it.kp_id)}>
                {it.title}
              </button>
              <span className="w-32 text-xs text-slate-400">
                {it.subject_name} · {it.module_name}
              </span>
              <div className="flex-1 min-w-[80px]">
                <div className="h-2 overflow-hidden rounded-full bg-slate-100">
                  <div className="h-2 rounded-full bg-rose-400"
                       style={{ width: `${Math.round(it.accuracy * 100)}%` }} />
                </div>
              </div>
              <span className="w-24 text-right text-xs text-slate-500">
                正确率 {Math.round(it.accuracy * 100)}%
              </span>
              <span className="w-20 text-right text-xs text-rose-500">错 {it.wrong_count} 次</span>
            </li>
          )
        })}
      </ul>
    </div>
  )
}
