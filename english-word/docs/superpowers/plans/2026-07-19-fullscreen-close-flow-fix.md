# Difficulty Picker Parent Close Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the difficulty picker back to the difficulty-training setup with one click, then close the training setup back to Home with a separate click.

**Architecture:** Keep the two explicit Home states and two independent close functions. Place the picker overlay above the shared fixed close button so browser hit testing reaches the picker control, then have that control clear only `showDifficultyPicker`.

**Tech Stack:** Vue 3, TypeScript, Vitest, Vite

---

### Task 1: Lock the parent-close behavior

**Files:**
- Modify: `rob_english_word_front/src/views/fullscreenNavigation.contract.test.ts`
- Create: `rob_english_word_front/src/views/HomeView.close-flow.test.ts`

- [x] Assert the training setup close binds `closeTrainingSetup()`.
- [x] Assert the picker close binds `closeDifficultyPicker()` and does not clear `showTrainingSetup`.
- [x] Mount `HomeView` and verify picker close and Escape return to the training setup.
- [x] Verify selecting a concrete difficulty also returns to the training setup.
- [x] Run the focused tests and observe the old one-click-Home implementation fail in three places.

### Task 2: Keep browser hit testing on the top close button

**Files:**
- Modify: `rob_english_word_front/src/views/HomeView.vue`
- Modify: `rob_english_word_front/src/views/fullscreenNavigation.contract.test.ts`

- [x] Reproduce in the logged-in 7002 browser that an overlay at z-index 90 leaves the underlying fixed close button at z-index 3000 on top.
- [x] Add a failing contract requiring the picker overlay z-index to exceed the shared close-button z-index.
- [x] Raise `.difficulty-overlay` to `z-index: 4000` and verify the contract passes.

### Task 3: Implement and verify the final close hierarchy

**Files:**
- Modify: `rob_english_word_front/src/views/HomeView.vue`
- Test: `rob_english_word_front/src/views/fullscreenNavigation.contract.test.ts`
- Test: `rob_english_word_front/src/views/HomeView.close-flow.test.ts`

- [x] Bind the picker close control to `closeDifficultyPicker()`.
- [x] Make Escape call `closeDifficultyPicker()` while the picker is visible.
- [x] Keep `closeTrainingSetup()` as the separate Home transition.
- [x] Run focused tests: 9/9 pass.
- [x] Reload the logged-in in-app browser and verify one click produces `home=false`, `training=true`, `picker=false`.
- [x] Run all frontend tests, the production build, and `git diff --check`.
