import { useMemo, useState, type MouseEvent } from 'react'
import type { MatrixModule, MatrixPoint } from '../../api/types'

type SkillCode = 'glyph_sense' | 'sense_char'

interface PreviewQuestion {
  id: string
  skill: SkillCode
  title: string
  difficulty: 'medium' | 'easy'
  char: string
  stemKind: 'glyph' | 'sense'
  prompt: string
  options: { char: string; correct: boolean }[]
}

const SKILL_META: Record<
  SkillCode,
  { title: string; difficulty: 'medium' | 'easy'; stemKind: PreviewQuestion['stemKind']; prompt: string }
> = {
  glyph_sense: {
    title: '看字选义',
    difficulty: 'easy',
    stemKind: 'glyph',
    prompt: '看字图，选出义图（暂用字卡）',
  },
  sense_char: {
    title: '看义选字',
    difficulty: 'easy',
    stemKind: 'sense',
    prompt: '看义图，选出字（暂用字卡）',
  },
}

const SKILL_ORDER: SkillCode[] = ['glyph_sense', 'sense_char']

function shuffle<T>(items: T[]): T[] {
  const out = [...items]
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[out[i], out[j]] = [out[j], out[i]]
  }
  return out
}

function speakChar(char: string, e?: MouseEvent) {
  e?.stopPropagation()
  if (typeof window === 'undefined' || !window.speechSynthesis) return
  const u = new SpeechSynthesisUtterance(char)
  u.lang = 'zh-CN'
  window.speechSynthesis.cancel()
  window.speechSynthesis.speak(u)
}

function buildQuestions(points: MatrixPoint[]): PreviewQuestion[] {
  const chars = points.map((p) => p.title)
  const out: PreviewQuestion[] = []
  for (const p of points) {
    for (const skill of SKILL_ORDER) {
      const meta = SKILL_META[skill]
      const pool = chars.filter((c) => c !== p.title)
      if (pool.length < 3) continue
      const distractors = shuffle(pool).slice(0, 3)
      const options = shuffle([
        { char: p.title, correct: true },
        ...distractors.map((c) => ({ char: c, correct: false })),
      ])
      out.push({
        id: `${p.id}:${skill}`,
        skill,
        title: meta.title,
        difficulty: meta.difficulty,
        char: p.title,
        stemKind: meta.stemKind,
        prompt: meta.prompt,
        options,
      })
    }
  }
  return out
}

export function LiteracyQuizSheet({
  module,
  onClose,
}: {
  module: MatrixModule
  onClose: () => void
}) {
  const questions = useMemo(() => buildQuestions(module.points), [module.points])
  const [activeId, setActiveId] = useState<string | null>(null)
  const active = questions.find((q) => q.id === activeId) ?? null

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-[#071012] text-slate-100">
      <header className="flex items-start justify-between gap-4 border-b border-white/10 px-5 py-4">
        <div>
          {active ? (
            <button type="button" className="mb-2 text-sm text-teal-300" onClick={() => setActiveId(null)}>
              ← 返回列表
            </button>
          ) : null}
          <h2 className="text-lg font-semibold tracking-wide">
            {module.name}
            {active ? ` · ${active.char} · ${active.title}` : ' · 题目列表'}
          </h2>
          {!active ? (
            <p className="mt-1 text-xs text-slate-400">
              两类题：看字选义 / 看义选字 · 左题干 · 右两列选项 · 点击进入详情试答
            </p>
          ) : null}
        </div>
        <button
          type="button"
          className="rounded-lg border border-white/15 px-3 py-1.5 text-sm text-slate-300 hover:border-teal-400/50 hover:text-teal-200"
          onClick={onClose}
        >
          关闭
        </button>
      </header>

      <div className="flex-1 overflow-auto px-5 py-4">
        {active ? (
          <QuestionDetail q={active} />
        ) : (
          <div className="mx-auto flex max-w-5xl flex-col gap-3">
            {questions.map((q) => (
              <div
                key={q.id}
                className="w-full cursor-pointer text-left"
                role="button"
                tabIndex={0}
                onClick={() => setActiveId(q.id)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault()
                    setActiveId(q.id)
                  }
                }}
              >
                <QuestionRow q={q} compact />
              </div>
            ))}
            {questions.length === 0 ? (
              <p className="text-sm text-slate-400">本组字数不足，暂无法生成四选一题目。</p>
            ) : null}
          </div>
        )}
      </div>
    </div>
  )
}

