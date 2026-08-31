package com.robword.service;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.robword.dto.ClozePracticeAnswerRequest;
import com.robword.dto.ClozePracticeDifficultyBatchRequest;
import com.robword.dto.ClozePracticePreferenceResponse;
import com.robword.dto.ClozePracticeSentenceCandidate;
import com.robword.dto.ClozePracticeStatsResponse;
import com.robword.dto.ClozePracticeTaskResponse;
import com.robword.dto.ClozeWrongSentenceAttempt;
import com.robword.dto.ClozeWrongSentenceDetail;
import com.robword.dto.ClozeWrongSentenceItem;
import com.robword.dto.ClozeWrongSentencePageResponse;
import com.robword.dto.UpdateSoloDifficultyRequest;
import com.robword.entity.SentenceClozeAnswerRecord;
import com.robword.entity.SentenceClozeItem;
import com.robword.entity.User;
import com.robword.entity.Word;
import com.robword.entity.WordCleanTts;
import com.robword.mapper.SentenceClozeAnswerRecordMapper;
import com.robword.mapper.SentenceClozeItemMapper;
import com.robword.mapper.SentenceClozeReviewScheduleMapper;
import com.robword.mapper.ClozeWrongSentenceQueryMapper;
import com.robword.mapper.UserMapper;
import com.robword.mapper.WordMapper;
import com.robword.mapper.WordCleanTtsMapper;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.ArgumentCaptor;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.lang.reflect.Field;
import java.time.LocalDateTime;
import java.util.List;

