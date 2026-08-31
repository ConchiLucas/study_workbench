import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useAnswerItem, useFinishPlan, usePlanDetail, useStartPlan } from '../api/plans'
import type { PlanItem } from '../api/plans'
import { useChildStore } from '../store/childStore'
import { OptionButton, type OptionState } from '../components/OptionButton'
import { ProgressDots } from '../components/ProgressDots'
import { QuestionVisual } from '../components/QuestionVisual'
import { useSound } from '../hooks/useSound'
import { useSpeech } from '../hooks/useSpeech'

/** 答对后停留多久再进下一题：够看到勾，又不至于让孩子等。 */
const RIGHT_HOLD_MS = 900
/** 答错还能重试时，红色抖动 + 语音提示的停留时间。 */
const RETRY_HOLD_MS = 1500
/** 两次都错时展示正确答案的停留时间，要够看清。 */
const REVEAL_HOLD_MS = 2400
/** 提交后锁定输入：孩子一定会连点，不锁会直接把下一题也点掉。 */
const TAP_LOCK_MS = 800
/** 迟迟不作答就再读一遍题，而不是催促。 */
const REREAD_MS = 20_000

type Phase = 'asking' | 'right' | 'retry' | 'reveal' | 'submitting'

export function KidQuiz() {
  const childId = useChildStore((s) => s.childId)
  const nav = useNavigate()
  const { planId: planIdParam } = useParams()
  const planId = Number(planIdParam) || 0
  const { data } = usePlanDetail(childId, planId)

  const start = useStartPlan(childId)
  const answer = useAnswerItem(childId, planId)
  const finish = useFinishPlan(childId)
  const sound = useSound()
  const speech = useSpeech()

  // 本地维护题目状态，避免每答一题就重拉整份计划（会闪一下）。
  const [items, setItems] = useState<PlanItem[]>([])
  const [cursor, setCursor] = useState(0)
  const [phase, setPhase] = useState<Phase>('asking')
  const [picked, setPicked] = useState<number | null>(null)
  const [correctIndex, setCorrectIndex] = useState<number | null>(null)
  const [showExit, setShowExit] = useState(false)

  const shownAt = useRef<number>(Date.now())
  const timers = useRef<number[]>([])
  const startedRef = useRef(false)

  const clearTimers = () => {
    timers.current.forEach(window.clearTimeout)
    timers.current = []
  }
  const later = (fn: () => void, ms: number) => {
    timers.current.push(window.setTimeout(fn, ms))
  }

  // 载入计划，并从第一道未完成的题继续——中途退出再进来能接着做。
  useEffect(() => {
    if (!data) return
    setItems(data.items)
    const next = data.items.findIndex((i) => i.status === 'pending')
    if (next === -1) {
      nav(`/task/${planId}/done`, { replace: true })
      return
    }
    setCursor(next)
    if (!startedRef.current) {
      startedRef.current = true
      start.mutate(data.plan.id)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data])

  useEffect(() => clearTimers, [])

  const item = items[cursor]
  const total = items.length
  const doneCount = useMemo(() => items.filter((i) => i.status !== 'pending').length, [items])

  const readAloud = useCallback((it: PlanItem) => {
    const { text, lang } = it.question.speech
    if (!text) return
    if (lang === 'en-US') {
      // 英语题先用中文说指令，再用英文念单词，中间留停顿——
      // 混成一句 TTS 会把英文读成中文腔。
      speech.speakPair('听一听', text)
    } else {
      speech.speak(text, 'zh-CN')
    }
  }, [speech])

  // 进入一道题：自动读题，20 秒没动静再读一遍。
  useEffect(() => {
    if (!item || phase !== 'asking') return
    shownAt.current = Date.now()
    const t = window.setTimeout(() => readAloud(item), 350)
    const reread = window.setTimeout(() => readAloud(item), REREAD_MS)
    return () => {
      window.clearTimeout(t)
      window.clearTimeout(reread)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [item?.id, phase])

  const advance = useCallback(async () => {
    clearTimers()
    setPicked(null)
    setCorrectIndex(null)

    const nextIdx = items.findIndex((i, idx) => idx > cursor && i.status === 'pending')
    if (nextIdx !== -1) {
      setCursor(nextIdx)
      setPhase('asking')
      return
    }
    // 前面可能还有跳过留下的未完成题（正常流程不会有，兜个底）
    const anyPending = items.findIndex((i) => i.status === 'pending')
    if (anyPending !== -1 && anyPending !== cursor) {
      setCursor(anyPending)
      setPhase('asking')
      return
    }
    speech.stop()
    if (planId) await finish.mutateAsync(planId).catch(() => undefined)
    nav(`/task/${planId}/done`, { replace: true })
  }, [items, cursor, planId, finish, nav, speech])

  const pick = (index: number) => {
    if (!item || phase !== 'asking') return
    setPhase('submitting')
    setPicked(index)
    speech.stop()

    const costMs = Date.now() - shownAt.current
    answer.mutate(
      { itemId: item.id, optionIndex: index, costMs },
      {
        onSuccess: (res) => {
          setCorrectIndex(res.answer_index)
          setItems((prev) =>
            prev.map((i) => (i.id === item.id ? { ...i, status: res.status, tries: res.tries } : i)),
          )

          if (res.correct) {
            sound.playRight()
            setPhase('right')
            later(() => void advance(), Math.max(RIGHT_HOLD_MS, TAP_LOCK_MS))
            return
          }
          sound.playWrong()
          if (res.can_retry) {
            setPhase('retry')
            later(() => speech.speak('再想想', 'zh-CN'), 420)
            later(() => {
              setPicked(null)
              setPhase('asking')
            }, RETRY_HOLD_MS)
            return
          }
          setPhase('reveal')
          later(() => speech.speak('这个才对，下次记住啦', 'zh-CN'), 420)
          later(() => void advance(), REVEAL_HOLD_MS)
        },
        onError: () => {
          // 网络出错就退回可作答状态，孩子再点一次即可（后端幂等）。
          setPhase('asking')
          setPicked(null)
        },
      },
    )
  }

  if (!item) {
    return (
      <div className="flex h-full items-center justify-center">
        <span className="animate-bob text-7xl">🌸</span>
      </div>
    )
  }

  const optionState = (index: number): OptionState => {
    if (phase === 'right' && index === picked) return 'chosen-right'
    if ((phase === 'retry' || phase === 'reveal') && index === picked) return 'chosen-wrong'
    if (phase === 'reveal' && index === correctIndex) return 'reveal'
    if (phase === 'reveal') return 'dimmed'
    return 'idle'
  }

  const locked = phase !== 'asking'

  return (
    <div className="flex h-full flex-col px-6 py-4">
      <header className="flex shrink-0 items-center justify-between">
        <ProgressDots total={total} done={doneCount} current={cursor} />
        <div className="flex items-center gap-3">
          <span className="rounded-full bg-white/70 px-4 py-1 text-base font-semibold text-candy-mute">
            {item.subject_name}
          </span>
          <button
            type="button"
            onClick={() => setShowExit(true)}
            aria-label="退出"
            className="flex h-11 w-11 items-center justify-center rounded-full bg-white/70 text-xl text-candy-mute"
          >
            ✕
          </button>
        </div>
      </header>

      <main className="flex min-h-0 flex-1 flex-col items-center justify-center gap-5">
        <div className="flex min-h-[132px] items-center justify-center">
          <QuestionVisual visual={item.question.visual} />
        </div>

        <div className="flex items-center gap-4">
          <h2 className="text-center text-[44px] font-bold leading-tight text-candy-ink">
            {item.question.stem}
          </h2>
          <button
            type="button"
            onClick={() => {
              sound.unlock()
              speech.prime()
              readAloud(item)
            }}
            aria-label="再读一遍"
            className="flex h-16 w-16 shrink-0 items-center justify-center rounded-full bg-white text-3xl
                       transition-all active:translate-y-[3px]"
            style={{ boxShadow: '0 4px 0 rgba(61,50,48,0.15)' }}
          >
            🔊
          </button>
        </div>

        {phase === 'retry' && (
          <p className="animate-popIn text-2xl font-bold text-candy-wrong">再想想 🤔</p>
        )}
        {phase === 'reveal' && (
          <p className="animate-popIn text-2xl font-bold text-candy-mute">这个才对，下次记住啦</p>
        )}
      </main>

      <div className="grid shrink-0 grid-cols-2 gap-4 pb-2">
        {item.question.options.map((opt, i) => (
          <OptionButton
            key={i}
            option={opt}
            slot={i}
            state={optionState(i)}
            disabled={locked}
            onPick={() => pick(i)}
          />
        ))}
      </div>

      {showExit && (
        <ExitConfirm
          onStay={() => setShowExit(false)}
          onLeave={() => {
            speech.stop()
            nav('/', { replace: true })
          }}
        />
      )}
    </div>
  )
}

/** 退出必须二次确认：右上角的叉太容易被误碰，孩子的进度不能一按就丢。 */
function ExitConfirm({ onStay, onLeave }: { onStay: () => void; onLeave: () => void }) {
  return (
    <div className="fixed inset-0 z-20 flex items-center justify-center bg-candy-ink/40 px-8">
      <div className="animate-popIn rounded-[2rem] bg-candy-paper px-10 py-8 text-center shadow-pop">
        <p className="text-3xl font-bold text-candy-ink">要休息一下吗？</p>
        <p className="mt-2 text-lg text-candy-mute">做过的题都存好了，回来能接着做</p>
        <div className="mt-8 flex gap-4">
          <button
            type="button"
            onClick={onLeave}
            className="min-h-[88px] flex-1 rounded-kid bg-white px-8 text-2xl font-bold text-candy-mute
                       transition-all active:translate-y-[4px]"
            style={{ boxShadow: '0 5px 0 rgba(61,50,48,0.12)' }}
          >
            休息一下
          </button>
          <button
            type="button"
            onClick={onStay}
            className="min-h-[88px] flex-1 rounded-kid px-8 text-2xl font-bold text-white
                       transition-all active:translate-y-[4px]"
            style={{ backgroundColor: '#FF8A5B', boxShadow: '0 5px 0 #D9663C' }}
          >
            继续做题
          </button>
        </div>
      </div>
    </div>
  )
}
