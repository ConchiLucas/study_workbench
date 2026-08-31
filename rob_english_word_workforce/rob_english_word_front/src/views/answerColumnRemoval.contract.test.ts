import { describe, expect, it } from 'vitest'
import trainingResultsSource from './TrainingAnswerResultsView.vue?raw'
import wrongWordsSource from './WrongWordsView.vue?raw'

describe('user-facing answer column contract', () => {
  it('keeps the correct answer but removes the submitted-answer column', () => {
    for (const source of [trainingResultsSource, wrongWordsSource]) {
      expect(source).toContain('正确答案')
      expect(source).not.toContain('你的答案')
    }
  })

  it('does not retain the training selected-answer presentation helper', () => {
    expect(trainingResultsSource).not.toContain('selectedAnswerText')
  })
})
