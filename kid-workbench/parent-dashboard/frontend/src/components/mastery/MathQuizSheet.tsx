import { useMemo, useState, type MouseEvent } from 'react'
import type { MatrixModule, MatrixPoint } from '../../api/types'

type QuizType = 'calc' | 'story' | 'find' | 'name'

interface PreviewOption {
  key: string
  label: string
  shape?: string
  correct: boolean
  style?: 'number' | 'text' | 'shape'
}

interface PreviewQuestion {
  id: string
  type: QuizType
  title: string
  prompt: string
  stemText: string
  speechText: string
  options: PreviewOption[]
}

const CN_DIGITS = ['零', '一', '二', '三', '四', '五', '六', '七', '八', '九', '十']
const SHAPE_KEYS: Record<string, string> = {
  圆形: 'circle',
  正方形: 'square',
  长方形: 'rect',
  三角形: 'triangle',
  椭圆形: 'oval',
  梯形: 'trapezoid',
  菱形: 'rhombus',
  五角星: 'star',
}

function cnNumber(n: number): string {
  if (n < 0) return String(n)
  if (n < CN_DIGITS.length) return CN_DIGITS[n]
  if (n < 20) return `十${CN_DIGITS[n - 10]}`
  if (n === 20) return '二十'
  return String(n)
}

function shuffle<T>(items: T[]): T[] {
  const out = [...items]
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[out[i], out[j]] = [out[j], out[i]]
  }
  return out
}

function consecutiveWindow(n: number): number[] {
  const lo = 0
  const hi = 20
  const size = 4
  const maxStart = hi - size + 1
  const starts: number[] = []
  for (let s = n - size + 1; s <= n; s++) {
    if (s >= lo && s <= maxStart) starts.push(s)
  }
  if (starts.length === 0) return []
  const start = starts[Math.floor(Math.random() * starts.length)]
  return [start, start + 1, start + 2, start + 3]
}

function numberOptions(answer: number): PreviewOption[] | null {
  const window = consecutiveWindow(answer)
  if (window.length !== 4) return null
  return shuffle(
    window.map((v) => ({
      key: String(v),
      label: String(v),
      correct: v === answer,
    })),
  )
}

function pickLabels(correct: string, pool: string[]): PreviewOption[] | null {
  const rest = pool.filter((x) => x !== correct)
  if (rest.length < 3) return null
  const picked = shuffle(rest).slice(0, 3)
  return shuffle(
    [correct, ...picked].map((label) => ({
      key: label,
      label,
      correct: label === correct,
      style: 'text' as const,
    })),
  )
}

function parsePoint(p: MatrixPoint): { kind: 'add' | 'sub' | 'shape'; a: number; b: number } | null {
  const add = p.title.match(/^(\d+)\+(\d+)$/)
  if (add) return { kind: 'add', a: Number(add[1]), b: Number(add[2]) }
  const sub = p.title.match(/^(\d+)-(\d+)$/)
  if (sub) return { kind: 'sub', a: Number(sub[1]), b: Number(sub[2]) }
  if (SHAPE_KEYS[p.title]) return { kind: 'shape', a: 0, b: 0 }
  return null
}

