import { useEffect, useRef } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useExtraPlan, usePlanDetail } from '../api/plans'
import { useOverview } from '../api/dashboard'
import { useChildStore } from '../store/childStore'
import { StarRow } from '../components/StarRow'
import { useSound } from '../hooks/useSound'
import { useSpeech } from '../hooks/useSpeech'

// 一星也是通过，文案里不出现"失败""不及格"这类词。
// show 带 emoji 给眼睛看，say 是纯文字给 TTS 念——emoji 会被读成"派对喇叭"。
const PRAISE: Record<number, { show: string; say: string }> = {
  3: { show: '太棒啦！全都答对了 🎉', say: '太棒啦，全都答对了' },
  2: { show: '做得很好！继续加油 💪', say: '做得很好，继续加油' },
  1: { show: '今天也很努力 🌱', say: '今天也很努力哦' },
}

export function KidDone() {
  const childId = useChildStore((s) => s.childId)
  const nav = useNavigate()
  const { planId: planIdParam } = useParams()
  const planId = Number(planIdParam) || 0
  const { data } = usePlanDetail(childId, planId)
  const { data: ov } = useOverview(childId)
  const extra = useExtraPlan(childId)
  const sound = useSound()
  const speech = useSpeech()
  const cheered = useRef(false)

  const plan = data?.plan
  const stars = plan?.stars ?? 1

  useEffect(() => {
    if (!plan || cheered.current) return
    cheered.current = true
    sound.playCheer()
    speech.speak(PRAISE[stars]?.say ?? '做完啦', 'zh-CN')
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [plan])

  if (!plan) {
    return (
      <div className="flex h-full items-center justify-center">
        <span className="animate-bob text-7xl">🌸</span>
      </div>
    )
  }

  const minutes = Math.max(1, Math.round(plan.duration_sec / 60))

  return (
    <div className="flex h-full items-center justify-center px-6">
      <div className="w-full max-w-lg animate-floatUp rounded-[2.5rem] bg-candy-paper px-12 py-10 text-center shadow-pop">
        <StarRow stars={stars} size={64} animate />

        <h1 className="mt-6 text-4xl font-bold text-candy-ink">
          {PRAISE[stars]?.show ?? '做完啦'}
        </h1>

        <div className="mt-8 grid grid-cols-3 gap-3">
          <Tile emoji="✅" value={`${plan.correct_count}`} label="答对" />
          <Tile emoji="📝" value={`${plan.target_count}`} label="题目" />
          <Tile emoji="⏱️" value={`${minutes}`} label="分钟" />
        </div>

        <div className="mt-8 rounded-kid bg-brand-50 py-5">
          <div className="text-2xl font-bold text-brand-700">
            🌸 得到 {plan.flowers} 朵小红花
          </div>
          <div className="mt-1 text-base text-candy-mute">
            现在一共有 {ov?.child.flowers ?? 0} 朵
          </div>
        </div>

        <div className="mt-8 flex flex-col gap-3">
          <button
            type="button"
            onClick={() => {
              sound.unlock()
              speech.prime()
              extra.mutate(undefined, {
                onSuccess: (res) => nav(`/task/${res.plan.id}`, { replace: true }),
              })
            }}
            disabled={extra.isPending}
            className="min-h-[92px] w-full rounded-kid text-3xl font-bold text-candy-ink
                       transition-all active:translate-y-[5px] disabled:opacity-50"
            style={{ backgroundColor: '#9BE6C4', boxShadow: '0 6px 0 #6FBE9B' }}
          >
            {extra.isPending ? '准备中…' : '再来一组 🍬'}
          </button>
          <button
            type="button"
            onClick={() => nav('/', { replace: true })}
            className="min-h-[80px] w-full rounded-kid bg-white text-2xl font-bold text-candy-mute
                       transition-all active:translate-y-[4px]"
            style={{ boxShadow: '0 5px 0 rgba(61,50,48,0.12)' }}
          >
            今天就到这里 🌙
          </button>
        </div>

        {extra.isError && (
          <p className="mt-4 text-base text-candy-mute">今天已经练得很多啦，明天再来 🌙</p>
        )}
      </div>
    </div>
  )
}

function Tile({ emoji, value, label }: { emoji: string; value: string; label: string }) {
  return (
    <div className="rounded-kid bg-white/80 py-4">
      <div className="text-2xl">{emoji}</div>
      <div className="mt-1 text-3xl font-bold text-candy-ink">{value}</div>
      <div className="text-sm text-candy-mute">{label}</div>
    </div>
  )
}