function QuestionRow({ q, compact }: { q: PreviewQuestion; compact?: boolean }) {
  const [a, b, c, d] = q.options
  const left = [a, c].filter(Boolean)
  const right = [b, d].filter(Boolean)
  const optionSpeech = q.skill === 'glyph_sense'
  return (
    <div
      className={`grid grid-cols-[minmax(140px,1.05fr)_1fr_1fr] gap-3 rounded-2xl border border-white/10 bg-white/5 p-3 ${
        compact ? '' : 'min-h-[320px] p-5'
      }`}
    >
      <div className="flex flex-col items-start gap-2">
        <span className="text-[11px] text-amber-200/80">
          {q.difficulty === 'medium' ? '中等' : '简单'} · {q.title}
        </span>
        <p className={`m-0 text-slate-100 ${compact ? 'text-sm' : 'text-base'}`}>{q.prompt}</p>
        <div className="flex items-end gap-2">
          {q.stemKind === 'glyph' ? (
            <span
              className={`grid place-items-center rounded-xl bg-white text-slate-900 ${
                compact ? 'h-16 w-16 text-3xl' : 'h-28 w-28 text-5xl'
              }`}
            >
              {q.char}
            </span>
          ) : (
            <>
              <span
                className={`grid place-items-center rounded-xl bg-white text-slate-900 ${
                  compact ? 'h-16 w-16 text-3xl' : 'h-28 w-28 text-5xl'
                }`}
              >
                {q.char}
              </span>
              <button
                type="button"
                className="rounded-full border border-teal-400/40 bg-teal-500/15 px-2 py-1 text-sm text-teal-200"
                title={`读「${q.char}」`}
                onClick={(e) => speakChar(q.char, e)}
              >
                🔊
              </button>
            </>
          )}
        </div>
      </div>
      <div className="flex flex-col gap-2">
        {left.map((opt) => (
          <OptionTile
            key={`L-${opt.char}`}
            char={opt.char}
            correct={opt.correct}
            compact={compact}
            showSpeech={optionSpeech}
          />
        ))}
      </div>
      <div className="flex flex-col gap-2">
        {right.map((opt) => (
          <OptionTile
            key={`R-${opt.char}`}
            char={opt.char}
            correct={opt.correct}
            compact={compact}
            showSpeech={optionSpeech}
          />
        ))}
      </div>
    </div>
  )
}

function OptionTile({
  char,
  correct,
  compact,
  interactive,
  state,
  onPick,
  showSpeech,
}: {
  char: string
  correct?: boolean
  compact?: boolean
  interactive?: boolean
  state?: 'idle' | 'correct' | 'wrong'
  onPick?: () => void
  showSpeech?: boolean
}) {
  let ring = 'border-white/15'
  if (!interactive && correct) ring = 'border-teal-300/50'
  if (state === 'correct') ring = 'border-teal-300'
  if (state === 'wrong') ring = 'border-rose-400 opacity-80'

  const cls = `relative flex-1 grid place-items-center rounded-xl border-2 bg-[#0a1214] ${ring} ${
    compact ? 'min-h-[72px] text-2xl' : 'min-h-[120px] text-4xl'
  }`

  const speech = showSpeech ? (
    <button
      type="button"
      className="absolute bottom-1.5 right-1.5 rounded-full border border-teal-400/40 bg-black/70 px-1.5 py-0.5 text-xs text-teal-200"
      title={`读「${char}」`}
      onClick={(e) => speakChar(char, e)}
    >
      🔊
    </button>
  ) : null

  if (!interactive) {
    return (
      <div className={cls}>
        {char}
        {speech}
      </div>
    )
  }
  if (showSpeech) {
    return (
      <div
        className={`${cls} ${state === 'idle' || state === undefined ? 'cursor-pointer' : ''}`}
        role="button"
        tabIndex={state === 'idle' || state === undefined ? 0 : -1}
        onClick={() => {
          if (state === 'idle' || state === undefined) onPick?.()
        }}
        onKeyDown={(e) => {
          if (state !== 'idle' && state !== undefined) return
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            onPick?.()
          }
        }}
      >
        {char}
        {speech}
      </div>
    )
  }
  return (
    <button
      type="button"
      className={cls}
      onClick={onPick}
      disabled={state !== 'idle' && state !== undefined}
    >
      {char}
    </button>
  )
}

