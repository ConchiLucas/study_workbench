# Best Sentence Cloze TTS Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the localhost:7003 cloze practice use `word_clean_best_sentence.cloze_sentence` and `cloze_answer`, and play the complete MinIO TTS audio for that best sentence.

**Architecture:** The Java difficulty-task query will select one best-sentence row per `word_clean` row and snapshot its sentence, cloze answer, and TTS URL into `sentence_cloze_item`. The API will expose the snapshot audio URL, while the React client will play it with `HTMLAudioElement`; old records without an audio URL retain the existing browser-speech fallback.

**Tech Stack:** PostgreSQL, Java 21, Spring Boot 3, MyBatis Plus, JUnit 5, React 19, TypeScript, Vite.

---

### Task 1: Lock the best-sentence task mapping with a failing Java test

**Files:**
- Create: `rob_english_word_back/src/test/java/com/robword/service/ClozePracticeServiceTest.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/dto/ClozePracticeSentenceCandidate.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/service/ClozePracticeService.java`

- [ ] **Step 1: Write a failing test** that invokes difficulty-batch creation with a candidate whose base word is `value`, answer is `values`, cloze sentence is `The company ____ fairness.`, and audio URL is `/ai-file-navigation/word_clean_tts/value.mp3`.
- [ ] **Step 2: Run** `mvn -Dtest=ClozePracticeServiceTest test` from `rob_english_word_back` and confirm the test fails because the candidate/task fields are not mapped yet.
- [ ] **Step 3: Add the candidate fields** `bestSentenceId`, `clozeSentence`, `clozeAnswer`, and `ttsObjectUrl`, then make task creation store the actual cloze answer and supplied cloze sentence rather than rebuilding them from the base word.
- [ ] **Step 4: Re-run** `mvn -Dtest=ClozePracticeServiceTest test` and confirm it passes.

### Task 2: Select and snapshot best-sentence data

**Files:**
- Modify: `rob_english_word_back/src/main/java/com/robword/mapper/SentenceClozeItemMapper.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/entity/SentenceClozeItem.java`
- Modify: `rob_english_word_back/db/sentence_cloze_item.sql`

- [ ] **Step 1: Change the candidate query** to join `word_clean_best_sentence` by `word_clean_id`, require non-empty sentence/cloze/answer, and return the best-sentence ID, source model, and successful TTS URL.
- [ ] **Step 2: Add snapshot columns** `best_sentence_id bigint` and `sentence_audio_url text NOT NULL DEFAULT ''` to the idempotent DDL and entity.
- [ ] **Step 3: Keep old records compatible** by allowing a null best-sentence ID and an empty audio URL.
- [ ] **Step 4: Run the Java unit test** again to confirm mapping behavior remains green.

### Task 3: Expose and resolve sentence audio in the API/client

**Files:**
- Modify: `rob_english_word_back/src/main/java/com/robword/dto/ClozePracticeTaskResponse.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/service/ClozePracticeService.java`
- Create: `rob_english_word_cloze_web/src/lib/sentenceAudio.ts`
- Create: `rob_english_word_cloze_web/test/sentenceAudio.test.ts`
- Modify: `rob_english_word_cloze_web/src/types/cloze.ts`

- [ ] **Step 1: Write a failing Node test** asserting that a non-empty MinIO proxy URL is selected and an empty URL requests browser-speech fallback.
- [ ] **Step 2: Run** `node --test --experimental-strip-types test/sentenceAudio.test.ts` and confirm it fails because the resolver does not exist.
- [ ] **Step 3: Add `sentenceAudioUrl`** to the Java response and TypeScript task type, and implement the minimal audio-source resolver.
- [ ] **Step 4: Re-run the Node test** and confirm it passes.

### Task 4: Play MinIO audio from the 7003 interface

**Files:**
- Modify: `rob_english_word_cloze_web/src/App.tsx`
- Modify: `rob_english_word_cloze_web/vite.config.ts`

- [ ] **Step 1: Replace the sentence speech ref** with an `HTMLAudioElement` ref while retaining browser speech only as a fallback for old tasks without `sentenceAudioUrl`.
- [ ] **Step 2: Implement play/stop/end/error handling** so the existing button state remains accurate and a playback failure displays a toast.
- [ ] **Step 3: Proxy `/ai-file-navigation`** from Vite port 7003 to the Go file endpoint on port 8009.
- [ ] **Step 4: Run** `npm run build` and confirm TypeScript and Vite finish successfully.

### Task 5: Apply and verify the database/API/browser path

**Files:**
- Modify through SQL execution: `public.sentence_cloze_item`

- [ ] **Step 1: Apply only the two idempotent `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` statements** to the local PostgreSQL database.
- [ ] **Step 2: Query candidate counts** and confirm best-sentence cloze data and successful TTS URLs are available for difficulty practice.
- [ ] **Step 3: Run** `mvn test` and `npm run build` as fresh full verification.
- [ ] **Step 4: Open localhost:7003**, create a difficulty batch, confirm the displayed blank and accepted answer come from the best-sentence row, and confirm “朗读句子” requests `/ai-file-navigation/...` and plays the full sentence before submission.
