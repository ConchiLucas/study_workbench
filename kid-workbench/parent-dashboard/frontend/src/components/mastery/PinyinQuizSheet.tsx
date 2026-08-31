import { useMemo, useState, type MouseEvent } from 'react'
import type { MatrixModule, MatrixPoint } from '../../api/types'

type SkillCode = 'inword' | 'listen'

interface PreviewQuestion {
  id: string
  skill: SkillCode
  title: string
  letter: string
  stemKind: 'word' | 'speech'
  prompt: string
  stemText: string
  speechText: string
  options: { letter: string; correct: boolean }[]
}

const SKILL_META: Record<
  SkillCode,
  { title: string; stemKind: PreviewQuestion['stemKind']; prompt: string }
> = {
  inword: {
    title: '听例字选音',
    stemKind: 'word',
    prompt: '听一听，这个字里有哪个音？',
  },
  listen: {
    title: '听单读选字母',
    stemKind: 'speech',
    prompt: '听一听，这是哪个字母？',
  },
}

/** 与 backend quiz/pinyin.go 的 pinyinTable 对齐（Solo / Word）。 */
const PINYIN_READING: Record<string, { solo: string; word: string }> = {
  b: { solo: '波', word: '爸' },
  p: { solo: '坡', word: '怕' },
  m: { solo: '摸', word: '妈' },
  f: { solo: '佛', word: '飞' },
  d: { solo: '得', word: '大' },
  t: { solo: '特', word: '题' },
  n: { solo: '呢', word: '你' },
  l: { solo: '勒', word: '来' },
  g: { solo: '哥', word: '高' },
  k: { solo: '科', word: '看' },
  h: { solo: '喝', word: '好' },
  j: { solo: '基', word: '家' },
  q: { solo: '欺', word: '去' },
  x: { solo: '希', word: '小' },
  zh: { solo: '知', word: '只' },
  ch: { solo: '吃', word: '车' },
  sh: { solo: '诗', word: '书' },
  r: { solo: '日', word: '肉' },
  z: { solo: '资', word: '走' },
  c: { solo: '次', word: '菜' },
  s: { solo: '思', word: '四' },
  y: { solo: '衣', word: '羊' },
  w: { solo: '乌', word: '我' },
  a: { solo: '啊', word: '妈' },
  o: { solo: '喔', word: '我' },
  e: { solo: '鹅', word: '鹅' },
  i: { solo: '衣', word: '米' },
  u: { solo: '乌', word: '鼓' },
  ü: { solo: '鱼', word: '鱼' },
  ai: { solo: '哀', word: '白' },
  ei: { solo: '诶', word: '飞' },
  ui: { solo: '威', word: '水' },
  ao: { solo: '熬', word: '猫' },
  ou: { solo: '欧', word: '头' },
  iu: { solo: '优', word: '牛' },
  ie: { solo: '耶', word: '叶' },
  üe: { solo: '约', word: '月' },
  er: { solo: '儿', word: '儿' },
  an: { solo: '安', word: '山' },
  en: { solo: '恩', word: '门' },
  in: { solo: '因', word: '心' },
  un: { solo: '温', word: '春' },
  ün: { solo: '晕', word: '云' },
  ang: { solo: '昂', word: '羊' },
  eng: { solo: '', word: '灯' },
}

const CONFUSION: string[][] = [
  ['b', 'd', 'p', 'q'],
  ['m', 'n', 'f', 'h'],
  ['g', 'k', 'h', 'j'],
  ['j', 'q', 'x', 'y'],
  ['zh', 'ch', 'sh', 'r'],
  ['z', 'c', 's', 'zh'],
  ['t', 'd', 'l', 'n'],
  ['y', 'w', 'm', 'n'],
  ['a', 'o', 'e', 'i'],
  ['i', 'u', 'ü', 'e'],
  ['ai', 'ei', 'ui', 'ao'],
  ['ao', 'ou', 'iu', 'ai'],
  ['ie', 'üe', 'er', 'iu'],
  ['an', 'en', 'in', 'un'],
  ['un', 'ün', 'in', 'en'],
  ['ang', 'eng', 'an', 'en'],
]

function shuffle<T>(items: T[]): T[] {
  const out = [...items]
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[out[i], out[j]] = [out[j], out[i]]
  }
  return out
}

