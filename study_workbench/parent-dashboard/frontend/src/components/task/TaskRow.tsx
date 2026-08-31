import clsx from 'clsx'
import { Link } from 'react-router-dom'
import type { PlanSummary } from '../../api/plans'

const STATUS_TEXT: Record<PlanSummary['status'], string> = {
  pending: '还没开始',
  doing: '正在做',
  done: '已完成',
}

function formatDate(date: string) {
  const [, m, d] = date.split('-').map(Number)
  return `${m}月${d}日`
}

function planLabel(p: PlanSummary) {
  const kind = p.seq_no === 1 ? '练习' : `加餐${p.seq_no - 1}`
  return `${formatDate(p.plan_date)} ${kind}`
}

function accuracy(p: PlanSummary) {
  if (p.done_count === 0) return null
  return Math.round((p.correct_count / p.done_count) * 100)
}

function formatDuration(sec: number) {
  if (sec <= 0) return null
  if (sec < 60) return `${sec} 秒`
  return `${Math.round(sec / 60)} 分钟`
}

export function TaskRow({ plan }: { plan: PlanSummary }) {
  const acc = accuracy(plan)
  const dur = formatDuration(plan.duration_sec)

  return (
    <Link
      to={`/tasks/${plan.id}`}
      className="flex items-center gap-3 rounded-xl2 bg-slate-50 px-4 py-3 transition hover:bg-slate-100"
    >
      <div className="w-28 shrink-0 text-sm font-medium text-slate-700">
        {planLabel(plan)}
      </div>

      <div className="hidden min-w-0 flex-1 flex-wrap gap-1 sm:flex">
        {plan.subjects?.map((s) => (
          <span
            key={s.code}
            className="rounded-md bg-white px-1.5 py-0.5 text-xs text-slate-500"
          >
            {s.icon} {s.name} {s.count}
          </span>
        ))}
      </div>

      <div className="flex h-2.5 w-24 shrink-0 overflow-hidden rounded-full bg-slate-200 sm:w-32">
        <i
          style={{ width: `${(plan.correct_count / plan.target_count) * 100}%` }}
          className="bg-emerald-300"
        />
        <i
          style={{
            width: `${((plan.done_count - plan.correct_count) / plan.target_count) * 100}%`,
          }}
          className="bg-rose-300"
        />
      </div>

      <div className="w-14 shrink-0 text-right text-sm tabular-nums text-slate-600">
        {plan.done_count}/{plan.target_count}
      </div>
      <div className="hidden w-12 shrink-0 text-right text-sm tabular-nums text-slate-500 sm:block">
        {acc === null ? '—' : `${acc}%`}
      </div>
      <div className="hidden w-14 shrink-0 text-right text-xs text-slate-400 sm:block">
        {dur ?? '—'}
      </div>
      <div
        className={clsx(
          'w-16 shrink-0 text-right text-xs',
          plan.status === 'done' ? 'text-amber-500' : 'text-slate-400',
        )}
      >
        {plan.status === 'done' ? '★'.repeat(plan.stars) : STATUS_TEXT[plan.status]}
      </div>
    </Link>
  )
}
