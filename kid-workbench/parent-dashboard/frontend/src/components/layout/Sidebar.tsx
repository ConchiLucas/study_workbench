import { NavLink } from 'react-router-dom'
import clsx from 'clsx'
import { useSubjects } from '../../api/dashboard'
import { useChildStore } from '../../store/childStore'

const extraNav = [
  { to: '/tasks', icon: '📋', label: '任务列表' },
  { to: '/rewards', icon: '🎁', label: '奖励商店' },
]

export function Sidebar() {
  const childId = useChildStore((s) => s.childId)
  const { data: subjects } = useSubjects(childId)

  return (
    <aside className="w-52 shrink-0 bg-brand-100/60 min-h-screen px-3 py-4 flex flex-col gap-1">
      <NavLink to="/" className={({ isActive }) => clsx(
        'rounded-xl2 px-3 py-2 font-medium flex items-center gap-2',
        isActive ? 'bg-white shadow-sm text-brand-700' : 'hover:bg-white/60',
      )}>
        <span>🏠</span> 学习总览
      </NavLink>

      <div className="mt-3 mb-1 px-3 text-xs text-slate-400">学科</div>
      {subjects?.map((s) => (
        <NavLink key={s.code} to={`/subjects/${s.code}`} className={({ isActive }) => clsx(
          'rounded-xl2 px-3 py-2 flex items-center justify-between',
          isActive ? 'bg-white shadow-sm text-brand-700' : 'hover:bg-white/60',
        )}>
          <span className="flex items-center gap-2">
            <span>{s.icon}</span>{s.name}
          </span>
          <span className="text-xs text-slate-400">
            {s.counts.mastered + s.counts.review_due}/{s.total}
          </span>
        </NavLink>
      ))}

      <div className="mt-3 mb-1 px-3 text-xs text-slate-400">其他</div>
      <NavLink to="/attention" className={({ isActive }) => clsx(
        'rounded-xl2 px-3 py-2 flex items-center gap-2',
        isActive ? 'bg-white shadow-sm text-brand-700' : 'hover:bg-white/60',
      )}>
        <span>🔍</span> 需要关注
      </NavLink>
      {extraNav.map((n) => (
        <NavLink key={n.to} to={n.to} className={({ isActive }) => clsx(
          'rounded-xl2 px-3 py-2 flex items-center gap-2',
          isActive ? 'bg-white shadow-sm text-brand-700' : 'hover:bg-white/60',
        )}>
          <span>{n.icon}</span> {n.label}
        </NavLink>
      ))}
    </aside>
  )
}
