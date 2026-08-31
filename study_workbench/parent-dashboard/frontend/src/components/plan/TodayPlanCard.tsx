import clsx from 'clsx'
import { Link } from 'react-router-dom'
import { usePlanHistory, type Plan } from '../../api/plans'
import { kidAppUrl } from '../../lib/kidAppUrl'
import { useChildStore } from '../../store/childStore'
import { localDate } from '../../lib/date'

const STATUS_TEXT: Record<Plan['status'], string> = {
  pending: '还没开始',
  doing: '正在做',
  done: '已完成',
}

/** 主计划叫"今天的练习"，之后的都是家长额外加的，叫"加餐"。 */
function planLabel(p: Plan) {
  return p.seq_no === 1 ? '今天的练习' : `加餐 ${p.seq_no - 1}`
}

function accuracy(p: Plan) {
  if (p.done_count === 0) return null
  return Math.round((p.correct_count / p.done_count) * 100)
}

export function TodayPlanCard() {
  const childId = useChildStore((s) => s.childId)
  const { data: plans } = usePlanHistory(childId)
  const today = localDate()
  const list = plans?.filter((p) => p.plan_date === today) ?? []

  const totalDone = list.reduce((n, p) => n + p.done_count, 0)
  const totalMin = Math.round(list.reduce((n, p) => n + p.duration_sec, 0) / 60)

  return (
    <section className="rounded-xl2 bg-white p-5 shadow-sm">
      <header className="flex items-baseline justify-between">
        <h2 className="font-medium text-slate-700">今日答题</h2>
        <div className="flex gap-3 text-xs">
          <Link to="/tasks" className="text-sky-600 hover:underline">
            查看全部任务 →
          </Link>
          <a href={kidAppUrl} target="_blank" rel="noreferrer"
             className="text-sky-600 hover:underline">
            在 iPad 上打开答题页 →
          </a>
        </div>
      </header>

      {list.length === 0 ? (
        <p className="mt-3 text-sm text-slate-400">
          今天还没有练习记录。用 iPad 打开答题页，或去任务列表生成一份。
        </p>
      ) : (
        <>
          <div className="mt-4 space-y-3">
            {list.map((p) => {
              const acc = accuracy(p)
              return (
                <Link key={p.id} to={`/tasks/${p.id}`}
                      className="flex items-center gap-4 rounded-lg px-1 py-1 hover:bg-slate-50">
                  <div className="w-24 shrink-0 text-sm text-slate-600">{planLabel(p)}</div>

                  <div className="flex h-2.5 flex-1 overflow-hidden rounded-full bg-slate-100">
                    <i style={{ width: `${(p.correct_count / p.target_count) * 100}%` }}
                       className="bg-emerald-300" />
                    <i style={{ width: `${((p.done_count - p.correct_count) / p.target_count) * 100}%` }}
                       className="bg-rose-300" />
                  </div>

                  <div className="w-16 shrink-0 text-right text-sm tabular-nums text-slate-600">
                    {p.done_count}/{p.target_count}
                  </div>
                  <div className="w-14 shrink-0 text-right text-sm tabular-nums text-slate-500">
                    {acc === null ? '—' : `${acc}%`}
                  </div>
                  <div className={clsx('w-16 shrink-0 text-right text-xs',
                    p.status === 'done' ? 'text-amber-500' : 'text-slate-400')}>
                    {p.status === 'done' ? '★'.repeat(p.stars) : STATUS_TEXT[p.status]}
                  </div>
                </Link>
              )
            })}
          </div>

          <p className="mt-4 text-xs text-slate-400">
            今天共做 {totalDone} 道{totalMin > 0 && `，专注 ${totalMin} 分钟`}
            {list.length > 1 && `，含 ${list.length - 1} 次加餐`}
          </p>
        </>
      )}
    </section>
  )
}
