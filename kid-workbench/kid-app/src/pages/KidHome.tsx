import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { useExtraPlan, useGenerateToday, useKidTodo, usePlanHistory } from '../api/plans'
import { useOverview } from '../api/dashboard'
import { localDate } from '../lib/date'
import { useChildStore } from '../store/childStore'
import { KidTaskCard } from '../components/KidTaskCard'
import { StarRow } from '../components/StarRow'
import { useSound } from '../hooks/useSound'
import { useSpeech } from '../hooks/useSpeech'

function relativeDayLabel(date: string, today: string): string {
  const d = new Date(`${date}T12:00:00`)
  const now = new Date(`${today}T12:00:00`)
  const diff = Math.round((now.getTime() - d.getTime()) / 86_400_000)
  if (diff === 1) return '昨天'
  if (diff === 2) return '前天'
  return `${date.slice(5).replace('-', '月')}日`
}

export function KidHome() {
  const childId = useChildStore((s) => s.childId)
  const nav = useNavigate()
  const today = localDate()
  const { data: todo, isLoading, error } = useKidTodo(childId)
  const { data: history } = usePlanHistory(childId)
  const { data: ov } = useOverview(childId)
  const generate = useGenerateToday(childId)
  const extra = useExtraPlan(childId)
  const sound = useSound()
  const speech = useSpeech()

  const todayPlans = useMemo(
    () => history?.filter((p) => p.plan_date === today) ?? [],
    [history, today],
  )
  const todayTodo = useMemo(
    () => (todo ?? []).filter((p) => p.plan_date === today).sort((a, b) => a.seq_no - b.seq_no),
    [todo, today],
  )
  const catchup = useMemo(
    () => (todo ?? []).filter((p) => p.plan_date !== today).slice(0, 2),
    [todo, today],
  )

  const todayPrimary = todayTodo[0]
  const todayAllDone = todayPlans.length > 0 && todayPlans.every((p) => p.status === 'done')
  const todayMainDone = todayPlans.find((p) => p.seq_no === 1 && p.status === 'done')

  const enter = (planId: number) => {
    sound.unlock()
    speech.prime()
    nav(`/task/${planId}`)
  }

  const startToday = () => {
    sound.unlock()
    speech.prime()
    generate.mutate(undefined, {
      onSuccess: (data) => enter(data.plan.id),
    })
  }

  const orderMore = () => {
    sound.unlock()
    speech.prime()
    extra.mutate(undefined, {
      onSuccess: (data) => enter(data.plan.id),
    })
  }

  if (isLoading) {
    return <Centered><span className="animate-bob text-7xl">🌸</span></Centered>
  }
  if (error) {
    return (
      <Centered>
        <p className="text-2xl font-bold text-candy-ink">今天的练习还没准备好</p>
        <p className="mt-2 text-base text-candy-mute">请让爸爸妈妈看一下</p>
      </Centered>
    )
  }

  return (
    <Centered>
      <div className="animate-floatUp w-full max-w-lg space-y-6">
        <h1 className="text-center text-4xl font-bold text-candy-ink">
          {ov?.child.name ?? '小朋友'}，今天练一练
        </h1>

        {todayPrimary ? (
          <KidTaskCard
            plan={todayPrimary}
            variant="primary"
            title={todayPrimary.seq_no === 1 ? '今天的练习' : `加餐 ${todayPrimary.seq_no - 1}`}
            subtitle={
              todayPrimary.done_count > 0
                ? `还剩 ${todayPrimary.target_count - todayPrimary.done_count} 道题`
                : `一共 ${todayPrimary.target_count} 道题，大概 8 分钟`
            }
            actionLabel={todayPrimary.done_count > 0 ? '接着做 →' : '开始 →'}
            onAction={() => enter(todayPrimary.id)}
          />
        ) : todayAllDone ? (
          <div className="rounded-[2rem] bg-candy-paper px-8 py-7 text-center shadow-pop">
            <StarRow stars={todayMainDone?.stars ?? todayPlans[todayPlans.length - 1]?.stars ?? 1} size={56} />
            <p className="mt-4 text-2xl font-bold text-candy-ink">今天完成啦！</p>
            <p className="mt-1 text-base text-candy-mute">
              做对 {todayMainDone?.correct_count ?? todayPlans[0]?.correct_count ?? 0} 道
            </p>
            <button
              type="button"
              onClick={orderMore}
              disabled={extra.isPending}
              className="mt-6 min-h-[88px] w-full rounded-kid bg-candy-mint px-10 text-3xl font-bold text-candy-ink
                         shadow-sticker transition-all active:translate-y-[5px] active:shadow-sticker-sm
                         disabled:opacity-50"
              style={{ boxShadow: '0 6px 0 #6FBE9B' }}
            >
              {extra.isPending ? '准备中…' : '再来一组 🍬'}
            </button>
            {extra.isError && (
              <p className="mt-3 text-base text-candy-mute">今天已经练得很多啦，明天再来 🌙</p>
            )}
          </div>
        ) : (
          <div className="rounded-[2rem] bg-candy-paper px-8 py-7 text-center shadow-pop">
            <p className="text-xl text-candy-mute">今天还没开始练习</p>
            <button
              type="button"
              onClick={startToday}
              disabled={generate.isPending}
              className="mt-6 min-h-[96px] w-full rounded-kid px-10 text-3xl font-bold text-white
                         transition-all active:translate-y-[6px] disabled:opacity-50"
              style={{ backgroundColor: '#FF8A5B', boxShadow: '0 7px 0 #D9663C' }}
            >
              {generate.isPending ? '准备中…' : '开始今天的练习 →'}
            </button>
            {generate.isError && (
              <p className="mt-3 text-base text-candy-mute">等一下再试试</p>
            )}
          </div>
        )}

        {catchup.length > 0 && (
          <div className="space-y-3">
            <p className="text-center text-lg font-semibold text-candy-mute">补做</p>
            {catchup.map((p) => (
              <KidTaskCard
                key={p.id}
                plan={p}
                variant="catchup"
                title={relativeDayLabel(p.plan_date, today)}
                subtitle={`还剩 ${p.target_count - p.done_count} 道`}
                actionLabel="接着做 →"
                onAction={() => enter(p.id)}
              />
            ))}
          </div>
        )}

        <div className="flex items-center justify-center gap-2 text-lg text-candy-mute">
          <span className="text-2xl">🌸</span>
          <span>我有 {ov?.child.flowers ?? 0} 朵小红花</span>
        </div>
      </div>
    </Centered>
  )
}

function Centered({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-full w-full items-center justify-center px-6">
      <div className="w-full text-center">{children}</div>
    </div>
  )
}
