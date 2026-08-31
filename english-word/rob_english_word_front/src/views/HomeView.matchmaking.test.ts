import { describe, expect, it } from 'vitest'
import source from './HomeView.vue?raw'

describe('difficulty-scoped matchmaking source contract', () => {
  it('sends the selected difficulty in every match request', () => {
    expect(source).toContain("wsStore.send('match_start', {")
    expect(source).toContain('difficultyGroup: difficulty.parentKey')
    expect(source).toContain('difficultyLevel: difficulty.key')
  })

  it('renders matching before difficulty selection and solo training', () => {
    const setup = source.slice(source.indexOf('<div class="training-setup-actions">'))
    const difficulty = setup.indexOf('难度选择')
    const match = setup.indexOf('开始匹配')
    const solo = setup.indexOf('单人训练')

    expect(difficulty).toBeGreaterThanOrEqual(0)
    expect(match).toBeLessThan(difficulty)
    expect(difficulty).toBeLessThan(solo)
  })

  it('shows the canonical difficulty label returned by the backend', () => {
    expect(source).toContain("case 'match_waiting':")
    expect(source).toContain('message.data?.difficultyLabel')
    expect(source).toContain('matchingDifficultyLabel')
  })
})
