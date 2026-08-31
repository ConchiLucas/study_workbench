import { useNavigate } from 'react-router-dom'
import { useOverview, useSubjects } from '../api/dashboard'
import { useChildStore } from '../store/childStore'
import { STATUS_STYLE } from '../theme'
import { StatCard } from '../components/stats/StatCard'
import { ProgressRing } from '../components/stats/ProgressRing'
import { MasteryMatrix } from '../components/mastery/MasteryMatrix'
import { TrendChart } from '../components/stats/TrendChart'
import { TodayPlanCard } from '../components/plan/TodayPlanCard'

export function Overview() {
  const childId = useChildStore((s) => s.childId)
  const nav = useNavigate()
  const { data: ov } = useOverview(childId)
  const { data: subjects } = useSubjects(childId)

  if (!ov) return <div className="text-sm text-slate-400">加载中…</div>
  const done = ov.counts.mastered + ov.counts.review_due

  return (
    <div className="space-y-6">
      <section className="grid grid-cols-2 gap-4 lg:grid-cols-5">
        <div className="flex items-center gap-4 rounded-xl2 bg-white p-4 shadow-sm">
          <ProgressRing value={done} total={ov.total_kp} />
          <div>
            <div className="text-sm text-slate-500">总掌握进度</div>
            <div className="text-2xl font-semibold text-slate-700">{done}/{ov.total_kp}</div>
            <div className="text-xs text-slate-400">今天学了 {ov.today.practice_min} 分钟</div>
          </div>
        </div>
        <StatCard label="已掌握" value={ov.counts.mastered} color={STATUS_STYLE.mastered.bg}
                  hint={`本周新增 ${ov.week_delta.mastered}`} />
        <StatCard label="学习中" value={ov.counts.learning} color={STATUS_STYLE.learning.bg}
                  hint="练过但还不稳" />
        <StatCard label="需巩固" value={ov.counts.shaky} color={STATUS_STYLE.shaky.bg}
                  hint="反复出错，点我查看" onClick={() => nav('/attention')} />
        <StatCard label="待复习" value={ov.counts.review_due} color={STATUS_STYLE.review_due.bg}
                  hint="今天该复习了" onClick={() => nav('/attention')} />
      </section>

      <TodayPlanCard />

      <section className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        {subjects?.map((s) => (
          <button key={s.code} onClick={() => nav(`/subjects/${s.code}`)}
                  className="rounded-xl2 bg-white p-4 text-left shadow-sm hover:shadow-md">
            <div className="flex items-center justify-between">
              <span className="font-medium text-slate-700">{s.icon} {s.name}</span>
              {s.week_new > 0 && (
                <span className="text-xs text-emerald-600">本周 +{s.week_new}</span>
              )}
            </div>
            <div className="mt-2 flex h-2 overflow-hidden rounded-full bg-slate-100">
              {(['mastered', 'review_due', 'learning', 'shaky'] as const).map((k) => (
                <i key={k} style={{
                  width: `${(s.counts[k] / s.total) * 100}%`,
                  backgroundColor: STATUS_STYLE[k].bg,
                }} />
              ))}
            </div>
            <div className="mt-1 text-xs text-slate-400">
              已掌握 {s.counts.mastered + s.counts.review_due}/{s.total}
            </div>
          </button>
        ))}
      </section>

      <TrendChart days={30} />

      <section className="space-y-6">
        {subjects?.filter((s) => s.counts.not_started < s.total).map((s) => (
          <MasteryMatrix key={s.code} subject={s.code} />
        ))}
      </section>
    </div>
  )
}
