export interface HighlightSegment {
  text: string
  highlighted: boolean
}

const ASCII_ALPHANUMERIC = /[A-Za-z0-9]/

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function isAsciiAlphanumeric(value: string | undefined) {
  return Boolean(value && ASCII_ALPHANUMERIC.test(value))
}

export function splitHighlightedSentence(
  sentence?: string | null,
  word?: string | null,
): HighlightSegment[] {
  if (!sentence) {
    return []
  }

  const target = word?.trim() || ''
  if (!target) {
    return [{ text: sentence, highlighted: false }]
  }

  const matcher = new RegExp(escapeRegExp(target), 'gi')
  const requiresLeftBoundary = isAsciiAlphanumeric(target[0])
  const requiresRightBoundary = isAsciiAlphanumeric(target[target.length - 1])
  const segments: HighlightSegment[] = []
  let cursor = 0
  let found = false

  for (const match of sentence.matchAll(matcher)) {
    if (match.index === undefined) {
      continue
    }
    const start = match.index
    const end = start + match[0].length
    const hasValidLeftBoundary = !requiresLeftBoundary
      || !isAsciiAlphanumeric(sentence[start - 1])
    const hasValidRightBoundary = !requiresRightBoundary
      || !isAsciiAlphanumeric(sentence[end])

    if (!hasValidLeftBoundary || !hasValidRightBoundary) {
      continue
    }

    if (start > cursor) {
      segments.push({ text: sentence.slice(cursor, start), highlighted: false })
    }
    segments.push({ text: sentence.slice(start, end), highlighted: true })
    cursor = end
    found = true
  }

  if (!found) {
    return [{ text: sentence, highlighted: false }]
  }

  if (cursor < sentence.length) {
    segments.push({ text: sentence.slice(cursor), highlighted: false })
  }
  return segments
}
