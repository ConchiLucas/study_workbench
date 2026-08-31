import type { LiteracyChar } from '../../api/literacyTypes'
import { speechAudioURL } from '../../api/literacy'

export type LiteracyQuizType = 'glyph_sense' | 'sense_char'

export type LiteracyQuizOptionKind = 'glyph' | 'sense' | 'char'

export interface LiteracyQuizOption {
  kpId: number
  charText: string
  kind: LiteracyQuizOptionKind
  imageUrl?: string
  speechUrl?: string
  correct: boolean
}

export interface LiteracyQuizQuestion {
  id: string
  type: LiteracyQuizType
  difficulty: 'medium' | 'easy'
  title: string
  target: LiteracyChar
  stemKind: 'glyph' | 'sense'
  optionKind: LiteracyQuizOptionKind
  speechUrl?: string
  options: LiteracyQuizOption[]
  available: boolean
  unavailableReason?: string
}

const TYPE_META: Record<
  LiteracyQuizType,
  { difficulty: 'medium' | 'easy'; title: string; stemKind: LiteracyQuizQuestion['stemKind']; optionKind: LiteracyQuizOptionKind }
> = {
  glyph_sense: {
    difficulty: 'easy',
    title: '看字图选义图',
    stemKind: 'glyph',
    optionKind: 'sense',
  },
  sense_char: {
    difficulty: 'easy',
    title: '看义图选字',
    stemKind: 'sense',
    optionKind: 'char',
  },
}

export const LITERACY_QUIZ_TYPES: LiteracyQuizType[] = ['glyph_sense', 'sense_char']

function shuffle<T>(items: T[], random: () => number = Math.random): T[] {
  const out = [...items]
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.floor(random() * (i + 1))
    ;[out[i], out[j]] = [out[j], out[i]]
  }
  return out
}

function pickDistractors(pool: LiteracyChar[], count: number, random: () => number): LiteracyChar[] {
  return shuffle(pool, random).slice(0, count)
}

function optionFromChar(
  char: LiteracyChar,
  kind: LiteracyQuizOptionKind,
  correct: boolean,
): LiteracyQuizOption {
  const speechUrl = speechAudioURL(char.kpId, char.speechAudioUrl || undefined)
  if (kind === 'glyph') {
    return {
      kpId: char.kpId,
      charText: char.charText,
      kind,
      imageUrl: char.glyphImageUrl || undefined,
      speechUrl,
      correct,
    }
  }
  if (kind === 'sense') {
    return {
      kpId: char.kpId,
      charText: char.charText,
      kind,
      imageUrl: char.senseImageUrl || undefined,
      speechUrl,
      correct,
    }
  }
  return {
    kpId: char.kpId,
    charText: char.charText,
    kind: 'char',
    imageUrl: char.glyphImageUrl || undefined,
    speechUrl,
    correct,
  }
}

function eligibleForType(char: LiteracyChar, type: LiteracyQuizType): boolean {
  if (type === 'glyph_sense') {
    return Boolean(char.glyphImageUrl && char.senseImageUrl && char.effectiveNeedsSenseImage)
  }
  return Boolean(char.senseImageUrl && char.effectiveNeedsSenseImage)
}

export function buildQuestionForChar(
  target: LiteracyChar,
  groupChars: LiteracyChar[],
  type: LiteracyQuizType,
  random: () => number = Math.random,
): LiteracyQuizQuestion {
  const meta = TYPE_META[type]
  const id = `${target.kpId}:${type}`

  if (!eligibleForType(target, type)) {
    return {
      id,
      type,
      ...meta,
      target,
      options: [],
      available: false,
      unavailableReason: type === 'glyph_sense' ? '缺少字图或义图' : '缺少义图',
    }
  }

  const pool = groupChars.filter(
    (c) => c.kpId !== target.kpId && eligibleForType(c, type),
  )
  if (pool.length < 3) {
    return {
      id,
      type,
      ...meta,
      target,
      options: [],
      available: false,
      unavailableReason: '同组可用干扰项不足 3 个',
    }
  }

  const distractors = pickDistractors(pool, 3, random)
  const options = shuffle(
    [
      optionFromChar(target, meta.optionKind, true),
      ...distractors.map((c) => optionFromChar(c, meta.optionKind, false)),
    ],
    random,
  )

  return {
    id,
    type,
    ...meta,
    target,
    speechUrl: speechAudioURL(target.kpId, target.speechAudioUrl || undefined),
    options,
    available: true,
  }
}

/** Every char × two types; options re-rolled each call. */
export function buildGroupQuestions(
  groupChars: LiteracyChar[],
  random: () => number = Math.random,
): LiteracyQuizQuestion[] {
  const out: LiteracyQuizQuestion[] = []
  for (const char of groupChars) {
    for (const type of LITERACY_QUIZ_TYPES) {
      out.push(buildQuestionForChar(char, groupChars, type, random))
    }
  }
  return out
}

export function buildPlayableQueue(
  groupChars: LiteracyChar[],
  random: () => number = Math.random,
): LiteracyQuizQuestion[] {
  return buildGroupQuestions(groupChars, random).filter((q) => q.available)
}
