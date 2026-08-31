import { useState } from 'react'
import { CalendarHeatmap } from '../components/stats/CalendarHeatmap'
import { TrendChart } from '../components/stats/TrendChart'
import { TaskRow } from '../components/task/TaskRow'
import {
  useGenerateToday,
  usePlanHistory,
  type PlanStatus,
} from '../api/plans'
import { useChildStore } from '../store/childStore'
import { localDate } from '../lib/date'

const FILTERS: { value: PlanStatus | ''; label: string }[] = [
  { value: '', label: '全部' },
  { value: 'done', label: '已完成' },
  { value: 'doing', label: '进行中' },
  { value: 'pending', label: '未开始' },
]

export function TasksPage() {
  const childId = useChildStore((s) => s.childId)
  const [status, setStatus] = useState<PlanStatus | ''>('')
  const { data: plans, isLoading } = usePlanHistory(childId, status)
  const generate = useGenerateToday(childId)
  const today = localDate()

  const list = plans ?? []
  const todayPlans = list.filter((p) => p.plan_date === today)
  const showGenerate = !status && todayPlans.length === 0 && !isLoading

  return (
    <div className="space-y-6">
      <section className="rounded-xl2 bg-white p-5 shadow-sm">
        <header className="mb-4 flex flex-wrap items-center justify-between gap-3">
          <h2 className="font-semibold text-slate-700">任务列表</h2>
          <div className="flex gap-1 rounded-lg bg-slate-100 p-1 text-xs">
            {FILTERS.map((f) => (
              <button
                key={f.value || 'all'}
                onClick={() => setStatus(f.value)}
                className={
                  status === f.value
                    ? 'rounded-md bg-white px-2.5 py-1 font-medium text-brand-700 shadow-sm'
                    : 'rounded-md px-2.5 py-1 text-slate-500 hover:text-slate-700'
                }
              >
                {f.label}
              </button>
            ))}
          </div>
        </header>

        {showGenerate && (
          <div className="mb-4 flex items-center justify-between rounded-xl2 border border-dashed border-slate-200 px-4 py-3">
            <p className="text-sm text-slate-500">今天还没有任务</p>
            <button
              disabled={generate.isPending}
              onClick={() => generate.mutate()}
              className="rounded-full bg-brand-500 px-4 py-1.5 text-sm text-white disabled:bg-slate-300"
            >
              {generate.isPending ? '生成中…' : '生成今天的任务'}
            </button>
          </div>
        )}

        {isLoading && <p className="text-sm text-slate-400">加载中…</p>}

        {!isLoading && list.length === 0 && !showGenerate && (
          <p className="text-sm text-slate-400">还没有任务记录</p>
        )}

        <ul className="space-y-2">
          {list.map((p) => (
            <li key={p.id}>
              <TaskRow plan={p} />
            </li>
          ))}
        </ul>
      </section>

      <CalendarHeatmap months={6} />
      <TrendChart days={60} />
    </div>
  )
}
