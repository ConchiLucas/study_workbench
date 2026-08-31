import type { QuestionVisual as Visual } from '../api/plans'
import { ShapeGlyph } from './ShapeGlyph'

type Size = 'kid' | 'compact'

const SIZE = {
  kid: {
    emoji: 'text-[40px]',
    bigEmoji: 'text-[96px]',
    seq: 'text-[52px]',
    charShort: 'text-[76px]',
    charLong: 'text-[40px]',
    shape: 116,
    op: 'text-4xl',
    gap: 'gap-4',
    charPad: 'rounded-3xl px-8 py-3',
  },
  compact: {
    emoji: 'text-[28px]',
    bigEmoji: 'text-[56px]',
    seq: 'text-[36px]',
    charShort: 'text-[40px]',
    charLong: 'text-[28px]',
    shape: 72,
    op: 'text-2xl',
    gap: 'gap-2',
    charPad: 'rounded-2xl px-5 py-2',
  },
} as const

/** 一行最多 5 个，超过换行——10 个挤成一排数不清。 */
function EmojiRow({
  count,
  emoji,
  faded = 0,
  size,
}: {
  count: number
  emoji: string
  faded?: number
  size: Size
}) {
  const items = Array.from({ length: count }, (_, i) => i)
  return (
    <div className="flex max-w-[280px] flex-wrap justify-center gap-1">
      {items.map((i) => (
        <span
          key={i}
          className={
            `${SIZE[size].emoji} leading-none transition-opacity ` +
            (i >= count - faded ? 'opacity-25 grayscale' : '')
          }
        >
          {emoji}
        </span>
      ))}
    </div>
  )
}

/**
 * 题干上方的图形区。把抽象的数字和算式变成能数的实物——
 * 5 岁孩子先会数苹果，才会算 2+5。
 *
 * size=compact 给家长端复盘用，避免整页都被大号 emoji 撑开。
 */
export function QuestionVisual({
  visual,
  size = 'kid',
}: {
  visual: Visual
  size?: Size
}) {
  const emoji = visual.emoji || '🍎'
  const s = SIZE[size]

  switch (visual.kind) {
    case 'count':
      return <EmojiRow count={visual.a ?? 0} emoji={emoji} size={size} />

    case 'add':
      return (
        <div className={`flex items-center justify-center ${s.gap}`}>
          <EmojiRow count={visual.a ?? 0} emoji={emoji} size={size} />
          <span className={`${s.op} font-bold text-candy-mute`}>+</span>
          <EmojiRow count={visual.b ?? 0} emoji={emoji} size={size} />
        </div>
      )

    case 'sub':
      // 减法用"划掉"表达：拿走的部分变灰，剩下的才是答案。
      return (
        <div className="flex flex-col items-center gap-1">
          <EmojiRow count={visual.a ?? 0} emoji={emoji} faded={visual.b ?? 0} size={size} />
          <span className="text-sm text-candy-mute">拿走 {visual.b ?? 0} 个</span>
        </div>
      )

    case 'compare':
      return (
        <div className={`flex items-center justify-center ${s.gap}`}>
          <div className="flex flex-col items-center gap-1">
            <span className="text-xs text-candy-mute">左边</span>
            <EmojiRow count={visual.a ?? 0} emoji={emoji} size={size} />
          </div>
          <span className={`${s.op} font-bold text-candy-mute`}>○</span>
          <div className="flex flex-col items-center gap-1">
            <span className="text-xs text-candy-mute">右边</span>
            <EmojiRow count={visual.b ?? 0} emoji={emoji} size={size} />
          </div>
        </div>
      )

    case 'shape':
      return <ShapeGlyph shape={visual.text ?? ''} size={s.shape} />

    case 'char': {
      // 单字用超大号；古诗首句稍小，避免五个字撑破屏幕。
      const long = (visual.text?.length ?? 0) > 2
      return (
        <div
          className={
            `${s.charPad} bg-white/70 font-bold leading-tight text-candy-ink ` +
            (long ? s.charLong : s.charShort)
          }
        >
          {visual.text}
        </div>
      )
    }

    case 'emoji':
      return <span className={`${s.bigEmoji} leading-none`}>{visual.emoji}</span>

    case 'seq':
      // 规律题：把序列摊开，末尾用问号提示「下一个」
      return (
        <div className="flex flex-wrap items-center justify-center gap-2">
          {(visual.items ?? []).map((it, i) => (
            <span key={i} className={`${s.seq} leading-none`}>
              {it}
            </span>
          ))}
          <span className={`${s.seq} font-bold text-candy-mute`}>？</span>
        </div>
      )

    default:
      return null
  }
}
