# The Zoo Treasure Hunt Dialogue Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the existing narrative with a seven-scene bilingual dialogue story whose English lines contain no more than five words while covering all 63 target words.

**Architecture:** Keep one Markdown learning artifact. Format every English line as `**Character:** dialogue`, place its Chinese translation directly below, and retain the authoritative vocabulary checklist. Use a read-only Python validation to parse dialogue lines, count lexical words after the character label, verify translation adjacency, and check target-word coverage.

**Tech Stack:** Markdown, Python 3 read-only validation

---

### Task 1: Rewrite the story as short dialogue

**Files:**
- Modify: `小学英语/3年级上册/The Zoo Treasure Hunt.md`
- Reference: `docs/superpowers/specs/2026-08-03-zoo-treasure-hunt-dialogue-story-design.md`

- [x] **Step 1: Replace the narrative body**

Write exactly seven scene headings:

```text
1. Getting Ready
2. At the Zoo
3. Animal Friends
4. Color Clues
5. Number Clues
6. Body Clues
7. Picnic Treasure
```

Under each scene, write only character dialogue lines in this format:

```markdown
**Mia:** My **bag** is ready.

> 米娅：我的书包准备好了。
```

The character label is excluded from the word count. Every dialogue line must contain at most five English word tokens after Markdown formatting is removed, with approximately three words preferred.

- [x] **Step 2: Cover the target vocabulary**

Use all of these words in meaningful dialogue and bold each target occurrence:

```text
arm, bag, bear, bird, black, blue, body, book, bread, brother, brown, cake, cat, crayon, dog, duck, ear, egg, eight, elephant, eraser, eye, face, fish, five, foot, four, funny, green, hand, head, juice, leg, milk, monkey, mouth, mum, nine, no, nose, OK, one, orange, panda, pen, pencil, pig, plate, red, rice, ruler, school, seven, six, ten, three, tiger, two, water, white, yellow, your, zoo
```

- [x] **Step 3: Retain the checklist**

Keep a 63-row table after the dialogue. Its numbers must be exactly 1 through 63 and its English words must match the authoritative list in order.

### Task 2: Validate the rewritten artifact

**Files:**
- Verify: `小学英语/3年级上册/The Zoo Treasure Hunt.md`

- [x] **Step 1: Parse dialogue structure**

Use Python 3 to read the Markdown and assert:

- exactly seven `## Scene` headings occur before the checklist;
- every dialogue line matches `**Character:** dialogue`;
- every dialogue line is followed by a non-empty Chinese blockquote;
- no narrative English paragraph remains between scenes.

- [x] **Step 2: Enforce line length**

Strip Markdown markers from each dialogue, extract English tokens with the regular expression `[A-Za-z]+(?:'[A-Za-z]+)?`, and assert that every line has between one and five tokens. Print the maximum and average line lengths.

Expected conditions:

```text
over_limit=[]
max_words<=5
average_words close to 3
```

- [x] **Step 3: Verify vocabulary coverage and checklist**

Compare bolded tokens in the dialogue body with the 63-word target set. Parse the checklist rows and compare both their numbering and word order with the authoritative list.

Expected output:

```text
target=63
covered=63
missing=[]
checklist=63
```

- [x] **Step 4: Review story quality**

Read all seven scenes in order and confirm they form one continuous treasure-hunt story, every Chinese translation matches the adjacent English dialogue, and target words appear in understandable contexts.

- [x] **Step 5: Report the verified artifact**

Return the absolute clickable file path and the exact dialogue count, maximum line length, average line length, and target coverage. This directory is not a Git repository, so no commit or branch integration applies.
