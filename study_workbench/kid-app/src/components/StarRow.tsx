/**
 * 星星最少一颗，没有"零星"这一档。
 * 这个年龄需要的是"今天也完成了"，不是被打分。
 */
export function StarRow({ stars, size = 44, animate = false }: {
  stars: number
  size?: number
  animate?: boolean
}) {
  return (
    <div className="flex items-center justify-center gap-2">
      {[0, 1, 2].map((i) => (
        <span
          key={i}
          className={animate ? 'animate-popIn' : undefined}
          style={{
            fontSize: size,
            lineHeight: 1,
            animationDelay: animate ? `${i * 0.22}s` : undefined,
            filter: i < stars ? undefined : 'grayscale(1)',
            opacity: i < stars ? 1 : 0.28,
          }}
        >
          ⭐
        </span>
      ))}
    </div>
  )
}
