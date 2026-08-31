import { useEffect, useState, type MouseEvent } from 'react'
import { speechAudioURL } from '../../api/pinyin'
import type { PinyinQuizOption, PinyinQuizQuestion } from './pinyinQuizPreview'

type AnswerState = 'idle' | 'correct' | 'wrong'

export function PinyinQuizRow({
  question,
  mode = 'list',
  onOpen,
}: {
  question: PinyinQuizQuestion
  mode?: 'list' | 'detail'
  onOpen?: () => void
}) {
  const [selected, setSelected] = useState<number | null>(null)
  const [answer, setAnswer] = useState<AnswerState>('idle')

  useEffect(() => {
    setSelected(null)
    setAnswer('idle')
  }, [question.id])

  const interactive = mode === 'detail' && question.available

  function onPick(opt: PinyinQuizOption) {
    if (!interactive || answer !== 'idle') return
    setSelected(opt.kpId)
    setAnswer(opt.correct ? 'correct' : 'wrong')
  }

  function onPlayStem(e?: MouseEvent) {
    e?.stopPropagation()
    void playPinyinSpeech({
      kpId: question.target.kpId,
      kind: question.speechKind,
      speechUrl: question.speechUrl,
      storedUrl:
        question.speechKind === 'word'
          ? question.target.wordSpeechUrl
          : question.target.soloSpeechUrl,
    }).catch(() => undefined)
  }

  if (!question.available) {
    const body = (
      <div className={`quiz-row quiz-row-unavailable${mode === 'detail' ? ' quiz-row-detail' : ''}`}>
        <div className="quiz-stem">
          <span className="quiz-type-tag">简单 · {question.title}</span>
          <div className="quiz-stem-body muted">
            {question.target.letter} · {question.unavailableReason}
          </div>
        </div>
        <div className="quiz-opt-col muted">—</div>
        <div className="quiz-opt-col muted">—</div>
      </div>
    )
    if (mode === 'list' && onOpen) {
      return (
        <div
          className="quiz-row-hit"
          role="button"
          tabIndex={0}
          onClick={onOpen}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault()
              onOpen()
            }
          }}
        >
          {body}
        </div>
      )
    }
    return body
  }

  const [a, b, c, d] = question.options
  const left = [a, c].filter(Boolean)
  const right = [b, d].filter(Boolean)

  const row = (
    <div className={`quiz-row${mode === 'detail' ? ' quiz-row-detail' : ''}`}>
      <div className="quiz-stem">
        <span className="quiz-type-tag">简单 · {question.title}</span>
        <StemVisual question={question} onPlayStem={onPlayStem} />
      </div>
      <div className="quiz-opt-col">
        {left.map((opt) => (
          <OptionTile
            key={`${question.id}-L-${opt.kpId}`}
            option={opt}
            interactive={interactive}
            answer={answer}
            selected={selected}
            onPick={onPick}
          />
        ))}
      </div>
      <div className="quiz-opt-col">
        {right.map((opt) => (
          <OptionTile
            key={`${question.id}-R-${opt.kpId}`}
            option={opt}
            interactive={interactive}
            answer={answer}
            selected={selected}
            onPick={onPick}
          />
        ))}
      </div>
    </div>
  )

  if (mode === 'list') {
    return (
      <div
        className="quiz-row-hit"
        role="button"
        tabIndex={0}
        onClick={onOpen}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            onOpen?.()
          }
        }}
      >
        {row}
      </div>
    )
  }

  return (
    <div className="quiz-detail-wrap">
      {row}
      {answer !== 'idle' ? (
        <div className="preview-feedback">
          <p className={answer === 'correct' ? 'ok' : 'err'}>
            {answer === 'correct'
              ? '答对了！'
              : `再想想～ 正确答案是「${question.target.letter}」`}
          </p>
          <button
            type="button"
            className="primary"
            onClick={() => {
              setSelected(null)
              setAnswer('idle')
            }}
          >
            再试一次
          </button>
        </div>
      ) : (
        <p className="muted quiz-detail-hint">点选项试答 · 与孩子端三列布局一致</p>
      )}
    </div>
  )
}

function StemVisual({
  question,
  onPlayStem,
}: {
  question: PinyinQuizQuestion
  onPlayStem: (e?: MouseEvent) => void
}) {
  return (
    <div className="quiz-stem-body">
      <p className="quiz-stem-prompt">{question.prompt}</p>
      <div className="quiz-stem-media">
        {question.stemKind === 'word' ? (
          <div className="quiz-stem-char">{question.target.wordText}</div>
        ) : null}
        <button type="button" className="quiz-speak-btn ghost" onClick={onPlayStem} title="读音">
          🔊
        </button>
      </div>
    </div>
  )
}

function OptionTile({
  option,
  interactive,
  answer,
  selected,
  onPick,
}: {
  option: PinyinQuizOption
  interactive: boolean
  answer: AnswerState
  selected: number | null
  onPick: (opt: PinyinQuizOption) => void
}) {
  let cls = 'quiz-option-tile pinyin-letter-opt'
  if (!interactive && option.correct) cls += ' is-correct-hint'
  if (interactive && answer !== 'idle') {
    if (option.correct) cls += ' is-correct'
    else if (selected === option.kpId) cls += ' is-wrong'
  }

  const card = (
    <span className="pinyin-letter-card" aria-hidden="true">
      <span className="pinyin-letter-grid">
        <span className="pinyin-letter-text">{option.letter}</span>
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
      disabled={answer !== 'idle'}
      onClick={() => onPick(option)}
    >
      {card}
    </button>
  )
}

export async function playPinyinSpeech(args: {
  kpId: number
  kind: 'solo' | 'word'
  speechUrl?: string
  storedUrl?: string
}) {
  const url = args.speechUrl || speechAudioURL(args.kpId, args.kind, args.storedUrl)
  const res = await fetch(url)
  if (!res.ok) throw new Error(`读音失败 ${res.status}`)
  const blob = await res.blob()
  const objectUrl = URL.createObjectURL(blob)
  const audio = new Audio(objectUrl)
  audio.onended = () => URL.revokeObjectURL(objectUrl)
  await audio.play()
}
