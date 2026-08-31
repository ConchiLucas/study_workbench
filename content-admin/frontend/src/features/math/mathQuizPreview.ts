import type { MathItem } from '../../api/mathTypes'

export type MathQuizType = 'calc' | 'story' | 'find' | 'name'

export interface MathQuizOption {
  key: string
  label: string
  shape?: string
  correct: boolean
  style: 'number' | 'text' | 'shape'
}

export interface MathQuizQuestion {
  id: string
  type: MathQuizType
  title: string
  prompt: string
  stemKind: 'equation' | 'visual' | 'speech' | 'shape'
  stemText: string
  target: MathItem
  speechText: string
  speechUrl?: string
  visual?: { kind: string; a: number; b: number; emoji?: string }
  options: MathQuizOption[]
  available: boolean
  unavailableReason?: string
}

const CN_DIGITS = ['零', '一', '二', '三', '四', '五', '六', '七', '八', '九', '十']
export const SHAPE_KEYS: Record<string, string> = {
  圆形: 'circle',
  正方形: 'square',
  长方形: 'rect',
  三角形: 'triangle',
  椭圆形: 'oval',
  梯形: 'trapezoid',
  菱形: 'rhombus',
  五角星: 'star',
}

export function cnNumber(n: number): string {
  if (n < 0) return String(n)
  if (n < CN_DIGITS.length) return CN_DIGITS[n]
  if (n < 20) return `十${CN_DIGITS[n - 10]}`
  if (n === 20) return '二十'
  return String(n)
}

function shuffle<T>(items: T[], random: () => number = Math.random): T[] {
  const out = [...items]
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.floor(random() * (i + 1))
    ;[out[i], out[j]] = [out[j], out[i]]
  }
  return out
}

export function consecutiveWindow(n: number, lo: number, hi: number, random: () => number): number[] {
  const size = 4
  if (hi - lo + 1 < size) return []
  const maxStart = hi - size + 1
  const starts: number[] = []
  for (let s = n - size + 1; s <= n; s++) {
    if (s >= lo && s <= maxStart) starts.push(s)
  }
  if (starts.length === 0) return []
  const start = starts[Math.floor(random() * starts.length)]
  return [start, start + 1, start + 2, start + 3]
}

function parsePayload(item: MathItem): { kind: string; a: number; b: number; emoji?: string } | null {
  try {
    const p = JSON.parse(item.payload) as { kind?: string; a?: number; b?: number; emoji?: string }
    if (p.kind !== 'add' && p.kind !== 'sub') return null
    return {
      kind: p.kind,
      a: typeof p.a === 'number' ? p.a : 0,
      b: typeof p.b === 'number' ? p.b : 0,
      emoji: p.emoji,
    }
  } catch {
    return null
  }
}

function numberOptions(answer: number, random: () => number): MathQuizOption[] | null {
  const window = consecutiveWindow(answer, 0, 20, random)
  if (window.length !== 4) return null
  return shuffle(
    window.map((v) => ({
      key: String(v),
      label: String(v),
      correct: v === answer,
      style: 'number' as const,
    })),
    random,
  )
}

function pickLabels(correct: string, pool: string[], random: () => number): MathQuizOption[] | null {
  const rest = pool.filter((x) => x !== correct)
  if (rest.length < 3) return null
  const picked = shuffle(rest, random).slice(0, 3)
  return shuffle(
    [correct, ...picked].map((label) => ({
      key: label,
      label,
      correct: label === correct,
      style: 'text' as const,
    })),
    random,
  )
}

function buildArithmetic(item: MathItem, random: () => number): MathQuizQuestion[] {
  const p = parsePayload(item)
  if (!p) return []
  const result = p.kind === 'add' ? p.a + p.b : p.a - p.b
  if (result < 0 || result > 20) return []
  const sign = p.kind === 'add' ? '+' : '-'
  const verb = p.kind === 'add' ? '加' : '减'
  const story = p.kind === 'add' ? '一共有几个？' : '还剩几个？'
  const emoji = p.emoji || (p.kind === 'add' ? '🍎' : '🍓')
  const calcOpts = numberOptions(result, random)
  const storyOpts = numberOptions(result, random)
  if (!calcOpts || !storyOpts) return []

  return [
    {
      id: `${item.kpId}:calc`,
      type: 'calc',
      title: '看算式选答案',
      prompt: `${p.a} ${sign} ${p.b} = ?`,
      stemKind: 'equation',
      stemText: `${p.a} ${sign} ${p.b}`,
      target: item,
      speechText: item.speechText || `${cnNumber(p.a)}${verb}${cnNumber(p.b)}等于几`,
      speechUrl: item.speechAudioUrl,
      visual: { kind: p.kind, a: p.a, b: p.b, emoji },
      options: calcOpts,
      available: true,
    },
    {
      id: `${item.kpId}:story`,
      type: 'story',
      title: '看图选答案',
      prompt: story,
      stemKind: 'visual',
      stemText: '',
      target: item,
      speechText: story,
      speechUrl: item.speechAudioUrl,
      visual: { kind: p.kind, a: p.a, b: p.b, emoji },
      options: storyOpts,
      available: true,
    },
  ]
}

function buildShape(item: MathItem, groupItems: MathItem[], random: () => number): MathQuizQuestion[] {
  const title = item.title
  const key = SHAPE_KEYS[title]
  if (!key) return []
  const names = groupItems.map((i) => i.title).filter((n) => SHAPE_KEYS[n])
  const uniq = [...new Set(names)]
  if (uniq.length < 4) return []
  const nameOpts = pickLabels(title, uniq, random)
  const shapePool = uniq.filter((n) => n !== title)
  const picked = shuffle(shapePool, random).slice(0, 3)
  const findOpts = shuffle(
    [title, ...picked].map((n) => ({
      key: n,
      label: n,
      shape: SHAPE_KEYS[n],
      correct: n === title,
      style: 'shape' as const,
    })),
    random,
  )
  if (!nameOpts) return []
  return [
    {
      id: `${item.kpId}:find`,
      type: 'find',
      title: '看名称选图形',
      prompt: `找出「${title}」`,
      stemKind: 'speech',
      stemText: title,
      target: item,
      speechText: item.speechText || title,
      speechUrl: item.speechAudioUrl,
      options: findOpts,
      available: true,
    },
    {
      id: `${item.kpId}:name`,
      type: 'name',
      title: '看图形选名称',
      prompt: '这是什么图形？',
      stemKind: 'shape',
      stemText: title,
      target: item,
      speechText: '这是什么图形',
      speechUrl: item.speechAudioUrl,
      options: nameOpts,
      available: true,
    },
  ]
}

/** 加减法 + 认识图形预览（不含比大小）。 */
export function buildMathGroupQuestions(
  items: MathItem[],
  random: () => number = Math.random,
): MathQuizQuestion[] {
  if (items.length === 0) return []
  const moduleCode = items[0].moduleCode
  const out: MathQuizQuestion[] = []
  if (moduleCode === 'add10' || moduleCode === 'sub10') {
    for (const item of items.slice(0, 30)) {
      out.push(...buildArithmetic(item, random))
    }
  } else if (moduleCode === 'shape') {
    for (const item of items) {
      out.push(...buildShape(item, items, random))
    }
  }
  return out
}
