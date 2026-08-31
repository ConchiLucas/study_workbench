import { Link, useParams } from 'react-router-dom'
import { useTaskReview } from '../api/plans'
import { ReviewItemCard } from '../components/task/ReviewItemCard'
import { kidAppUrl } from '../lib/kidAppUrl'
import { useChildStore } from '../store/childStore'

const STATUS_TEXT = {
  pending: '还没开始',
  doing: '正在做',
  done: '已完成',
} as const

export function TaskDetailPage() {
  const childId = useChildStore((s) => s.childId)
  const { planId: raw } = useParams()
  const planId = Number(raw) || 0
  const { data, isLoading, error } = useTaskReview(childId, planId)

  if (isLoading) return <div className="text-sm text-slate-400">加载中…</div>
  if (error || !data) {
    return (
      <div className="space-y-3">
        <Link to="/tasks" className="text-sm text-sky-600 hover:underline">
          ← 返回任务列表
        </Link>
        <p className="text-sm text-rose-500">找不到这份任务</p>
      </div>
    )
  }

  const { plan, items } = data
  const acc =
    plan.done_count > 0
      ? Math.round((plan.correct_count / plan.done_count) * 100)
      : null
  const wrongKps = items.filter((it) => it.status === 'wrong')
  const label = plan.seq_no === 1 ? '练习' : `加餐 ${plan.seq_no - 1}`

  return (
    <div className="space-y-5">
      <Link to="/tasks" className="inline-block text-sm text-sky-600 hover:underline">
        ← 返回任务列表
      </Link>

      <section className="rounded-xl2 bg-white p-5 shadow-sm">
        <div className="flex flex-wrap items-baseline justify-between gap-2">
          <h2 className="text-lg font-semibold text-slate-700">
            {plan.plan_date} · {label}
          </h2>
          <span className="text-sm text-slate-500">{STATUS_TEXT[plan.status]}</span>
        </div>
        <div className="mt-3 flex flex-wrap gap-4 text-sm text-slate-600">
          <span>
            进度 {plan.done_count}/{plan.target_count}
          </span>
          {acc !== null && <span>正确率 {acc}%</span>}
          {plan.duration_sec > 0 && (
            <span>用时 {Math.round(plan.duration_sec / 60) || 1} 分钟</span>
          )}
          {plan.status === 'done' && (
            <>
              <span className="text-amber-500">{'★'.repeat(plan.stars)}</span>
              {plan.flowers > 0 && <span>🌸 {plan.flowers}</span>}
            </>
          )}
        </div>
        {plan.status !== 'done' && (
          <p className="mt-3 text-xs text-slate-400">
            孩子用 iPad 打开{' '}
            <a href={kidAppUrl} className="text-sky-600 hover:underline" target="_blank" rel="noreferrer">
              练一练
            </a>{' '}
            开始答题
          </p>
        )}
      </section>

      <div className="space-y-4">
        {items.map((item) => (
          <ReviewItemCard key={item.seq} item={item} />
        ))}
      </div>

      {wrongKps.length > 0 && (
        <section className="rounded-xl2 bg-rose-50 px-5 py-4">
          <h3 className="text-sm font-medium text-rose-800">这次没答对的知识点</h3>
          <ul className="mt-2 space-y-1 text-sm text-rose-700">
            {wrongKps.map((it) => (
              <li key={it.seq}>
                <Link
                  to={`/subjects/${it.subject_code}`}
                  className="hover:underline"
                >
                  {it.subject_name} · {it.kp_title}
                </Link>
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  )
}
