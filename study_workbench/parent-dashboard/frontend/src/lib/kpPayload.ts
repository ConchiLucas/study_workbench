/**
 * 把知识点 payload 解析成家长端详情可展示的结构。
 * 出题器也读同一份 JSON，两边字段约定保持一致。
 */
export type KpContent =
  | { kind: 'poem'; author: string; lines: string[] }
  | { kind: 'fact'; q: string; a: string; emoji?: string }
  | { kind: 'logic'; prompt: string; seq: string[]; answer: string }
  | { kind: 'chengyu'; pinyin: string; meaning: string; example?: string }
  | { kind: 'phrase'; zh: string }
  | { kind: 'unknown' }

export function parseKpPayload(raw: string): KpContent {
  if (!raw) return { kind: 'unknown' }
  try {
    const p = JSON.parse(raw) as Record<string, unknown>
    switch (p.kind) {
      case 'poem': {
        const lines = Array.isArray(p.lines)
          ? p.lines.map(String).filter(Boolean)
          : [String(p.line1 ?? ''), String(p.line2 ?? '')].filter(Boolean)
        return {
          kind: 'poem',
          author: String(p.author ?? ''),
          lines,
        }
      }
      case 'fact':
        return {
          kind: 'fact',
          q: String(p.q ?? ''),
          a: String(p.a ?? ''),
          emoji: p.emoji ? String(p.emoji) : undefined,
        }
      case 'pattern':
      case 'classify':
      case 'order':
      case 'diff':
        return {
          kind: 'logic',
          prompt: String(p.prompt ?? ''),
          seq: Array.isArray(p.seq) ? p.seq.map(String) : [],
          answer: String(p.a ?? ''),
        }
      case 'chengyu':
        return {
          kind: 'chengyu',
          pinyin: String(p.pinyin ?? ''),
          meaning: String(p.meaning ?? ''),
          example: p.example ? String(p.example) : undefined,
        }
      case 'phrase':
        return {
          kind: 'phrase',
          zh: String(p.zh ?? ''),
        }
      default:
        return { kind: 'unknown' }
    }
  } catch {
    return { kind: 'unknown' }
  }
}
