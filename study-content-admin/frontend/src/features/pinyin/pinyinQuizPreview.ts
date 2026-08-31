import type { PinyinItem } from '../../api/pinyinTypes'
import { speechAudioURL } from '../../api/pinyin'

export type PinyinQuizType = 'inword' | 'listen'

export interface PinyinQuizOption {
  kpId: number
  letter: string
  correct: boolean
}

export interface PinyinQuizQuestion {
  id: string
  type: PinyinQuizType
  title: string
  prompt: string
  target: PinyinItem
  stemKind: 'word' | 'speech'
  speechKind: 'word' | 'solo'
  speechUrl?: string
  options: PinyinQuizOption[]
  available: boolean
  unavailableReason?: string
}

/** 与 backend quiz/pinyin.go pinyinConfusion 对齐 */
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

const TYPE_META: Record<
  PinyinQuizType,
  { title: string; prompt: string; stemKind: PinyinQuizQuestion['stemKind']; speechKind: 'word' | 'solo' }
> = {
  inword: {
    title: '听例字选音',
    prompt: '听一听，这个字里有哪个音？',
    stemKind: 'word',
    speechKind: 'word',
  },
  listen: {
    title: '听单读选字母',
    prompt: '听一听，这是哪个字母？',
    stemKind: 'speech',
    speechKind: 'solo',
  },
}

export const PINYIN_QUIZ_TYPES: PinyinQuizType[] = ['inword', 'listen']

function shuffle<T>(items: T[], random: () => number = Math.random): T[] {
  const out = [...items]
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.floor(random() * (i + 1))
    ;[out[i], out[j]] = [out[j], out[i]]
  }
  return out
}

/** 第一层易混组，第二层同组其余字母（与 workbench confusionTiers 一致）。 */
function pickDistractorLetters(
  letter: string,
  groupLetters: string[],
  random: () => number,
): string[] {
  const pool = groupLetters.filter((l) => l !== letter)
  const seen = new Set<string>([letter])
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
  const rest = pool.filter((l) => !seen.has(l))
  return shuffle([...confusing, ...rest], random).slice(0, 3)
}

function letterToItem(letter: string, groupItems: PinyinItem[]): PinyinItem | undefined {
  return groupItems.find((i) => i.letter === letter)
}

function optionFromLetter(
  letter: string,
  groupItems: PinyinItem[],
  correct: boolean,
): PinyinQuizOption | null {
  const item = letterToItem(letter, groupItems)
  if (!item) return null
  return { kpId: item.kpId, letter, correct }
}

function eligibleForType(item: PinyinItem, type: PinyinQuizType): boolean {
  if (type === 'inword') return Boolean(item.wordText?.trim())
  return Boolean(item.soloText?.trim())
}

export function buildQuestionForItem(
  target: PinyinItem,
  groupItems: PinyinItem[],
  type: PinyinQuizType,
  random: () => number = Math.random,
): PinyinQuizQuestion | null {
  const meta = TYPE_META[type]
  const id = `${target.kpId}:${type}`

  if (!eligibleForType(target, type)) {
    return null
  }

  const groupLetters = groupItems.map((i) => i.letter)
  const distractorLetters = pickDistractorLetters(target.letter, groupLetters, random)
  if (distractorLetters.length < 3) {
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

  const distractorOptions = distractorLetters
    .map((l) => optionFromLetter(l, groupItems, false))
    .filter((o): o is PinyinQuizOption => o !== null)
  const correctOption = optionFromLetter(target.letter, groupItems, true)
  if (!correctOption || distractorOptions.length < 3) {
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

  const speechUrl =
    type === 'inword'
      ? speechAudioURL(target.kpId, 'word', target.wordSpeechUrl || undefined)
      : speechAudioURL(target.kpId, 'solo', target.soloSpeechUrl || undefined)

  const options = shuffle([correctOption, ...distractorOptions], random)

  return {
    id,
    type,
    ...meta,
    target,
    speechUrl,
    options,
    available: true,
  }
}

/** 每组字母 × 两类题；listen 无 soloText 时跳过。 */
export function buildGroupQuestions(
  groupItems: PinyinItem[],
  random: () => number = Math.random,
): PinyinQuizQuestion[] {
  const out: PinyinQuizQuestion[] = []
  for (const item of groupItems) {
    for (const type of PINYIN_QUIZ_TYPES) {
      const q = buildQuestionForItem(item, groupItems, type, random)
      if (q) out.push(q)
    }
  }
  return out
}
