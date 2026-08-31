import type { ReactElement } from 'react'

/**
 * 图形题的图形用 SVG 画，不用 emoji。
 *
 * Unicode 里没有可靠的长方形、椭圆、梯形字符，各系统字体表现差异很大
 * ——「▬」在有的字体里细成一条线，孩子根本看不出是长方形。
 */
const PATHS: Record<string, ReactElement> = {
  circle: <circle cx="50" cy="50" r="34" />,
  square: <rect x="18" y="18" width="64" height="64" rx="4" />,
  rect: <rect x="10" y="30" width="80" height="40" rx="4" />,
  triangle: <polygon points="50,16 86,82 14,82" />,
  oval: <ellipse cx="50" cy="50" rx="40" ry="26" />,
  trapezoid: <polygon points="30,22 70,22 88,78 12,78" />,
  rhombus: <polygon points="50,12 86,50 50,88 14,50" />,
  star: (
    <polygon points="50,10 61,38 92,38 66,57 76,87 50,68 24,87 34,57 8,38 39,38" />
  ),
}

export function ShapeGlyph({ shape, size = 96, color = '#3D3230' }: {
  shape: string
  size?: number
  color?: string
}) {
  const path = PATHS[shape]
  if (!path) return null
  return (
    <svg viewBox="0 0 100 100" width={size} height={size} fill={color} aria-hidden>
      {path}
    </svg>
  )
}
