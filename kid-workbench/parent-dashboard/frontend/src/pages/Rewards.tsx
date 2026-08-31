import { useOverview } from '../api/dashboard'
import { useRedeem, useRewards } from '../api/rewards'
import { useChildStore } from '../store/childStore'

export function Rewards() {
  const childId = useChildStore((s) => s.childId)
  const { data: ov } = useOverview(childId)
  const { data: rewards } = useRewards(childId)
  const redeem = useRedeem(childId)

  return (
    <div className="space-y-4">
      <div className="rounded-xl2 bg-white p-5 shadow-sm">
        <div className="text-sm text-slate-500">小红花余额</div>
        <div className="text-3xl font-semibold text-brand-700">🌸 {ov?.child.flowers ?? 0}</div>
      </div>
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        {rewards?.map((r) => (
          <div key={r.id} className="rounded-xl2 bg-white p-4 shadow-sm">
            <div className="font-medium text-slate-700">{r.name}</div>
            <div className="mt-1 text-xs text-slate-400">剩余 {r.stock} 次</div>
            <button
              disabled={(ov?.child.flowers ?? 0) < r.cost || r.stock <= 0 || redeem.isPending}
              onClick={() => redeem.mutate(r.id)}
              className="mt-3 w-full rounded-xl2 bg-brand-500 py-2 text-sm text-white disabled:bg-slate-300"
            >
              兑换（🌸 {r.cost}）
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}
