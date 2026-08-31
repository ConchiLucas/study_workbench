import clsx from 'clsx'

export function StatCard({ label, value, hint, color, onClick }: {
  label: string; value: number | string; hint?: string; color: string; onClick?: () => void
}) {
  return (
    <button
      onClick={onClick}
      className={clsx('rounded-xl2 bg-white p-4 text-left shadow-sm transition',
        onClick && 'hover:shadow-md')}
    >
      <div className="flex items-center gap-2 text-sm text-slate-500">
        <i className="inline-block h-3 w-3 rounded" style={{ backgroundColor: color }} />
        {label}
      </div>
      <div className="mt-1 text-3xl font-semibold text-slate-700">{value}</div>
      {hint && <div className="mt-1 text-xs text-slate-400">{hint}</div>}
    </button>
  )
}
