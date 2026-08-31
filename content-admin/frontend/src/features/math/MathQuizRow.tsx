import { useState, type ReactNode } from 'react'
import { speechAudioURL } from '../../api/math'
import { SHAPE_KEYS, type MathQuizOption, type MathQuizQuestion } from './mathQuizPreview'

async function playSpeech(kpId: number, speechUrl?: string) {
  const res = await fetch(speechAudioURL(kpId, speechUrl))
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(typeof body.error === 'string' ? body.error : `读音失败 ${res.status}`)
  }
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const audio = new Audio(url)
  audio.onended = () => URL.revokeObjectURL(url)
  await audio.play()
}

export function MathQuizRow({
  question,
  detail,
}: {
  question: MathQuizQuestion
  detail?: boolean
}) {
  const [selected, setSelected] = useState<string | null>(null)
  const [speaking, setSpeaking] = useState(false)
  const [error, setError] = useState('')

  const onSpeak = async () => {
    setError('')
    setSpeaking(true)
    try {
      await playSpeech(question.target.kpId, question.speechUrl)
    } catch (e) {
      setError(e instanceof Error ? e.message : '读音失败')
    } finally {
      setSpeaking(false)
    }
  }

  const [a, b, c, d] = question.options
  const left = [a, c].filter(Boolean)
  const right = [b, d].filter(Boolean)

  return (
    <div className={`quiz-row ${detail ? 'quiz-row-detail' : ''}`}>
      <div className="quiz-stem">
        <span className="quiz-type-tag">简单 · {question.title}</span>
        <p className="quiz-prompt">{question.prompt}</p>
        <StemPreview question={question} />
        <button type="button" className="mini-btn" disabled={speaking} onClick={() => void onSpeak()}>
          {speaking ? '播放中…' : '🔊 读音'}
        </button>
        {error ? <p className="quiz-error">{error}</p> : null}
        {!question.available ? (
          <p className="muted">{question.unavailableReason ?? '暂不可用'}</p>
        ) : null}
      </div>
      <div className="quiz-options-col">
        {left.map((opt) => (
          <OptionTile
            key={`L-${opt.key}`}
            opt={opt}
            interactive={Boolean(detail)}
            selected={selected}
            onPick={() => setSelected(opt.key)}
          />
        ))}
      </div>
      <div className="quiz-options-col">
        {right.map((opt) => (
          <OptionTile
            key={`R-${opt.key}`}
            opt={opt}
            interactive={Boolean(detail)}
            selected={selected}
            onPick={() => setSelected(opt.key)}
          />
        ))}
      </div>
    </div>
  )
}

function StemPreview({ question }: { question: MathQuizQuestion }) {
  const v = question.visual
  if (question.stemKind === 'equation' && question.stemText) {
    return <div className="math-stem-eq">{question.stemText}</div>
  }
  if (question.stemKind === 'shape') {
    return (
      <div className="math-stem-shape" aria-label={question.stemText}>
        <ShapeGlyph shape={SHAPE_KEYS[question.stemText] ?? ''} />
      </div>
    )
  }
  if (question.stemKind === 'visual' && v) {
    const emoji = v.emoji || '🍎'
    if (v.kind === 'add') {
      return (
        <div className="math-stem-emoji">
          {emoji.repeat(v.a)} + {emoji.repeat(v.b)}
        </div>
      )
    }
    if (v.kind === 'sub') {
      return (
        <div className="math-stem-emoji">
          {emoji.repeat(v.a)}
          <span className="muted"> 拿走 {v.b}</span>
        </div>
      )
    }
  }
  return null
}

function ShapeGlyph({ shape }: { shape: string }) {
  const common = { width: 72, height: 72, viewBox: '0 0 100 100' }
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
      return <span>{shape}</span>
  }
}

function OptionTile({
  opt,
  interactive,
  selected,
  onPick,
}: {
  opt: MathQuizOption
  interactive?: boolean
  selected: string | null
  onPick: () => void
}) {
  let body: ReactNode
  if (opt.style === 'shape' && opt.shape) {
    body = (
      <span className="pinyin-letter-card math-shape-opt-card" aria-hidden="true">
        <span className="math-shape-opt-inner">
          <ShapeGlyph shape={opt.shape} />
        </span>
      </span>
    )
  } else {
    // 数字 / 图形名称：纯白正方形 + 横排字（无田字格）
    body = (
      <span className="math-label-card" aria-hidden="true">
        <span className={`math-label-text${opt.label.length > 2 ? ' is-long' : ''}`}>
          {opt.label}
        </span>
      </span>
    )
  }

  let cls = 'quiz-option-tile pinyin-letter-opt'
  if (!interactive && opt.correct) cls += ' is-correct-hint'
  if (interactive && selected === opt.key) {
    cls += opt.correct ? ' is-correct' : ' is-wrong'
  }

  if (!interactive) {
    return <div className={cls}>{body}</div>
  }
  return (
    <button type="button" className={cls} onClick={onPick} disabled={selected !== null}>
      {body}
    </button>
  )
}
