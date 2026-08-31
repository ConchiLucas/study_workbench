import { describe, expect, it } from 'vitest'
import { splitHighlightedSentence } from './highlightSentence'

const highlightedText = (sentence: string, word: string) =>
  splitHighlightedSentence(sentence, word)
    .filter((segment) => segment.highlighted)
    .map((segment) => segment.text)

describe('splitHighlightedSentence', () => {
  it('highlights a target while preserving sentence case', () => {
    expect(highlightedText('This Word is useful.', 'word')).toEqual(['Word'])
  })

  it('highlights a multi-word phrase', () => {
    expect(highlightedText('Do not make a mistake today.', 'make a mistake'))
      .toEqual(['make a mistake'])
  })

  it('highlights every legal occurrence', () => {
    expect(highlightedText('Word by word.', 'word')).toEqual(['Word', 'word'])
  })

  it('does not match inside a longer alphanumeric word', () => {
    expect(highlightedText('Draw the raw material.', 'raw')).toEqual(['raw'])
  })

  it('escapes regular-expression characters', () => {
    expect(highlightedText('Use c++ carefully.', 'c++')).toEqual(['c++'])
  })

  it('returns one plain segment for empty or unmatched targets', () => {
    expect(splitHighlightedSentence('A sentence.', '')).toEqual([
      { text: 'A sentence.', highlighted: false },
    ])
    expect(splitHighlightedSentence('A sentence.', 'word')).toEqual([
      { text: 'A sentence.', highlighted: false },
    ])
  })

  it('returns no segments when the sentence is empty', () => {
    expect(splitHighlightedSentence('', 'word')).toEqual([])
    expect(splitHighlightedSentence(null, 'word')).toEqual([])
  })
})