function QuestionDetail({ q }: { q: PreviewQuestion }) {
  const [selected, setSelected] = useState<string | null>(null)
  const [answer, setAnswer] = useState<'idle' | 'correct' | 'wrong'>('idle')
  const [a, b, c, d] = q.options
  const left = [a, c].filter(Boolean)
  const right = [b, d].filter(Boolean)
  const optionSpeech = q.skill === 'glyph_sense'

  function pick(char: string, correct: boolean) {
    if (answer !== 'idle') return
    setSelected(char)
    setAnswer(correct ? 'correct' : 'wrong')
  }

  function tileState(char: string, correct: boolean): 'idle' | 'correct' | 'wrong' {
    if (answer === 'idle') return 'idle'
    if (correct) return 'correct'
    if (selected === char) return 'wrong'
    return 'idle'
  }

  return (
    <div className="mx-auto flex max-w-4xl flex-col gap-4">
      <div className="grid min-h-[360px] grid-cols-[minmax(160px,1.05fr)_1fr_1fr] gap-4 rounded-2xl border border-white/10 bg-white/5 p-5">
        <div className="flex flex-col items-start gap-3">
          <span className="text-xs text-amber-200/80">
            {q.difficulty === 'medium' ? '中等' : '简单'} · {q.title}
          </span>
          <p className="m-0 text-lg text-slate-100">{q.prompt}</p>
          <div className="flex items-end gap-2">
            <span className="grid h-32 w-32 place-items-center rounded-2xl bg-white text-6xl text-slate-900">
              {q.char}
            </span>
            {q.skill === 'sense_char' ? (
              <button
                type="button"
                className="rounded-full border border-teal-400/40 bg-teal-500/15 px-3 py-2 text-teal-200"
                title={`读「${q.char}」`}
                onClick={(e) => speakChar(q.char, e)}
              >
                🔊
              </button>
            ) : null}
          </div>
        </div>
        <div className="flex flex-col gap-3">
          {left.map((opt) => (
            <OptionTile
              key={`dL-${opt.char}`}
              char={opt.char}
              interactive
              state={tileState(opt.char, opt.correct)}
              onPick={() => pick(opt.char, opt.correct)}
              showSpeech={optionSpeech}
            />
          ))}
        </div>
        <div className="flex flex-col gap-3">
          {right.map((opt) => (
            <OptionTile
              key={`dR-${opt.char}`}
              char={opt.char}
              interactive
              state={tileState(opt.char, opt.correct)}
              onPick={() => pick(opt.char, opt.correct)}
              showSpeech={optionSpeech}
            />
          ))}
        </div>
      </div>
      {answer !== 'idle' ? (
        <div className="flex items-center justify-between gap-3 rounded-xl border border-white/10 bg-white/5 px-4 py-3">
          <p className={answer === 'correct' ? 'text-teal-300' : 'text-rose-300'}>
            {answer === 'correct' ? '答对了！' : `再想想～ 正确答案是「${q.char}」`}
          </p>
          <button
            type="button"
            className="rounded-lg border border-teal-400/40 bg-teal-500/20 px-3 py-1.5 text-sm text-teal-100"
            onClick={() => {
              setSelected(null)
              setAnswer('idle')
            }}
          >
            再试一次
          </button>
        </div>
      ) : (
        <p className="text-center text-xs text-slate-500">点选项试答 · 左题干、右两列上下选项</p>
      )}
    </div>
  )
}
