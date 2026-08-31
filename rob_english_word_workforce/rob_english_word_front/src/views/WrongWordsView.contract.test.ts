import { describe, expect, it } from 'vitest'
import source from './WrongWordsView.vue?raw'

describe('WrongWordsView unfinished word contract', () => {
  it('uses the persistent review endpoint projection and approved columns', () => {
    expect(source).toContain('/api/wrong-words/events')
    for (const label of [
      '来源单词',
      '例句',
      '答错时间',
      '入口 / 模式',
      '词库 / 难度',
      '词难度',
      '耗时',
      '正确答案',
    ]) {
      expect(source).toContain(label)
    }
    expect(source).toContain('待复习单词')
    expect(source).toContain('未完成复习的错词会持续显示')
    expect(source).toContain(':key="item.progressKey"')
    expect(source).toContain('exampleSentence')
    expect(source).toContain('exampleSource')
    expect(source).toContain('splitHighlightedSentence')
    expect(source).toContain('class="example-cell"')
    expect(source).toContain('class="example-highlight"')
    expect(source).toContain('data-label="例句"')
    expect(source).not.toContain('v-html')
    expect(source).not.toContain('可入队错题记录')
  })

  it('keeps the clamped sentence inside the responsive card field', () => {
    expect(source).toMatch(
      /class="example-cell"[\s\S]*?data-label="例句"[\s\S]*?class="example-sentence"/,
    )
    expect(source).toContain('-webkit-line-clamp: 2')
  })

  it('does not render the removed answer column or legacy detail modal', () => {
    expect(source).not.toContain('你的答案')
    expect(source).not.toContain('/details')
    expect(source).not.toContain('detail-modal')
  })
})
