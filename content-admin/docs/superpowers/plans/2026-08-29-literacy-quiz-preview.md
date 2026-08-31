# Literacy Quiz Preview Implementation Plan

> **For agentic workers:** Implement task-by-task. Checkboxes track progress.

**Goal:** Show three quiz types per literacy group (listen→glyph, glyph→sense, sense→char) in a 3-column layout, with a kid-style effect preview page; options randomized within the group; no DB writes.

**Architecture:** Pure frontend. Build preview questions from existing group `LiteracyChar` assets. List page shows per-char question rows; `/literacy/preview/:moduleCode` is the interactive effect shell.

**Tech Stack:** React + TypeScript + existing Vite SPA; reuse literacy list/speech URLs.

---

### Task 1: Quiz builder helper + tests
- [x] `quizPreview.ts` — build questions with in-group distractors

### Task 2: Three-column question row UI + group section on LiteracyPage
- [x] `QuizQuestionRow.tsx` + `GroupQuizList` on LiteracyPage + 「查看效果」

### Task 3: Preview effect page + route + styles
- [x] `LiteracyPreviewPage.tsx`, route, CSS

### Task 4: Wire「查看效果」and smoke-check in browser
- [x] Build passes; Docker rebuild