function buildQuestions(module: MatrixModule): PreviewQuestion[] {
  if (module.code !== 'add10' && module.code !== 'sub10' && module.code !== 'shape') return []
  const out: PreviewQuestion[] = []
  if (module.code === 'shape') {
    const names = module.points.map((x) => x.title).filter((n) => SHAPE_KEYS[n])
    const uniq = [...new Set(names)]
    if (uniq.length < 4) return []
    for (const p of module.points) {
      if (!SHAPE_KEYS[p.title]) continue
      const nameOpts = pickLabels(p.title, uniq)
      const rest = uniq.filter((n) => n !== p.title)
      const picked = shuffle(rest).slice(0, 3)
      const findOpts = shuffle(
        [p.title, ...picked].map((n) => ({
          key: n,
          label: n,
          shape: SHAPE_KEYS[n],
          correct: n === p.title,
          style: 'shape' as const,
        })),
      )
      if (!nameOpts) continue
      out.push({
        id: `${p.id}:find`,
        type: 'find',
        title: '看名称选图形',
        prompt: `找出「${p.title}」`,
        stemText: p.title,
        speechText: p.title,
        options: findOpts,
      })
      out.push({
        id: `${p.id}:name`,
        type: 'name',
        title: '看图形选名称',
        prompt: '这是什么图形？',
        stemText: p.title,
        speechText: '这是什么图形',
        options: nameOpts,
      })
    }
    return out
  }
  for (const p of module.points.slice(0, 30)) {
    const parsed = parsePoint(p)
    if (!parsed || (parsed.kind !== 'add' && parsed.kind !== 'sub')) continue
    const result = parsed.kind === 'add' ? parsed.a + parsed.b : parsed.a - parsed.b
    if (result < 0 || result > 20) continue
    const sign = parsed.kind === 'add' ? '+' : '-'
    const verb = parsed.kind === 'add' ? '加' : '减'
    const story = parsed.kind === 'add' ? '一共有几个？' : '还剩几个？'
    const calcOpts = numberOptions(result)
    const storyOpts = numberOptions(result)
    if (!calcOpts || !storyOpts) continue
    out.push({
      id: `${p.id}:calc`,
      type: 'calc',
      title: '看算式选答案',
      prompt: `${parsed.a} ${sign} ${parsed.b} = ?`,
      stemText: `${parsed.a} ${sign} ${parsed.b}`,
      speechText: `${cnNumber(parsed.a)}${verb}${cnNumber(parsed.b)}等于几`,
      options: calcOpts.map((o) => ({ ...o, style: 'number' as const })),
    })
    out.push({
      id: `${p.id}:story`,
      type: 'story',
      title: '看图选答案',
      prompt: story,
      stemText: '',
      speechText: story,
      options: storyOpts.map((o) => ({ ...o, style: 'number' as const })),
    })
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

export function MathQuizSheet({
  module,
  onClose,
}: {
  module: MatrixModule
  onClose: () => void
}) {
  const questions = useMemo(() => buildQuestions(module), [module])
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
            {active ? ` · ${active.prompt} · ${active.title}` : ' · 题目列表'}
          </h2>
          {!active ? (
            <p className="mt-1 text-xs text-slate-400">
              {module.code === 'shape'
                ? '看名称选图形 / 看图形选名称 · 点击进入详情试答'
                : '看算式 / 看图 · 四个连续数字选项 · 预览前 30 题 · 点击进入详情试答'}
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
              <p className="text-sm text-slate-400">本组暂无法生成题目预览。</p>
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
  const stemShape = q.type === 'name' ? SHAPE_KEYS[q.stemText] : undefined
  return (
    <div
      className={`grid grid-cols-[minmax(140px,1.05fr)_1fr_1fr] gap-3 rounded-2xl border border-white/10 bg-white/5 p-3 ${
        compact ? '' : 'min-h-[320px] p-5'
      }`}
    >
      <div className="flex flex-col items-start gap-2">
        <span className="text-[11px] text-amber-200/80">简单 · {q.title}</span>
        <p className={`m-0 text-slate-100 ${compact ? 'text-sm' : 'text-base'}`}>{q.prompt}</p>
        {stemShape ? (
          <span
            className={`grid aspect-square place-items-center rounded-[10px] bg-white shadow-sm ${
              compact ? 'h-[88px] w-[88px]' : 'h-[140px] w-[140px]'
            }`}
          >
            <ShapeGlyph shape={stemShape} compact={compact} />
          </span>
        ) : q.stemText ? (
          <span
            className={`grid aspect-square place-items-center rounded-[10px] bg-white font-serif font-medium text-slate-900 shadow-sm whitespace-nowrap [writing-mode:horizontal-tb] ${
              compact ? 'h-[88px] w-[88px] text-2xl' : 'h-[140px] w-[140px] text-4xl'
            }`}
          >
            {q.stemText}
          </span>
        ) : null}
        <button
          type="button"
          className="rounded-full border border-teal-400/40 bg-teal-500/15 px-2 py-1 text-sm text-teal-200"
          onClick={(e) => speakText(q.speechText, e)}
        >
          🔊 读音
        </button>
      </div>
      <div className="flex flex-col gap-2">
        {left.map((opt) => (
          <OptionTile key={`L-${opt.key}`} opt={opt} compact={compact} />
        ))}
      </div>
      <div className="flex flex-col gap-2">
        {right.map((opt) => (
          <OptionTile key={`R-${opt.key}`} opt={opt} compact={compact} />
        ))}
      </div>
    </div>
  )
}

function OptionTile({
  opt,
  compact,
  interactive,
  state,
  onPick,
}: {
  opt: PreviewOption
  compact?: boolean
  interactive?: boolean
  state?: 'idle' | 'correct' | 'wrong'
  onPick?: () => void
}) {
  let ring = 'border-transparent'
  if (!interactive && opt.correct) ring = 'border-teal-300/70'
  if (state === 'correct') ring = 'border-teal-300'
  if (state === 'wrong') ring = 'border-rose-400 opacity-80'

  const cls = `rounded-xl border-2 bg-transparent p-0 ${ring}`

  const size = compact ? 'h-[88px] w-[88px]' : 'h-[140px] w-[140px]'
  const card =
    opt.style === 'shape' && opt.shape ? (
      <span className={`mx-auto grid aspect-square place-items-center rounded-[10px] bg-white p-2 shadow-sm ${size}`}>
        <ShapeGlyph shape={opt.shape} compact={compact} />
      </span>
    ) : (
      <span className={`mx-auto grid aspect-square place-items-center rounded-[10px] bg-white p-2 shadow-sm ${size}`}>
        <span
          className={`font-serif font-medium leading-none text-slate-900 whitespace-nowrap [writing-mode:horizontal-tb] ${
            compact ? 'text-2xl tracking-wide' : 'text-3xl tracking-wide'
          }`}
        >
          {opt.label}
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

function ShapeGlyph({ shape, compact }: { shape: string; compact?: boolean }) {
  const size = compact ? 48 : 72
  const common = { width: size, height: size, viewBox: '0 0 100 100' }
  switch (shape) {
    case 'circle':
      return (
        <svg {...common}>
          <circle cx="50" cy="50" r="36" fill="#3b82f6" />
        </svg>
      )
    case 'square':
      return (
        <svg {...common}>
          <rect x="18" y="18" width="64" height="64" fill="#22c55e" />
        </svg>
      )
    case 'rect':
      return (
        <svg {...common}>
          <rect x="10" y="28" width="80" height="44" fill="#eab308" />
        </svg>
      )
    case 'triangle':
      return (
        <svg {...common}>
          <polygon points="50,12 90,88 10,88" fill="#ef4444" />
        </svg>
      )
    case 'oval':
      return (
        <svg {...common}>
          <ellipse cx="50" cy="50" rx="40" ry="28" fill="#a855f7" />
        </svg>
      )
    case 'trapezoid':
      return (
        <svg {...common}>
          <polygon points="25,22 75,22 92,78 8,78" fill="#f97316" />
        </svg>
      )
    case 'rhombus':
      return (
        <svg {...common}>
          <polygon points="50,10 90,50 50,90 10,50" fill="#14b8a6" />
        </svg>
      )
    case 'star':
      return (
        <svg {...common}>
          <polygon
            points="50,8 61,38 94,38 67,58 78,90 50,70 22,90 33,58 6,38 39,38"
            fill="#f59e0b"
          />
        </svg>
      )
    default:
      return <span className="text-slate-800">{shape}</span>
  }
}

function QuestionDetail({ q }: { q: PreviewQuestion }) {
  const [selected, setSelected] = useState<string | null>(null)
  const [a, b, c, d] = q.options
  const left = [a, c].filter(Boolean)
  const right = [b, d].filter(Boolean)
  const stemShape = q.type === 'name' ? SHAPE_KEYS[q.stemText] : undefined

  return (
    <div className="mx-auto max-w-5xl">
      <div className="grid grid-cols-[minmax(140px,1.05fr)_1fr_1fr] gap-3 rounded-2xl border border-white/10 bg-white/5 p-5 min-h-[320px]">
        <div className="flex flex-col items-start gap-2">
          <span className="text-[11px] text-amber-200/80">简单 · {q.title}</span>
          <p className="m-0 text-base text-slate-100">{q.prompt}</p>
          {stemShape ? (
            <span className="grid aspect-square h-[140px] w-[140px] place-items-center rounded-[10px] bg-white shadow-sm">
              <ShapeGlyph shape={stemShape} />
            </span>
          ) : q.stemText ? (
            <span className="grid aspect-square h-[140px] w-[140px] place-items-center rounded-[10px] bg-white font-serif text-4xl font-medium text-slate-900 shadow-sm whitespace-nowrap [writing-mode:horizontal-tb]">
              {q.stemText}
            </span>
          ) : null}
          <button
            type="button"
            className="rounded-full border border-teal-400/40 bg-teal-500/15 px-2 py-1 text-sm text-teal-200"
            onClick={(e) => speakText(q.speechText, e)}
          >
            🔊 读音
          </button>
        </div>
        <div className="flex flex-col gap-2">
          {left.map((opt) => (
            <OptionTile
              key={`L-${opt.key}`}
              opt={opt}
              interactive
              state={
                selected == null ? 'idle' : selected === opt.key ? (opt.correct ? 'correct' : 'wrong') : 'idle'
              }
              onPick={() => setSelected(opt.key)}
            />
          ))}
        </div>
        <div className="flex flex-col gap-2">
          {right.map((opt) => (
            <OptionTile
              key={`R-${opt.key}`}
              opt={opt}
              interactive
              state={
                selected == null ? 'idle' : selected === opt.key ? (opt.correct ? 'correct' : 'wrong') : 'idle'
              }
              onPick={() => setSelected(opt.key)}
            />
          ))}
        </div>
      </div>
    </div>
  )
}