function speakText(text: string, e?: MouseEvent) {
  e?.stopPropagation()
  if (!text || typeof window === 'undefined' || !window.speechSynthesis) return
  const u = new SpeechSynthesisUtterance(text)
  u.lang = 'zh-CN'
  window.speechSynthesis.cancel()
  window.speechSynthesis.speak(u)
}

function pickDistractors(letter: string, pool: string[]): string[] {
  const seen = new Set([letter])
  const confusing: string[] = []
  for (const group of CONFUSION) {
    if (!group.includes(letter)) continue
    for (const v of group) {
      if (!seen.has(v) && pool.includes(v)) {
        seen.add(v)
        confusing.push(v)
      }
    }
  }
  const rest = pool.filter((p) => !seen.has(p))
  return shuffle([...confusing, ...rest]).slice(0, 3)
}

function buildQuestions(points: MatrixPoint[]): PreviewQuestion[] {
  const letters = points.map((p) => p.title)
  const out: PreviewQuestion[] = []
  for (const p of points) {
    const reading = PINYIN_READING[p.title]
    if (!reading) continue
    const distractors = pickDistractors(p.title, letters.filter((l) => l !== p.title))
    if (distractors.length < 3) continue

    if (reading.word) {
      const meta = SKILL_META.inword
      out.push({
        id: `${p.id}:inword`,
        skill: 'inword',
        title: meta.title,
        letter: p.title,
        stemKind: meta.stemKind,
        prompt: meta.prompt,
        stemText: reading.word,
        speechText: reading.word,
        options: shuffle([
          { letter: p.title, correct: true },
          ...distractors.map((l) => ({ letter: l, correct: false })),
        ]),
      })
    }
    if (reading.solo) {
      const meta = SKILL_META.listen
      out.push({
        id: `${p.id}:listen`,
        skill: 'listen',
        title: meta.title,
        letter: p.title,
        stemKind: meta.stemKind,
        prompt: meta.prompt,
        stemText: '',
        speechText: reading.solo,
        options: shuffle([
          { letter: p.title, correct: true },
          ...distractors.map((l) => ({ letter: l, correct: false })),
        ]),
      })
    }
  }
  return out
}

