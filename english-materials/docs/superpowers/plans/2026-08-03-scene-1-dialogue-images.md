# Scene 1 Dialogue Images Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Generate and save 12 consistent 3D children’s picture-book images for Scene 1, with one exact English dialogue subtitle embedded in each image.

**Architecture:** Use the built-in image generation tool once per dialogue line. Repeat a fixed character, prop, home-environment, composition, and subtitle specification in every prompt, while changing only the required action and verbatim subtitle. Copy every accepted generated image into the project directory and visually inspect each saved file.

**Tech Stack:** Built-in ImageGen, PNG raster images, local image inspection

---

### Task 1: Prepare the output directory and prompt contract

**Files:**
- Create directory: `小学英语/3年级上册/Scene 1 - Getting Ready/`
- Reference: `docs/superpowers/specs/2026-08-03-scene-1-dialogue-images-design.md`

- [x] **Step 1: Create and verify the output directory**

Run:

```bash
mkdir -p '小学英语/3年级上册/Scene 1 - Getting Ready'
test -d '小学英语/3年级上册/Scene 1 - Getting Ready'
```

Expected: exit code 0.

- [x] **Step 2: Apply the shared prompt contract to every image**

Every prompt must state:

- illustration-story use case for a Grade 3 English learning card;
- polished warm 3D children’s animation picture-book style;
- a single 16:9 landscape scene, not a collage or split panel;
- the fixed Mia, Ben, Mum designs from the approved specification;
- the fixed yellow backpack and colored school supplies;
- the same warm home environment and afternoon lighting;
- one dark-blue translucent rounded subtitle panel inside the bottom of the image;
- one exact white bold sans-serif English subtitle;
- no Chinese, translation, watermark, logo, or other text.

### Task 2: Generate the 12 dialogue images

**Files:**
- Create: `小学英语/3年级上册/Scene 1 - Getting Ready/01-school-is-over.png`
- Create: `小学英语/3年级上册/Scene 1 - Getting Ready/02-hi-mum.png`
- Create: `小学英语/3年级上册/Scene 1 - Getting Ready/03-pack-your-bag.png`
- Create: `小学英语/3年级上册/Scene 1 - Getting Ready/04-heres-my-book.png`
- Create: `小学英语/3年级上册/Scene 1 - Getting Ready/05-take-this-pen.png`
- Create: `小学英语/3年级上册/Scene 1 - Getting Ready/06-and-my-pencil.png`
- Create: `小学英语/3年级上册/Scene 1 - Getting Ready/07-bring-the-ruler.png`
- Create: `小学英语/3年级上册/Scene 1 - Getting Ready/08-i-need-an-eraser.png`
- Create: `小学英语/3年级上册/Scene 1 - Getting Ready/09-take-one-crayon.png`
- Create: `小学英语/3年级上册/Scene 1 - Getting Ready/10-im-your-brother.png`
- Create: `小学英语/3年级上册/Scene 1 - Getting Ready/11-i-can-help.png`
- Create: `小学英语/3年级上册/Scene 1 - Getting Ready/12-ok-lets-go.png`

- [x] **Step 1: Generate images 01–03**

Use these exact subtitle/action pairs:

1. `Mum: School is over.` — Mum greets Mia in the entryway after school; Mia still wears the yellow backpack.
2. `Mia: Hi, mum!` — Mia waves happily to Mum in the same entryway.
3. `Mum: Pack your bag.` — Mum points to the open yellow backpack beside the child’s desk.

- [x] **Step 2: Generate images 04–06**

4. `Mia: Here's my book.` — Mia holds the blue book above the open yellow backpack.
5. `Ben: Take this pen.` — Ben gives the red pen to Mia.
6. `Mia: And my pencil.` — Mia holds the yellow pencil; the book and pen are visible in the backpack.

- [x] **Step 3: Generate images 07–09**

7. `Ben: Bring the ruler.` — Ben points to the light-blue ruler on the desk.
8. `Mia: I need an eraser.` — Mia finds the pink eraser on the desk.
9. `Mum: Take one crayon.` — Mum hands Mia one green crayon from a crayon box.

- [x] **Step 4: Generate images 10–12**

10. `Ben: I'm your brother.` — Ben points to himself while Mia looks up at him warmly.
11. `Ben: I can help.` — Ben closes the blue zipper on Mia’s yellow backpack while Mia holds it.
12. `Mia: OK, let's go!` — Mia wears the packed backpack and walks toward the open door with Ben and Mum.

For every generated result, copy or move the accepted PNG from the built-in generation location into the exact project path listed above.

### Task 3: Inspect and validate the image set

**Files:**
- Verify: all PNG files under `小学英语/3年级上册/Scene 1 - Getting Ready/`

- [x] **Step 1: Verify file count and formats**

Run:

```bash
find '小学英语/3年级上册/Scene 1 - Getting Ready' -maxdepth 1 -type f -name '*.png' | sort
```

Expected: exactly the 12 numbered PNG files specified in Task 2.

- [x] **Step 2: Inspect every image visually**

Open all 12 files and check:

- the subtitle is inside the image at the bottom;
- the subtitle exactly matches its approved English text;
- there is no Chinese or unrelated text;
- the action and props match the dialogue;
- the image is one landscape scene, not a collage;
- character clothing and major features remain recognizably consistent.

- [x] **Step 3: Regenerate failures individually**

If an image has incorrect subtitle text, Chinese text, an unrelated scene, or a materially inconsistent character, regenerate only that image with the same shared prompt contract and a stronger correction emphasizing the failed constraint. Keep accepted images unchanged.

- [x] **Step 4: Report final paths and prompts**

Return the output directory, the 12 filenames, the shared prompt contract, and note that the built-in ImageGen path was used. The workspace is not a Git repository, so no commit or branch integration applies.
