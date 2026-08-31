import clsx from 'clsx'
import type { QuestionOption } from '../api/plans'
import { ShapeGlyph } from './ShapeGlyph'

/**
 * 四个固定色位。给每个位置一个稳定的颜色，孩子能靠颜色记住"我点的是哪个"，
 * 不必依赖读字——这对还不识字的年龄很重要。
 */
const SLOT = [
  { bg: '#FFB4A2', edge: '#E08974' },
  { bg: '#9FD4F5', edge: '#71ADD1' },
  { bg: '#FFDE8A', edge: '#D9B65B' },
  { bg: '#9BE6C4', edge: '#6FBE9B' },
]

export type OptionState = 'idle' | 'chosen-right' | 'chosen-wrong' | 'reveal' | 'dimmed'

export function OptionButton({ option, slot, state, disabled, onPick }: {
  option: QuestionOption
  slot: number
  state: OptionState
  disabled: boolean
  onPick: () => void
}) {
  const skin = SLOT[slot % SLOT.length]

  const content = option.shape
    ? <ShapeGlyph shape={option.shape} size={78} />
    : option.emoji
      ? <span className="text-[62px] leading-none">{option.emoji}</span>
      : (
        <span
          className={clsx(
            'font-bold leading-none text-candy-ink',
            // 汉字和数字用超大字号；英文单词较长，缩一档才不会溢出
            (option.label?.length ?? 0) > 3 ? 'text-[38px]' : 'text-[58px]',
          )}
        >
          {option.label}
        </span>
      )

  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onPick}
      aria-label={option.label ?? option.emoji ?? option.shape}
      className={clsx(
        'relative flex min-h-[132px] items-center justify-center rounded-kid',
        'transition-all duration-150 ease-out',
        // 未按下时靠偏移阴影"浮"起来，按下时下移并收缩阴影 = 实体按键手感
        'active:translate-y-[5px] active:shadow-sticker-sm',
        state === 'chosen-wrong' && 'animate-shake',
        state === 'chosen-right' && 'scale-[1.04]',
        state === 'dimmed' && 'opacity-35',
        disabled && state === 'idle' && 'opacity-60',
      )}
      style={{
        backgroundColor: state === 'chosen-right'
          ? '#3FC77F'
          : state === 'chosen-wrong'
            ? '#FB7185'
            : skin.bg,
        boxShadow:
          state === 'reveal'
            ? `0 0 0 6px #3FC77F, 0 6px 0 ${skin.edge}`
            : `0 6px 0 ${state === 'chosen-right' ? '#2FA365' : state === 'chosen-wrong' ? '#D25467' : skin.edge}`,
      }}
    >
      {content}

      {state === 'chosen-right' && (
        <span className="absolute -right-2 -top-3 animate-popIn text-4xl">✅</span>
      )}
      {state === 'reveal' && (
        <span className="absolute -right-2 -top-3 animate-popIn text-4xl">👈</span>
      )}
    </button>
  )
}
