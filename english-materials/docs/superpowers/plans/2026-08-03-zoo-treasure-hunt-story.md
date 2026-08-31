# The Zoo Treasure Hunt Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create a bilingual Markdown adventure story that naturally uses and visibly marks all 63 Grade 3 first-semester target words.

**Architecture:** Use one self-contained Markdown artifact with five short story chapters, an English paragraph followed by its Chinese translation, and a final vocabulary checklist. Verify target-word coverage mechanically against the authoritative 63-word list and inspect the rendered Markdown structure manually.

**Tech Stack:** Markdown, shell, Python 3 read-only validation

---

### Task 1: Write the bilingual story

**Files:**
- Create: `小学英语/3年级上册/The Zoo Treasure Hunt.md`
- Reference: `docs/superpowers/specs/2026-08-03-zoo-treasure-hunt-story-design.md`

- [x] **Step 1: Confirm the destination directory**

Run:

```bash
test -d '小学英语/3年级上册'
```

Expected: exit code 0.

- [x] **Step 2: Create the story artifact**

Write five numbered chapters with these fixed story beats:

1. The hero packs a bag, book, pen, pencil, ruler, eraser, and crayon, then leaves with Mum and a brother.
2. At the zoo, animal friends introduce the treasure hunt.
3. The hero solves number and color clues using the target number and color words.
4. The hero solves a body-part clue and shares the target food and drink words.
5. The hero finds the treasure and celebrates with the family and animals.

Every English paragraph must be followed immediately by a Chinese translation block. Every target word must appear at least once in English and use Markdown bold syntax such as `**panda**`.

- [x] **Step 3: Add the vocabulary checklist**

Add a final section containing a Markdown table numbered 1 through 63. Each row must include the English target word and its Chinese meaning. The checklist must use this authoritative order:

```text
arm, bag, bear, bird, black, blue, body, book, bread, brother, brown, cake, cat, crayon, dog, duck, ear, egg, eight, elephant, eraser, eye, face, fish, five, foot, four, funny, green, hand, head, juice, leg, milk, monkey, mouth, mum, nine, no, nose, OK, one, orange, panda, pen, pencil, pig, plate, red, rice, ruler, school, seven, six, ten, three, tiger, two, water, white, yellow, your, zoo
```

### Task 2: Verify content and vocabulary coverage

**Files:**
- Verify: `小学英语/3年级上册/The Zoo Treasure Hunt.md`

- [x] **Step 1: Check Markdown structure**

Run:

```bash
rg -n '^#|^##|^###|^> \*\*中文：\*\*|^\| [0-9]+ \|' '小学英语/3年级上册/The Zoo Treasure Hunt.md'
```

Expected: one title, five numbered story chapters, a Chinese translation after each English section, and 63 numbered checklist rows.

- [x] **Step 2: Check all target words are bolded in the story body**

Run a Python 3 validation that reads the file only up to the vocabulary-checklist heading, extracts every `**...**` token case-insensitively, and compares the result with the authoritative 63-word set. Expected output:

```text
target=63
covered=63
missing=[]
```

- [x] **Step 3: Review readability**

Read the final file from start to finish and confirm:

- sentences are short enough for a Grade 3 learner;
- Chinese translations match the English meaning;
- the adventure has a clear beginning, problem, clues, solution, and warm ending;
- target words are used in meaningful contexts rather than as an isolated list.

- [x] **Step 4: Report completion**

Return the absolute clickable path to the story and report the mechanical coverage result. This workspace is not a Git repository, so no commit step is available.
