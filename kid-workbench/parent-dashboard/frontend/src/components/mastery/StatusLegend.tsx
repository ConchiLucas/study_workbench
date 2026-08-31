import { STATUS_ORDER, STATUS_STYLE } from '../../theme'

export function StatusLegend() {
  return (
    <div className="flex flex-wrap items-center gap-4 text-xs text-slate-500">
      {STATUS_ORDER.map((s) => (
        <span key={s} className="flex items-center gap-1.5" title={STATUS_STYLE[s].desc}>
          <i className="inline-block h-3 w-3 rounded" style={{ backgroundColor: STATUS_STYLE[s].bg }} />
          {STATUS_STYLE[s].label}
        </span>
      ))}
    </div>
  )
}