import static org.junit.jupiter.api.Assertions.assertDoesNotThrow;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.argThat;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.doAnswer;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class ClozePracticeServiceTest {

    @Mock
    private SentenceClozeItemMapper sentenceClozeItemMapper;
    @Mock
    private SentenceClozeAnswerRecordMapper answerRecordMapper;
    @Mock
    private SentenceClozeReviewScheduleMapper reviewScheduleMapper;
    @Mock
    private UserMapper userMapper;
    @Mock
    private WordMapper wordMapper;
    @Mock
    private WordCleanTtsMapper wordCleanTtsMapper;
    @Mock
    private WrongWordAgentNotificationService wrongWordAgentNotificationService;
    @Mock
    private UserWordMasteryService userWordMasteryService;
    @Mock
    private WrongWordReviewProgressService wrongWordReviewProgressService;
    @Mock
    private ClozeWrongSentenceQueryMapper wrongSentenceQueryMapper;

    private ClozePracticeService service;

    @BeforeEach
    void setUp() {
        service = new ClozePracticeService(
                sentenceClozeItemMapper,
                answerRecordMapper,
                reviewScheduleMapper,
                userMapper,
                wordMapper,
                wordCleanTtsMapper,
                new ObjectMapper(),
                wrongWordAgentNotificationService,
                userWordMasteryService,
                wrongWordReviewProgressService,
                wrongSentenceQueryMapper
        );
    }

    @Test
    void shouldCreateDifficultyTaskFromBestSentenceClozeAndAudio() throws Exception {
        ClozePracticeSentenceCandidate candidate = new ClozePracticeSentenceCandidate();
        candidate.setWordCleanId(11L);
        candidate.setWord("value");
        candidate.setMeaning("重视；价值");
        candidate.setSentence("The company values fairness.");
        candidate.setSentenceTranslation("该公司重视公平。");
        candidate.setModelName("best-model");
        setRequiredField(candidate, "bestSentenceId", 21L);
        setRequiredField(candidate, "clozeSentence", "The company ____ fairness.");
        setRequiredField(candidate, "clozeAnswer", "values");
        setRequiredField(candidate, "ttsObjectUrl", "/ai-file-navigation/word_clean_tts/value.mp3");

        when(sentenceClozeItemMapper.selectDifficultyCandidates(eq(7L), eq(List.of(9)), eq(1)))
                .thenReturn(List.of(candidate));
        doAnswer(invocation -> {
            SentenceClozeItem item = invocation.getArgument(0);
            item.setId(99L);
            return 1;
        }).when(sentenceClozeItemMapper).insert(any(SentenceClozeItem.class));

        ClozePracticeDifficultyBatchRequest request = new ClozePracticeDifficultyBatchRequest();
        request.setDifficultyGroup("junior");
        request.setDifficultyLevel("junior_7_1");
        request.setLimit(1);

        List<ClozePracticeTaskResponse> tasks = service.createDifficultyBatch(7L, request);

        assertEquals(1, tasks.size());
        ClozePracticeTaskResponse task = tasks.getFirst();
        assertEquals("value", task.getWord());
        assertEquals("The company values fairness.", task.getSentence());
        assertEquals("The company ____ fairness.", task.getClozeSentence());
        assertEquals(List.of(6), task.getBlankLengths());
        assertEquals("/ai-file-navigation/word_clean_tts/value.mp3", requiredGetter(task, "getSentenceAudioUrl"));
    }

    @Test
    void shouldRepairInvalidStoredSoloDifficultyToJuniorDefault() {
        User user = new User();
        user.setId(7L);
        user.setSoloDifficultyGroup("rank");
        user.setSoloDifficultyLevel("rank_current");
        when(userMapper.selectById(7L)).thenReturn(user);

        ClozePracticePreferenceResponse response = service.getPreferences(7L);

        assertEquals("junior", response.getSoloDifficultyGroup());
        assertEquals("junior", response.getSoloDifficultyLevel());
        verify(userMapper).updateById(argThat(updated ->
                updated.getId().equals(7L)
                        && "junior".equals(updated.getSoloDifficultyGroup())
                        && "junior".equals(updated.getSoloDifficultyLevel())));
    }

    @Test
    void shouldSaveValidSoloDifficultyAndRejectRankDifficulty() {
        User user = new User();
        user.setId(7L);
        when(userMapper.selectById(7L)).thenReturn(user);

        UpdateSoloDifficultyRequest request = new UpdateSoloDifficultyRequest();
        request.setDifficultyGroup("junior");
        request.setDifficultyLevel("junior_7_1");

        ClozePracticePreferenceResponse response = service.updateSoloDifficulty(7L, request);

        assertEquals("junior", response.getSoloDifficultyGroup());
        assertEquals("junior_7_1", response.getSoloDifficultyLevel());
        verify(userMapper).updateById(argThat(updated ->
                updated.getId().equals(7L)
                        && "junior".equals(updated.getSoloDifficultyGroup())
                        && "junior_7_1".equals(updated.getSoloDifficultyLevel())));

        UpdateSoloDifficultyRequest rankRequest = new UpdateSoloDifficultyRequest();
        rankRequest.setDifficultyGroup("rank");
        rankRequest.setDifficultyLevel("rank_current");
        assertThrows(IllegalArgumentException.class, () -> service.updateSoloDifficulty(7L, rankRequest));
    }

    @Test
    void shouldLoadOnlyDueWrongReviewTasks() {
        SentenceClozeItem item = new SentenceClozeItem();
        item.setId(88L);
        item.setUserId(7L);
        item.setWord("values");
        item.setSentence("The company values fairness.");
        item.setClozeSentence("The company ____ fairness.");
        item.setBlankWordsJson("[\"values\"]");
        when(sentenceClozeItemMapper.selectDueWrongReviewItems(7L, 10)).thenReturn(List.of(item));

        List<ClozePracticeTaskResponse> tasks = service.getDueReviewTasks(7L, 10);

        assertEquals(1, tasks.size());
        assertEquals(88L, tasks.getFirst().getId());
        verify(sentenceClozeItemMapper).selectDueWrongReviewItems(7L, 10);
    }

    @Test
    void shouldExposeExactWrongSentenceAndDueTaskCountsInStats() {
        when(sentenceClozeItemMapper.selectCount(any())).thenReturn(30L);
        when(answerRecordMapper.countCompletedItems(7L)).thenReturn(8L);
        when(answerRecordMapper.selectCount(any())).thenReturn(12L, 7L, 5L);
        when(sentenceClozeItemMapper.countActiveWrongSentences(7L)).thenReturn(18L);
        when(sentenceClozeItemMapper.countDueReviewItems(7L)).thenReturn(13L);

        ClozePracticeStatsResponse response = service.getStats(7L);

        assertEquals(18L, response.getActiveWrongSentences());
        assertEquals(13L, response.getDueReviewTasks());
        verify(sentenceClozeItemMapper).countActiveWrongSentences(7L);
        verify(sentenceClozeItemMapper).countDueReviewItems(7L);
        verify(sentenceClozeItemMapper, never()).selectDueWrongReviewItems(eq(7L), any());
    }

    @Test
    void shouldNormalizeWrongSentenceFiltersAndDecodeSnapshots() {
        ClozeWrongSentenceItem item = new ClozeWrongSentenceItem();
        item.setProgressId(81L);
        item.setClozeItemId(91L);
        item.setTargetWordsJson("[\"raw\",\"momentum\"]");
        item.setWrongBlankIndexesJson("[1]");
        when(wrongSentenceQueryMapper.selectWrongSentences(
                7L, "active", "all", "all", "momentum", "nextReview", 0, 100))
                .thenReturn(List.of(item));
        when(wrongSentenceQueryMapper.countWrongSentences(
                7L, "active", "all", "all", "momentum"))
                .thenReturn(1L);

        ClozeWrongSentencePageResponse response = service.getWrongSentences(
                7L, "invalid", "invalid", "invalid", " Momentum ", "invalid", 0, 500);

        assertEquals(1L, response.getTotal());
        assertEquals(1, response.getCurrent());
        assertEquals(List.of("raw", "momentum"), response.getItems().getFirst().getTargetWords());
        assertEquals(List.of(1), response.getItems().getFirst().getWrongBlankIndexes());
        assertEquals(1, response.getItems().getFirst().getWrongBlankCount());
        assertEquals(0L, response.getSummary().getActiveCount());
    }

    @Test
    void shouldBuildOwnedWrongSentenceDetailWithoutOriginalAnswers() {
        ClozeWrongSentenceItem item = new ClozeWrongSentenceItem();
        item.setProgressId(81L);
        item.setClozeItemId(91L);
        item.setStatus("active");
        item.setReviewStage(1);
        item.setPracticeContext("review");
        item.setTargetWordsJson("[\"raw\",\"momentum\"]");
        item.setWrongBlankIndexesJson("[1]");
        when(wrongSentenceQueryMapper.selectWrongSentenceById(7L, 81L)).thenReturn(item);

        ClozeWrongSentenceAttempt attempt = new ClozeWrongSentenceAttempt();
        attempt.setRecordId(301L);
        attempt.setCorrect(false);
        attempt.setWrongBlankIndexesJson("[1]");
        when(wrongSentenceQueryMapper.selectRecentAttempts(7L, 91L, 5))
                .thenReturn(List.of(attempt));
        when(wrongSentenceQueryMapper.selectWordProgresses(7L, 91L)).thenReturn(List.of());

        ClozeWrongSentenceDetail detail = service.getWrongSentenceDetail(7L, 81L);

        assertEquals(true, detail.getBlanks().get(0).getLastCorrect());
        assertEquals(false, detail.getBlanks().get(1).getLastCorrect());
        assertEquals("review", detail.getAttempts().getFirst().getPracticeContext());
        assertEquals("answer", detail.getAttempts().getFirst().getActionType());
        assertEquals("completed", detail.getReviewStages().getFirst().getState());
        assertEquals("current", detail.getReviewStages().get(1).getState());
    }

    @Test
    void shouldScheduleWrongAnswerForImmediateReview() {
        SentenceClozeItem item = practiceItem(92L, "value");
        when(sentenceClozeItemMapper.selectOwnedByIdForUpdate(92L, 7L)).thenReturn(item);
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

    @Test
    void shouldTreatRevealAsWrongEvenWhenPayloadContainsExpectedWords() {
        SentenceClozeItem item = practiceItem(96L, "value");
        when(sentenceClozeItemMapper.selectOwnedByIdForUpdate(96L, 7L)).thenReturn(item);
        when(answerRecordMapper.selectCount(any())).thenReturn(0L);
        doAnswer(invocation -> {
            SentenceClozeAnswerRecord record = invocation.getArgument(0);
            record.setId(304L);
            return 1;
        }).when(answerRecordMapper).insert(any(SentenceClozeAnswerRecord.class));

        ClozePracticeAnswerRequest request = new ClozePracticeAnswerRequest();
        request.setClozeItemId(96L);
        request.setAnswers(List.of("values"));
        request.setActionType("reveal");

        var response = service.submitAnswer(7L, request);

        assertFalse(response.getCorrect());
        verify(reviewScheduleMapper).upsertWrongSchedule(
                eq(7L), eq(96L), eq(304L), any(LocalDateTime.class));
        verify(reviewScheduleMapper, never()).advanceDueCorrectSchedule(
                any(), any(), any(), any(), any(), any());
        verify(wrongWordAgentNotificationService).notifyClozeWrongAnswer(
                any(), eq(item), eq(List.of("values")), eq(List.of("values")), eq(List.of(0)));
    }

    @Test
    void shouldAdvanceWordAgentSentencePerBlankAndMasterOnlyCompletedWords() {
        SentenceClozeItem item = practiceItem(93L, "raw");
        item.setSource("word-agent");
        item.setBlankWordsJson("[\"raw\",\"momentum\",\"fracture\"]");
        when(sentenceClozeItemMapper.selectOwnedByIdForUpdate(93L, 7L)).thenReturn(item);
        when(answerRecordMapper.selectCount(any())).thenReturn(0L);
        doAnswer(invocation -> {
            SentenceClozeAnswerRecord record = invocation.getArgument(0);
            record.setId(302L);
            return 1;
        }).when(answerRecordMapper).insert(any(SentenceClozeAnswerRecord.class));
        when(wrongWordReviewProgressService.applyAnswer(
                eq(7L),
                eq(item),
                eq(302L),
                argThat(comparison -> comparison.wrongIndexes().equals(List.of(1))),
                any(LocalDateTime.class)
        )).thenReturn(new WrongWordReviewProgressService.WordReviewUpdateResult(
                List.of(new WrongWordReviewProgressService.CompletedReviewWord(33L, "fracture"))
        ));
        Word completedWord = new Word();
        completedWord.setId(33L);
        completedWord.setWord("fracture");
        completedWord.setMeaning("破裂；裂痕");
        when(wordMapper.selectById(33L)).thenReturn(completedWord);

        ClozePracticeAnswerRequest request = new ClozePracticeAnswerRequest();
        request.setClozeItemId(93L);
        request.setAnswers(List.of("ＲＡＷ", "wrong", "fracture"));

        var response = service.submitAnswer(7L, request);

        assertFalse(response.getCorrect());
        verify(reviewScheduleMapper).upsertWrongSchedule(
                eq(7L), eq(93L), eq(302L), any(LocalDateTime.class));
        verify(wrongWordReviewProgressService).applyAnswer(
                eq(7L),
                eq(item),
                eq(302L),
                argThat(comparison -> comparison.wrongIndexes().equals(List.of(1))),
                any(LocalDateTime.class)
        );
        verify(userWordMasteryService).markMasteredFromCloze(
                7L, 33L, "fracture", "破裂；裂痕", 302L);
        verify(wrongWordAgentNotificationService).notifyClozeWrongAnswer(
                any(SentenceClozeAnswerRecord.class),
                eq(item),
                eq(List.of("raw", "momentum", "fracture")),
                eq(List.of("ＲＡＷ", "wrong", "fracture")),
                eq(List.of(1))
        );
    }

    @Test
    void shouldNotMasterWordAgentWordsAfterOnlyTheFirstCorrectReview() {
        SentenceClozeItem item = practiceItem(94L, "raw");
        item.setSource("word-agent");
        when(sentenceClozeItemMapper.selectOwnedByIdForUpdate(94L, 7L)).thenReturn(item);
        when(answerRecordMapper.selectCount(any())).thenReturn(0L);
        doAnswer(invocation -> {
            SentenceClozeAnswerRecord record = invocation.getArgument(0);
            record.setId(303L);
            return 1;
        }).when(answerRecordMapper).insert(any(SentenceClozeAnswerRecord.class));
        when(wrongWordReviewProgressService.applyAnswer(
                eq(7L),
                eq(item),
                eq(303L),
                argThat(ClozeAnswerComparison::correct),
                any(LocalDateTime.class)
        )).thenReturn(new WrongWordReviewProgressService.WordReviewUpdateResult(List.of()));

        ClozePracticeAnswerRequest request = new ClozePracticeAnswerRequest();
        request.setClozeItemId(94L);
        request.setAnswers(List.of("values"));

        var response = service.submitAnswer(7L, request);

        assertEquals(true, response.getCorrect());
        verify(userWordMasteryService, never()).markMasteredFromCloze(
                any(), any(), any(), any(), any());
        verify(reviewScheduleMapper).advanceDueCorrectSchedule(
                eq(7L), eq(94L), eq(303L), any(), any(), any());
    }

    @Test
    void shouldReplayExistingSubmissionWithoutWritingOrAdvancingAgain() {
        SentenceClozeItem item = practiceItem(95L, "value");
        when(sentenceClozeItemMapper.selectOwnedByIdForUpdate(95L, 7L)).thenReturn(item);
        SentenceClozeAnswerRecord existing = new SentenceClozeAnswerRecord();
        existing.setId(901L);
        existing.setUserId(7L);
        existing.setClozeItemId(95L);
        existing.setAnswerText("wrong");
        existing.setAnswersJson("[\"wrong\"]");
        existing.setExpectedWordsJson("[\"values\"]");
        existing.setIsCorrect(false);
        existing.setAttemptNo(2);
        when(answerRecordMapper.selectBySubmissionKey(7L, "submission-1")).thenReturn(existing);

        ClozePracticeAnswerRequest request = new ClozePracticeAnswerRequest();
        request.setClozeItemId(95L);
        request.setAnswers(List.of("wrong"));
        request.setSubmissionKey("submission-1");
        request.setPracticeContext("solo");
        request.setActionType("answer");

        var response = service.submitAnswer(7L, request);

        assertEquals(901L, response.getRecordId());
        assertFalse(response.getCorrect());
        verify(answerRecordMapper, never()).insert(any(SentenceClozeAnswerRecord.class));
        verify(reviewScheduleMapper, never()).upsertWrongSchedule(any(), any(), any(), any());
        verify(reviewScheduleMapper, never()).advanceDueCorrectSchedule(
                any(), any(), any(), any(), any(), any());
        verify(wrongWordReviewProgressService, never()).applyAnswer(
                any(), any(), any(), any(ClozeAnswerComparison.class), any());
        verify(wrongWordAgentNotificationService, never()).notifyClozeWrongAnswer(
                any(), any(), any(), any(), any());
    }

    @Test
    void shouldExposeSuccessfulBaseWordAudioWithoutChangingSentenceAudio() {
        SentenceClozeItem item = practiceItem(90L, "value");
        item.setSentenceAudioUrl("/ai-file-navigation/sentence/value-sentence.wav");
        WordCleanTts wordTts = new WordCleanTts();
        wordTts.setWord("value");
        wordTts.setTtsObjectUrl("/ai-file-navigation/word_tts/value.wav");
        when(sentenceClozeItemMapper.selectDueWrongReviewItems(7L, 10)).thenReturn(List.of(item));
        when(wordCleanTtsMapper.selectSuccessfulByWords(List.of("value"))).thenReturn(List.of(wordTts));

        ClozePracticeTaskResponse response = service.getDueReviewTasks(7L, 10).getFirst();

        assertEquals("/ai-file-navigation/word_tts/value.wav", response.getWordAudioUrl());
        assertEquals("/ai-file-navigation/sentence/value-sentence.wav", response.getSentenceAudioUrl());
    }

    @Test
    void shouldLeaveWordAudioEmptyWhenNoSuccessfulTtsExists() {
        SentenceClozeItem item = practiceItem(91L, "missing");
        when(sentenceClozeItemMapper.selectDueWrongReviewItems(7L, 10)).thenReturn(List.of(item));
        when(wordCleanTtsMapper.selectSuccessfulByWords(List.of("missing"))).thenReturn(List.of());

        ClozePracticeTaskResponse response = service.getDueReviewTasks(7L, 10).getFirst();

        assertNull(response.getWordAudioUrl());
    }

    private SentenceClozeItem practiceItem(Long id, String word) {
        SentenceClozeItem item = new SentenceClozeItem();
        item.setId(id);
        item.setUserId(7L);
        item.setWord(word);
        item.setSentence("The company values fairness.");
        item.setClozeSentence("The company ____ fairness.");
        item.setBlankWordsJson("[\"values\"]");
        return item;
    }

    private void setRequiredField(Object target, String fieldName, Object value) {
        Field field = assertDoesNotThrow(
                () -> target.getClass().getDeclaredField(fieldName),
                () -> "Missing required field: " + fieldName
        );
        field.setAccessible(true);
        assertDoesNotThrow(() -> field.set(target, value));
    }

    private Object requiredGetter(Object target, String methodName) {
        return assertDoesNotThrow(
                () -> target.getClass().getMethod(methodName).invoke(target),
                () -> "Missing required getter: " + methodName
        );
    }
}
