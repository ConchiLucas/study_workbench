import { useEffect, useState, type MouseEvent } from 'react'
import { speechAudioURL } from '../../api/literacy'
import type { LiteracyQuizOption, LiteracyQuizQuestion } from './quizPreview'

type AnswerState = 'idle' | 'correct' | 'wrong'

export function QuizQuestionRow({
  question,
  mode = 'list',
  onOpen,
}: {
  question: LiteracyQuizQuestion
  mode?: 'list' | 'detail'
  onOpen?: () => void
}) {
  const difficultyLabel = question.difficulty === 'medium' ? '中等' : '简单'
  const [selected, setSelected] = useState<number | null>(null)
  const [answer, setAnswer] = useState<AnswerState>('idle')

  useEffect(() => {
    setSelected(null)
    setAnswer('idle')
  }, [question.id])

  const interactive = mode === 'detail' && question.available

  function onPick(opt: LiteracyQuizOption) {
    if (!interactive || answer !== 'idle') return
    setSelected(opt.kpId)
    setAnswer(opt.correct ? 'correct' : 'wrong')
  }

  function onPlayStem(e?: MouseEvent) {
    e?.stopPropagation()
    void playCharSpeech({
      kpId: question.target.kpId,
      charText: question.target.charText,
      speechUrl: question.speechUrl,
      speechAudioUrl: question.target.speechAudioUrl,
    }).catch(() => undefined)
  }

  if (!question.available) {
    const body = (
      <div className={`quiz-row quiz-row-unavailable${mode === 'detail' ? ' quiz-row-detail' : ''}`}>
        <div className="quiz-stem">
          <span className="quiz-type-tag">{difficultyLabel} · {question.title}</span>
          <div className="quiz-stem-body muted">
            {question.target.charText} · {question.unavailableReason}
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
  const optionSpeech = question.type === 'glyph_sense'

  const row = (
    <div className={`quiz-row${mode === 'detail' ? ' quiz-row-detail' : ''}`}>
      <div className="quiz-stem">
        <span className="quiz-type-tag">{difficultyLabel} · {question.title}</span>
        <StemVisual question={question} onPlayStem={onPlayStem} showStemSpeech={question.type === 'sense_char'} />
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
            showSpeech={optionSpeech}
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
            showSpeech={optionSpeech}
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
              : `再想想～ 正确答案是「${question.target.charText}」`}
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
  showStemSpeech,
}: {
  question: LiteracyQuizQuestion
  onPlayStem: (e?: MouseEvent) => void
  showStemSpeech: boolean
}) {
  if (question.stemKind === 'glyph') {
    return (
      <div className="quiz-stem-body">
        <p className="quiz-stem-prompt">看字图，选出义图</p>
        <div className="quiz-stem-media">
          {question.target.glyphImageUrl ? (
            <img
              className="quiz-stem-img"
              src={question.target.glyphImageUrl}
              alt={question.target.charText}
            />
          ) : (
            <div className="quiz-stem-char">{question.target.charText}</div>
          )}
        </div>
      </div>
    )
  }
  return (
    <div className="quiz-stem-body">
      <p className="quiz-stem-prompt">看义图，选出字</p>
      <div className="quiz-stem-media">
        {question.target.senseImageUrl ? (
          <img
            className="quiz-stem-img"
            src={question.target.senseImageUrl}
            alt={`${question.target.charText}义图`}
          />
        ) : (
          <div className="sense-placeholder">无义图</div>
        )}
        {showStemSpeech ? (
          <button type="button" className="quiz-speak-btn ghost" onClick={onPlayStem} title="读音">
            🔊
          </button>
        ) : null}
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
  showSpeech,
}: {
  option: LiteracyQuizOption
  interactive: boolean
  answer: AnswerState
  selected: number | null
  onPick: (opt: LiteracyQuizOption) => void
  showSpeech: boolean
}) {
  let cls = 'quiz-option-tile'
  if (showSpeech) cls += ' has-speech'
  if (!interactive && option.correct) cls += ' is-correct-hint'
  if (interactive && answer !== 'idle') {
    if (option.correct) cls += ' is-correct'
    else if (selected === option.kpId) cls += ' is-wrong'
  }

  const speechBtn = showSpeech ? (
    <button
      type="button"
      className="quiz-option-speak"
      title={`读「${option.charText}」`}
      onClick={(e) => {
        e.stopPropagation()
        void playCharSpeech({
          kpId: option.kpId,
          charText: option.charText,
          speechUrl: option.speechUrl,
        }).catch(() => undefined)
      }}
    >
      🔊
    </button>
  ) : null

  const media = option.imageUrl ? (
    <img src={option.imageUrl} alt={option.charText} />
  ) : (
    <span className="quiz-option-char">{option.charText}</span>
  )

  if (!interactive) {
    return (
      <div className={cls}>
        {media}
        {speechBtn}
      </div>
    )
  }

  // Avoid nested <button>: pick target is a div when speech control is present.
  if (showSpeech) {
    return (
      <div
        className={`${cls}${answer !== 'idle' ? '' : ' is-clickable'}`}
        role="button"
        tabIndex={answer === 'idle' ? 0 : -1}
        onClick={() => {
          if (answer === 'idle') onPick(option)
        }}
        onKeyDown={(e) => {
          if (answer !== 'idle') return
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            onPick(option)
          }
        }}
      >
        {media}
        {speechBtn}
      </div>
    )
  }

  return (
    <button
      type="button"
      className={cls}
      disabled={answer !== 'idle'}
      onClick={() => onPick(option)}
    >
      {media}
    </button>
  )
}

export async function playCharSpeech(args: {
  kpId: number
  charText: string
  speechUrl?: string
  speechAudioUrl?: string
}) {
  const url = args.speechUrl || speechAudioURL(args.kpId, args.speechAudioUrl)
  try {
    const res = await fetch(url)
    if (!res.ok) throw new Error(`读音失败 ${res.status}`)
    const blob = await res.blob()
    const objectUrl = URL.createObjectURL(blob)
    const audio = new Audio(objectUrl)
    audio.onended = () => URL.revokeObjectURL(objectUrl)
    await audio.play()
    return
  } catch {
    const text = args.charText?.trim()
    if (!text || typeof window === 'undefined' || !window.speechSynthesis) {
      throw new Error('读音失败')
    }
    const utterance = new SpeechSynthesisUtterance(text)
    utterance.lang = 'zh-CN'
    window.speechSynthesis.cancel()
    window.speechSynthesis.speak(utterance)
  }
}

export async function playQuestionSpeech(question: LiteracyQuizQuestion) {
  return playCharSpeech({
    kpId: question.target.kpId,
    charText: question.target.charText,
    speechUrl: question.speechUrl,
    speechAudioUrl: question.target.speechAudioUrl,
  })
}
