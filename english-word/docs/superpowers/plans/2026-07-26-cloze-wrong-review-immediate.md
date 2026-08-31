# Cloze Wrong Review Immediate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make a wrong cloze answer immediately eligible for the due-review queue while preserving the existing 7-day and 15-day correct-answer intervals.

**Architecture:** Keep the existing review schedule table, mapper upsert, and due query. Change only the timestamp supplied by `ClozePracticeService` for a wrong answer, and protect the behavior with a service-level Mockito test that captures the scheduled time.

**Tech Stack:** Java 21, Spring Boot 3.2, JUnit 5, Mockito, Maven

---

### Task 1: Schedule wrong answers immediately

**Files:**
- Modify: `rob_english_word_back/src/test/java/com/robword/service/ClozePracticeServiceTest.java`
- Modify: `rob_english_word_back/src/main/java/com/robword/service/ClozePracticeService.java:349`

- [ ] **Step 1: Write the failing test**

Add the required imports and a test that submits a wrong answer, captures the timestamp passed to `upsertWrongSchedule`, and requires it to fall inside the service call:

```java
import com.robword.dto.ClozePracticeAnswerRequest;
import com.robword.entity.SentenceClozeAnswerRecord;
import org.mockito.ArgumentCaptor;

import java.time.LocalDateTime;

import static org.junit.jupiter.api.Assertions.assertFalse;

@Test
void shouldScheduleWrongAnswerForImmediateReview() {
    SentenceClozeItem item = practiceItem(92L, "value");
    when(sentenceClozeItemMapper.selectById(92L)).thenReturn(item);
    when(answerRecordMapper.selectCount(any())).thenReturn(0L);
    doAnswer(invocation -> {
        SentenceClozeAnswerRecord record = invocation.getArgument(0);
        record.setId(301L);
        return 1;
    }).when(answerRecordMapper).insert(any(SentenceClozeAnswerRecord.class));

    ClozePracticeAnswerRequest request = new ClozePracticeAnswerRequest();
    request.setClozeItemId(92L);
    request.setAnswers(List.of("wrong"));

    LocalDateTime before = LocalDateTime.now();
    var response = service.submitAnswer(7L, request);
    LocalDateTime after = LocalDateTime.now();

    ArgumentCaptor<LocalDateTime> reviewTime = ArgumentCaptor.forClass(LocalDateTime.class);
    verify(reviewScheduleMapper).upsertWrongSchedule(eq(7L), eq(92L), eq(301L), reviewTime.capture());
    assertFalse(response.getCorrect());
    assertFalse(reviewTime.getValue().isBefore(before));
    assertFalse(reviewTime.getValue().isAfter(after));
}
```

- [ ] **Step 2: Run the test and verify RED**

Run:

```bash
cd rob_english_word_back
mvn -Dtest=ClozePracticeServiceTest#shouldScheduleWrongAnswerForImmediateReview test
```

Expected: FAIL because the captured review time is one day later than `after`.

- [ ] **Step 3: Implement the minimal change**

Change the wrong-answer schedule call to:

```java
reviewScheduleMapper.upsertWrongSchedule(userId, item.getId(), record.getId(), LocalDateTime.now());
```

Do not modify `advanceReviewScheduleOnCorrect`; its 7-day, 15-day, and mastered behavior remains unchanged.

- [ ] **Step 4: Run the focused test and verify GREEN**

Run:

```bash
cd rob_english_word_back
mvn -Dtest=ClozePracticeServiceTest#shouldScheduleWrongAnswerForImmediateReview test
```

Expected: BUILD SUCCESS with the focused test passing.

- [ ] **Step 5: Run backend regression tests**

Run:

```bash
cd rob_english_word_back
mvn test
```

Expected: BUILD SUCCESS with all backend tests passing.

- [ ] **Step 6: Review the diff**

Run:

```bash
git diff --check
git diff -- rob_english_word_back/src/test/java/com/robword/service/ClozePracticeServiceTest.java rob_english_word_back/src/main/java/com/robword/service/ClozePracticeService.java
```

Expected: no whitespace errors; only the immediate scheduling behavior and its regression test change.
