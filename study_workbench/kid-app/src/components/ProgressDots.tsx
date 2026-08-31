import clsx from 'clsx'

/**
 * 进度用圆点而不是百分比或"3/10"——数字对 5 岁孩子不直观，
 * 一排点点能看出"还剩几个就做完了"。
 * 刻意不区分对错：孩子只看进度，正确率给家长看。
 */
export function ProgressDots({ total, done, current }: {
  total: number
  done: number
  current: number
}) {
  return (
    <div className="flex items-center gap-1.5">
      {Array.from({ length: total }, (_, i) => (
        <span
          key={i}
          className={clsx(
            'h-3 rounded-full transition-all duration-300',
            i < done ? 'w-3 bg-candy-go' : i === current ? 'w-7 bg-candy-ink/40' : 'w-3 bg-candy-ink/15',
          )}
        />
      ))}
    </div>
  )
}
