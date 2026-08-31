import { describe, expect, it } from 'vitest'
import answerDetailSource from '../components/AnswerDetailModal.vue?raw'
import closeButtonSource from '../components/FullscreenCloseButton.vue?raw'
import gameSource from './GameView.vue?raw'
import masteredSource from './MasteredWordsView.vue?raw'
import recordSource from './RecordView.vue?raw'
import resultsSource from './TrainingAnswerResultsView.vue?raw'
import homeSource from './HomeView.vue?raw'
import wrongWordsSource from './WrongWordsView.vue?raw'

describe('fullscreen navigation source contracts', () => {
  it('closes the difficulty picker back to the training setup', () => {
    expect(homeSource).not.toContain('training-back-btn')
    expect(homeSource).not.toContain('difficulty-back-btn')
    expect(homeSource).not.toContain('training-setup-user')
    expect(homeSource).toContain('@close="closeTrainingSetup"')
    expect(homeSource).toContain('@close="closeDifficultyPicker"')
    expect(homeSource).toMatch(
      /function closeTrainingSetup\(\) \{[\s\S]*?showTrainingSetup\.value = false[\s\S]*?\}/
    )
    const difficultyClose = homeSource.match(/function closeDifficultyPicker\(\) \{([\s\S]*?)\}/)
    expect(difficultyClose).not.toBeNull()
    expect(difficultyClose?.[1] || '').toContain('showDifficultyPicker.value = false')
    expect(difficultyClose?.[1] || '').not.toContain('showTrainingSetup.value = false')
    expect(homeSource).toMatch(
      /function selectDifficulty\([\s\S]*?showDifficultyPicker\.value = false[\s\S]*?\}/
    )
    expect(homeSource).toMatch(
      /function selectRankDifficulty\(\)[\s\S]*?showDifficultyPicker\.value = false[\s\S]*?\}/
    )
  })

  it('stacks the difficulty picker above the underlying shared close button', () => {
    const pickerLayer = homeSource.match(/\.difficulty-overlay\s*\{[\s\S]*?z-index:\s*(\d+)/)
    const closeButtonLayer = closeButtonSource.match(/\.fullscreen-close-button\s*\{[\s\S]*?z-index:\s*(\d+)/)

    expect(pickerLayer).not.toBeNull()
    expect(closeButtonLayer).not.toBeNull()
    expect(Number(pickerLayer?.[1])).toBeGreaterThan(Number(closeButtonLayer?.[1]))
  })

  it('closes training route pages back to the training setup', () => {
    for (const source of [resultsSource, masteredSource]) {
      expect(source).toContain("path: '/home'")
      expect(source).toContain('openTrainingSetup: true')
    }
  })

  it('uses explicit close targets on route pages', () => {
    for (const source of [masteredSource, recordSource, resultsSource, wrongWordsSource]) {
      expect(source).not.toContain('router.back()')
      expect(source).toContain('<FullscreenCloseButton')
      expect(source).toContain('useEscapeClose')
      expect(source).toContain('closePage')
    }

    expect(recordSource).toContain("router.push('/home')")
    expect(wrongWordsSource).toContain("router.push('/home')")
  })

  it('uses the shared close control for route details', () => {
    expect(recordSource.match(/<FullscreenCloseButton/g)).toHaveLength(2)
    expect(wrongWordsSource.match(/<FullscreenCloseButton/g)).toHaveLength(1)
    expect(answerDetailSource).not.toContain('back-btn')
    expect(answerDetailSource).toContain('<FullscreenCloseButton')
    expect(answerDetailSource).toContain("emit('close')")
  })

  it('closes the game result from the top-right control', () => {
    expect(gameSource).not.toContain('返回首页')
    expect(gameSource).not.toContain('home-btn')
    expect(gameSource).toContain('<FullscreenCloseButton')
    expect(gameSource).toContain('v-if="gameOver"')
    expect(gameSource).toContain('useEscapeClose')
  })
})
