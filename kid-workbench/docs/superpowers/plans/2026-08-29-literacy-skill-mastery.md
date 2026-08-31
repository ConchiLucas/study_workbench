# Literacy Skill Mastery + Matrix UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** On `/subjects/literacy`, each group shows per-character × 3 question-type mastery; a character is fully mastered only when all three skills are mastered/review_due; each group has a 「题目」entry to fullscreen three-column quiz preview.

**Architecture:** Add `mastery_skills` table `(child_id, kp_id, skill_code)` for literacy skills `listen_glyph` | `glyph_sense` | `sense_char`. Attempts resolve skill via `questions.code`, Apply updates skill row, then roll up to existing `mastery_states`. Matrix API adds `skills[]` per point; literacy FE renders three dots + group skill counts + fullscreen preview.

**Tech Stack:** Go (Gin/GORM), React + TS, existing mastery engine, Vite parent-dashboard.

---

## File map

| File | Responsibility |
|------|----------------|
| `backend/internal/db/migrations/*/005_mastery_skills.sql` | New table |
| `backend/internal/model/model.go` | `MasterySkill` model |
| `backend/internal/mastery/skills.go` | Skill codes + rollup rules |
| `backend/internal/repo/repo.go` | Upsert/get skills |
| `backend/internal/service/attempt.go` | Apply skill then rollup |
| `backend/internal/service/dashboard.go` | Matrix `skills[]`, aggregate mastered |
| `backend/internal/quiz/literacy.go` | Seed 3 codes (text/image-capable options) |
| `frontend/src/api/types.ts` | `MatrixSkill`, extend `MatrixPoint` |
| `frontend/src/components/mastery/MasteryCell.tsx` | Literacy 3-dot card |
| `frontend/src/components/mastery/MasteryMatrix.tsx` | Group skill counts + 「题目」 |
| `frontend/src/components/mastery/LiteracyQuizSheet.tsx` | Fullscreen list/detail 3-col |

---

### Task 1: Skill codes + rollup (TDD) ✅
### Task 2: Migration + model + repo ✅
### Task 3: Attempt path applies skill mastery ✅
### Task 4: Matrix API returns skills[] ✅
### Task 5: Literacy quiz specs for 3 codes ✅
### Task 6: FE types + MasteryCell 3 dots ✅
### Task 7: Matrix group header + 「题目」sheet ✅
### Task 8: Wire Docker / smoke ✅
