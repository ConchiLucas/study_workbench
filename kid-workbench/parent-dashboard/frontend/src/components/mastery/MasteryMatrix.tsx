import { useState } from 'react'
import { useMatrix } from '../../api/dashboard'
import type { MatrixModule, MatrixPoint } from '../../api/types'
import { useChildStore } from '../../store/childStore'
import { type MasteryStatus } from '../../theme'
import { LiteracyQuizSheet } from './LiteracyQuizSheet'
import { MasteryCell } from './MasteryCell'
import { MathQuizSheet } from './MathQuizSheet'
import { PinyinQuizSheet } from './PinyinQuizSheet'
import { StatusLegend } from './StatusLegend'

const LITERACY_SKILL_LABEL: Record<string, string> = {
  glyph_sense: '义',
  sense_char: '字',
}

const PINYIN_SKILL_LABEL: Record<string, string> = {
  inword: '例',
  listen: '音',
}

export function MasteryMatrix({ subject }: { subject: string }) {
  const childId = useChildStore((s) => s.childId)
  const { data, isLoading } = useMatrix(childId, subject)
  const [expanded, setExpanded] = useState<Record<string, boolean>>({})
  const [quizModule, setQuizModule] = useState<MatrixModule | null>(null)
  const literacy = subject === 'literacy'
  const pinyin = subject === 'pinyin'
  const skillSubject = literacy || pinyin
  const skillLabel = literacy ? LITERACY_SKILL_LABEL : PINYIN_SKILL_LABEL
  const skillCodes = literacy ? ['glyph_sense', 'sense_char'] : ['inword', 'listen']
  const showQuizButton = (mod: MatrixModule) =>
    skillSubject ||
    (subject === 'math' && (mod.code === 'add10' || mod.code === 'sub10' || mod.code === 'shape'))

  if (isLoading) return <div className="text-sm text-slate-400">加载中…</div>
  if (!data) return null

  const masteredLabel = skillSubject ? '完全掌握' : '已掌握'

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between flex-wrap gap-2">
        <h3 className="font-semibold text-slate-700">
          {data.subject.icon} {data.subject.name}
          <span className="ml-2 text-sm font-normal text-slate-400">
            {masteredLabel} {data.subject.counts.mastered + data.subject.counts.review_due}/{data.subject.total}
          </span>
        </h3>
        <StatusLegend />
      </div>
      {literacy ? (
        <p className="text-xs text-slate-400 -mt-2">
          字变绿 = 义 / 字 两种题型都过；两点颜色对应各题型状态
        </p>
      ) : null}
      {pinyin ? (
        <p className="text-xs text-slate-400 -mt-2">
          字母变绿 = 例字选音 / 听音选字母 两种题型都过；两点颜色对应各题型状态
        </p>
      ) : null}

      {data.modules.map((mod) => {
        const started = mod.total - mod.points.filter((p) => p.status === 'not_started').length
        const open = expanded[mod.code] ?? started > 0
        const skillCounts = skillSubject ? countModuleSkills(mod.points, skillCodes) : null
        return (
          <div key={mod.code} className="rounded-xl2 bg-white p-4 shadow-sm">
            <div className="flex w-full items-start justify-between gap-3">
              <button
                type="button"
                className="flex-1 text-left"
                onClick={() => setExpanded((e) => ({ ...e, [mod.code]: !open }))}
              >
                <span className="font-medium text-slate-600">
                  {mod.name}
                  <span className="ml-2 text-xs text-slate-400">
                    {skillSubject ? '完全掌握' : ''} {mod.mastered}/{mod.total}
                  </span>
                </span>
                {skillCounts ? (
                  <div className="mt-1 flex flex-wrap gap-2 text-[11px] text-slate-400">
                    {skillCounts.map((s) => (
                      <span key={s.code}>
                        {skillLabel[s.code] ?? s.code} {s.done}/{mod.total}
                      </span>
                    ))}
                  </div>
                ) : null}
              </button>
              <div className="flex items-center gap-2 shrink-0">
                {showQuizButton(mod) ? (
                  <button
                    type="button"
                    className="rounded-lg border border-slate-200 px-2.5 py-1 text-xs text-slate-600 hover:border-teal-300 hover:text-teal-700"
                    onClick={() => setQuizModule(mod)}
                  >
                    题目
                  </button>
                ) : null}
                <button
                  type="button"
                  className="text-xs text-slate-400"
                  onClick={() => setExpanded((e) => ({ ...e, [mod.code]: !open }))}
                >
                  {open ? '收起' : '展开'}
                </button>
              </div>
            </div>
            {open && (
              <div className="mt-3 flex flex-wrap gap-1.5">
                {mod.points.map((p) => (
                  <MasteryCell
                    key={p.id}
                    point={p}
                    skillMode={skillSubject}
                    skillCodes={skillCodes}
                    skillLabel={skillLabel}
                  />
                ))}
              </div>
            )}
          </div>
        )
      })}

      {quizModule && literacy ? (
        <LiteracyQuizSheet module={quizModule} onClose={() => setQuizModule(null)} />
      ) : null}
      {quizModule && pinyin ? (
        <PinyinQuizSheet module={quizModule} onClose={() => setQuizModule(null)} />
      ) : null}
      {quizModule &&
      subject === 'math' &&
      (quizModule.code === 'add10' || quizModule.code === 'sub10' || quizModule.code === 'shape') ? (
        <MathQuizSheet module={quizModule} onClose={() => setQuizModule(null)} />
      ) : null}
    </div>
  )
}

function countModuleSkills(points: MatrixPoint[], codes: string[]) {
  return codes.map((code) => {
    let done = 0
    for (const p of points) {
      const sk = p.skills?.find((s) => s.code === code)
      const st = (sk?.status ?? 'not_started') as MasteryStatus
      if (st === 'mastered' || st === 'review_due') done++
    }
    return { code, done }
  })
}
