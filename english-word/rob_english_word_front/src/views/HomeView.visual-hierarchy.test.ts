import { describe, expect, it } from 'vitest'
import homeSource from './HomeView.vue?raw'

describe('HomeView action visual hierarchy', () => {
  const [templateSource, styleSource = ''] = homeSource.split('<script setup')

  it('marks every non-match entry as a secondary action', () => {
    const secondaryButtons = templateSource.match(/<button[^>]*class="[^"]*secondary-action-btn[^"]*"/g) || []

    expect(secondaryButtons).toHaveLength(6)
    expect(templateSource).not.toMatch(/class="[^"]*match-btn[^"]*secondary-action-btn/)
  })

  it('keeps matching prominent and renders secondary actions quietly', () => {
    const matchStyle = styleSource.match(/\.match-btn\s*\{([\s\S]*?)\}/)?.[1] || ''
    const secondaryStyleMatches = [...styleSource.matchAll(/\.secondary-action-btn\s*\{([\s\S]*?)\}/g)]
    const secondaryStyle = secondaryStyleMatches.find((match) => match[1].includes('box-shadow: none'))?.[1] || ''

    expect(matchStyle).toContain('linear-gradient')
    expect(matchStyle).toContain('box-shadow')
    expect(secondaryStyle).toContain('background: rgba(')
    expect(secondaryStyle).toContain('border: 1px solid')
    expect(secondaryStyle).toContain('box-shadow: none')
    expect(secondaryStyle).not.toContain('linear-gradient')
  })

  it('places the primary match action before difficulty selection', () => {
    const setupActions = templateSource.match(/<div class="training-setup-actions">([\s\S]*?)<\/div>/)?.[1] || ''

    expect(setupActions.indexOf('开始匹配')).toBeGreaterThanOrEqual(0)
    expect(setupActions.indexOf('开始匹配')).toBeLessThan(setupActions.indexOf('难度选择'))
  })
})
