export function ProgressRing({ value, total, size = 96 }: {
  value: number; total: number; size?: number
}) {
  const pct = total > 0 ? value / total : 0
  const r = size / 2 - 8
  const c = 2 * Math.PI * r

  return (
    <svg width={size} height={size} className="-rotate-90">
      <circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke="#FFE4EE" strokeWidth={10} />
      <circle
        cx={size / 2} cy={size / 2} r={r} fill="none" stroke="#F472A6" strokeWidth={10}
        strokeLinecap="round" strokeDasharray={c} strokeDashoffset={c * (1 - pct)}
      />
      <text x="50%" y="50%" dominantBaseline="middle" textAnchor="middle"
            className="rotate-90 fill-slate-700 text-sm font-semibold"
            style={{ transformOrigin: 'center', transformBox: 'fill-box' as never }}>
        {Math.round(pct * 100)}%
      </text>
    </svg>
  )
}
