import type { PlanSummary } from '../api/plans'

type Variant = 'primary' | 'catchup'

interface Props {
  plan: PlanSummary
  variant: Variant
  title: string
  subtitle: string
  actionLabel: string
  disabled?: boolean
  onAction: () => void
}

export function KidTaskCard({
  plan,
  variant,
  title,
  subtitle,
  actionLabel,
  disabled,
  onAction,
}: Props) {
  const primary = variant === 'primary'

  return (
    <div
      className={
        primary
          ? 'rounded-[2rem] bg-candy-paper px-8 py-7 text-center shadow-pop'
          : 'rounded-[1.5rem] bg-white/60 px-6 py-5 text-left shadow-sticker-sm'
      }
    >
      <div className={primary ? 'text-xl font-bold text-candy-ink' : 'text-lg font-bold text-candy-ink'}>
        {title}
      </div>
      <div className="mt-2 flex flex-wrap justify-center gap-2">
        {plan.subjects.map((s) => (
          <span
            key={s.code}
            className={
              primary
                ? 'rounded-full bg-white/80 px-3 py-1 text-base font-semibold text-candy-mute'
                : 'rounded-full bg-white/70 px-2.5 py-0.5 text-sm font-semibold text-candy-mute'
            }
          >
            {s.icon} {s.count}
          </span>
        ))}
      </div>
      <p className={primary ? 'mt-3 text-lg text-candy-mute' : 'mt-2 text-base text-candy-mute'}>
        {subtitle}
      </p>
      <button
        type="button"
        onClick={onAction}
        disabled={disabled}
        className={
          primary
            ? `mt-6 min-h-[96px] w-full rounded-kid px-10 text-3xl font-bold text-white
               transition-all active:translate-y-[6px] disabled:opacity-50`
            : `mt-4 min-h-[72px] w-full rounded-kid px-8 text-2xl font-bold text-candy-ink
               transition-all active:translate-y-[4px] disabled:opacity-50`
        }
        style={
          primary
            ? { backgroundColor: '#FF8A5B', boxShadow: '0 7px 0 #D9663C' }
            : { backgroundColor: '#FFFFFF', boxShadow: '0 5px 0 rgba(61,50,48,0.12)' }
        }
      >
        {actionLabel}
      </button>
    </div>
  )
}
