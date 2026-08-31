import { useParams } from 'react-router-dom'
import { useSubjects } from '../api/dashboard'
import { useChildStore } from '../store/childStore'
import { STATUS_ORDER, STATUS_STYLE } from '../theme'
import { ProgressRing } from '../components/stats/ProgressRing'
import { MasteryMatrix } from '../components/mastery/MasteryMatrix'

export function SubjectDetail() {
  const { code = '' } = useParams()
  const childId = useChildStore((s) => s.childId)
  const { data: subjects } = useSubjects(childId)
  const s = subjects?.find((x) => x.code === code)
  if (!s) return <div className="text-sm text-slate-400">加载中…</div>

  const done = s.counts.mastered + s.counts.review_due

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-6 rounded-xl2 bg-white p-5 shadow-sm flex-wrap">
        <ProgressRing value={done} total={s.total} size={110} />
        <div>
          <h2 className="text-xl font-semibold text-slate-700">{s.icon} {s.name}</h2>
          <div className="mt-1 text-sm text-slate-500">已掌握 {done} / {s.total} 个知识点</div>
          {s.week_new > 0 && (
            <div className="text-xs text-emerald-600">本周新掌握 {s.week_new} 个</div>
          )}
        </div>
        <div className="ml-auto grid grid-cols-5 gap-3 text-center">
          {STATUS_ORDER.map((k) => (
            <div key={k}>
              <div className="text-lg font-semibold" style={{ color: STATUS_STYLE[k].text }}>
                {s.counts[k]}
              </div>
              <div className="text-xs text-slate-400">{STATUS_STYLE[k].label}</div>
            </div>
          ))}
        </div>
      </div>

      <MasteryMatrix subject={code} />
    </div>
  )
}
