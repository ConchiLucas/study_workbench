import { speechAudioURL } from '../../api/literacy'
import type { LiteracyChar } from '../../api/literacyTypes'
import type { QTaskItem } from '../../api/qtaskTypes'
import type {
  LiteracyQuizOption,
  LiteracyQuizOptionKind,
  LiteracyQuizQuestion,
  LiteracyQuizType,
} from '../literacy/quizPreview'

const CODE_META: Record<
  string,
  {
    type: LiteracyQuizType
    difficulty: 'medium' | 'easy'
    title: string
    stemKind: LiteracyQuizQuestion['stemKind']
    optionKind: LiteracyQuizOptionKind
  }
> = {
  glyph_sense: {
    type: 'glyph_sense',
    difficulty: 'easy',
    title: '看字图选义图',
    stemKind: 'glyph',
    optionKind: 'sense',
  },
  sense_char: {
    type: 'sense_char',
    difficulty: 'easy',
    title: '看义图选字',
    stemKind: 'sense',
    optionKind: 'char',
  },
}

export function isLiteracyPackQuestionCode(code: string): boolean {
  return code === 'glyph_sense' || code === 'sense_char'
}

function stubChar(partial: {
  kpId: number
  charText: string
  glyphImageUrl?: string
  senseImageUrl?: string
  speechAudioUrl?: string
}): LiteracyChar {
  return {
    kpId: partial.kpId,
    charText: partial.charText,
    moduleCode: '',
    moduleName: '',
    moduleOrder: 0,
    kpOrder: 0,
    needsSenseImage: Boolean(partial.senseImageUrl),
    needsSenseImageOverride: null,
    effectiveNeedsSenseImage: Boolean(partial.senseImageUrl),
    glyphImageUrl: partial.glyphImageUrl ?? '',
    senseImageUrl: partial.senseImageUrl ?? '',
    speechAudioUrl: partial.speechAudioUrl ?? '',
  }
}

function optionImage(kind: LiteracyQuizOptionKind, char: LiteracyChar | undefined): string | undefined {
  if (!char) return undefined
  if (kind === 'glyph') return char.glyphImageUrl || undefined
  if (kind === 'sense') return char.senseImageUrl || undefined
  return char.glyphImageUrl || undefined
}

function optionLabel(opt: { label?: string } | string | undefined): string {
  if (typeof opt === 'string') return opt
  if (opt && typeof opt.label === 'string') return opt.label
  return ''
}

/** Map a stored task item into the literacy quiz-list row shape. */
export function qtaskItemToQuizQuestion(
  item: QTaskItem,
  byKpId: Map<number, LiteracyChar>,
  byChar: Map<string, LiteracyChar>,
): LiteracyQuizQuestion | null {
  const meta = CODE_META[item.code]
  if (!meta) return null

  const targetAsset = byKpId.get(item.kpId)
  const target = stubChar({
    kpId: item.kpId,
    charText: item.charText || targetAsset?.charText || '',
    glyphImageUrl: targetAsset?.glyphImageUrl,
    senseImageUrl: targetAsset?.senseImageUrl,
    speechAudioUrl: targetAsset?.speechAudioUrl,
  })

  const rawOptions = Array.isArray(item.options) ? item.options : []
  const options: LiteracyQuizOption[] = rawOptions.map((opt, i) => {
    const label = optionLabel(opt)
    const asset =
      i === item.answerIndex
        ? targetAsset ?? byChar.get(label)
        : byChar.get(label)
    const kpId = asset?.kpId ?? (i === item.answerIndex ? item.kpId : -1000 - i)
    return {
      kpId,
      charText: label || asset?.charText || '?',
      kind: meta.optionKind,
      imageUrl: optionImage(meta.optionKind, asset),
      speechUrl: speechAudioURL(kpId > 0 ? kpId : item.kpId, asset?.speechAudioUrl),
      correct: i === item.answerIndex,
    }
  })

  return {
    id: `qtask-${item.questionId}-${item.seq}`,
    type: meta.type,
    difficulty: meta.difficulty,
    title: meta.title,
    target,
    stemKind: meta.stemKind,
    optionKind: meta.optionKind,
    speechUrl: speechAudioURL(target.kpId, target.speechAudioUrl || undefined),
    options,
    available: options.length > 0,
    unavailableReason: options.length > 0 ? undefined : '无选项',
  }
}