export function PinyinQuizSheet({
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
            {active ? ` · ${active.letter} · ${active.title}` : ' · 题目列表'}
          </h2>
          {!active ? (
            <p className="mt-1 text-xs text-slate-400">
              两类题：听例字选音 / 听单读选字母 · 四选一 · 点击进入详情试答
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
              <p className="text-sm text-slate-400">本组字母不足，暂无法生成四选一题目。</p>
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
  return (
    <div
      className={`grid grid-cols-[minmax(140px,1.05fr)_1fr_1fr] gap-3 rounded-2xl border border-white/10 bg-white/5 p-3 ${
        compact ? '' : 'min-h-[320px] p-5'
      }`}
    >
      <div className="flex flex-col items-start gap-2">
        <span className="text-[11px] text-amber-200/80">简单 · {q.title}</span>
        <p className={`m-0 text-slate-100 ${compact ? 'text-sm' : 'text-base'}`}>{q.prompt}</p>
        <div className="flex items-end gap-2">
          {q.stemKind === 'word' ? (
            <span
              className={`grid place-items-center rounded-xl bg-white text-slate-900 ${
                compact ? 'h-16 w-16 text-3xl' : 'h-28 w-28 text-5xl'
              }`}
            >
              {q.stemText}
            </span>
          ) : null}
          <button
            type="button"
            className="rounded-full border border-teal-400/40 bg-teal-500/15 px-2 py-1 text-sm text-teal-200"
            title="读音"
            onClick={(e) => speakText(q.speechText, e)}
          >
            🔊 读音
          </button>
        </div>
      </div>
      <div className="flex flex-col gap-2">
        {left.map((opt) => (
          <OptionTile key={`L-${opt.letter}`} letter={opt.letter} correct={opt.correct} compact={compact} />
        ))}
      </div>
      <div className="flex flex-col gap-2">
        {right.map((opt) => (
          <OptionTile key={`R-${opt.letter}`} letter={opt.letter} correct={opt.correct} compact={compact} />
        ))}
      </div>
    </div>
  )
}

function OptionTile({
  letter,
  correct,
  compact,
  interactive,
  state,
  onPick,
}: {
  letter: string
  correct?: boolean
  compact?: boolean
  interactive?: boolean
  state?: 'idle' | 'correct' | 'wrong'
  onPick?: () => void
}) {
  let ring = 'border-transparent'
  if (!interactive && correct) ring = 'border-teal-300/70'
  if (state === 'correct') ring = 'border-teal-300'
  if (state === 'wrong') ring = 'border-rose-400 opacity-80'

  const cls = `flex-1 rounded-xl border-2 bg-transparent p-0 ${ring} ${
    compact ? 'min-h-[72px]' : 'min-h-[120px]'
  }`

  const card = (
    <span
      className={`mx-auto block aspect-square w-full max-w-[140px] rounded-[10px] bg-white p-[8%] shadow-sm ${
        compact ? 'max-w-[88px]' : ''
      }`}
    >
      <span
        className="relative grid h-full w-full place-items-center"
        style={{
          border: '1px solid #c4c4c4',
          backgroundImage:
            'linear-gradient(#c4c4c4,#c4c4c4),linear-gradient(#c4c4c4,#c4c4c4)',
          backgroundSize: '1px 100%,100% 1px',
          backgroundPosition: 'center,center',
          backgroundRepeat: 'no-repeat',
        }}
      >
        <span
          className={`relative z-[1] font-serif font-medium leading-none text-slate-900 ${
            compact ? 'text-3xl' : 'text-5xl'
          }`}
        >
          {letter}
        </span>
      </span>
    </span>
  )

  if (!interactive) {
    return <div className={cls}>{card}</div>
  }
  return (
    <button
      type="button"
      className={cls}
      onClick={onPick}
      disabled={state !== 'idle' && state !== undefined}
    >
      {card}
    </button>
  )
}

function QuestionDetail({ q }: { q: PreviewQuestion }) {
  const [selected, setSelected] = useState<string | null>(null)
  const [answer, setAnswer] = useState<'idle' | 'correct' | 'wrong'>('idle')
  const [a, b, c, d] = q.options
  const left = [a, c].filter(Boolean)
  const right = [b, d].filter(Boolean)

  function pick(letter: string, correct: boolean) {
    if (answer !== 'idle') return
    setSelected(letter)
    setAnswer(correct ? 'correct' : 'wrong')
  }

  function tileState(letter: string, correct: boolean): 'idle' | 'correct' | 'wrong' {
    if (answer === 'idle') return 'idle'
    if (correct) return 'correct'
    if (selected === letter) return 'wrong'
    return 'idle'
  }

  return (
    <div className="mx-auto flex max-w-4xl flex-col gap-4">
      <div className="grid min-h-[360px] grid-cols-[minmax(160px,1.05fr)_1fr_1fr] gap-4 rounded-2xl border border-white/10 bg-white/5 p-5">
        <div className="flex flex-col items-start gap-3">
          <span className="text-xs text-amber-200/80">简单 · {q.title}</span>
          <p className="m-0 text-lg text-slate-100">{q.prompt}</p>
          <div className="flex items-end gap-2">
            {q.stemKind === 'word' ? (
              <span className="grid h-32 w-32 place-items-center rounded-2xl bg-white text-6xl text-slate-900">
                {q.stemText}
              </span>
            ) : null}
            <button
              type="button"
              className="rounded-full border border-teal-400/40 bg-teal-500/15 px-3 py-2 text-teal-200"
              onClick={(e) => speakText(q.speechText, e)}
            >
              🔊 读音
            </button>
          </div>
        </div>
        <div className="flex flex-col gap-3">
          {left.map((opt) => (
            <OptionTile
              key={`dL-${opt.letter}`}
              letter={opt.letter}
              interactive
              state={tileState(opt.letter, opt.correct)}
              onPick={() => pick(opt.letter, opt.correct)}
            />
          ))}
        </div>
        <div className="flex flex-col gap-3">
          {right.map((opt) => (
            <OptionTile
              key={`dR-${opt.letter}`}
              letter={opt.letter}
              interactive
              state={tileState(opt.letter, opt.correct)}
              onPick={() => pick(opt.letter, opt.correct)}
            />
          ))}
        </div>
      </div>
      {answer !== 'idle' ? (
        <div className="flex items-center justify-between gap-3 rounded-xl border border-white/10 bg-white/5 px-4 py-3">
          <p className={answer === 'correct' ? 'text-teal-300' : 'text-rose-300'}>
            {answer === 'correct' ? '答对了！' : `再想想～ 正确答案是「${q.letter}」`}
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
