import { useOverview } from '../../api/dashboard'
import { useChildStore } from '../../store/childStore'

export function TopBar() {
  const childId = useChildStore((s) => s.childId)
  const { data } = useOverview(childId)
  if (!data) return <div className="h-16" />

  const done = data.counts.mastered + data.counts.review_due
  const pct = Math.round((done / data.total_kp) * 100)

  return (
    <header className="flex items-center gap-6 px-6 py-4 bg-white/70 backdrop-blur">
      <div className="text-lg font-semibold text-brand-700">
        🌸 {data.child.name} · {data.child.grade}学习情况
      </div>
      <div className="flex-1 max-w-md">
        <div className="flex justify-between text-xs text-slate-500 mb-1">
          <span>总掌握进度</span>
          <span>{done}/{data.total_kp}（{pct}%）</span>
        </div>
        <div className="h-2 rounded-full bg-brand-100">
          <div className="h-2 rounded-full bg-brand-500 transition-all"
               style={{ width: `${pct}%` }} />
        </div>
      </div>
      <div className="flex items-center gap-3 text-sm">
        <span className="rounded-full bg-brand-100 px-3 py-1">🌸 {data.child.flowers}</span>
        <span className="rounded-full bg-amber-100 px-3 py-1">🔥 连续 {data.streak_days} 天</span>
      </div>
    </header>
  )
}
