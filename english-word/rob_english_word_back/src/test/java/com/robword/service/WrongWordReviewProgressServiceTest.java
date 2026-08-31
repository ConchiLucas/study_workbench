package com.robword.service;

import com.robword.entity.SentenceClozeItem;
import com.robword.entity.WrongWordReviewProgress;
import com.robword.mapper.WrongWordReviewProgressMapper;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.time.LocalDateTime;
import java.util.List;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class WrongWordReviewProgressServiceTest {

    @Mock
    private WrongWordReviewProgressMapper mapper;

    private WrongWordReviewProgressService service;

    @BeforeEach
    void setUp() {
        service = new WrongWordReviewProgressService(mapper);
    }

    @Test
    void normalizesWordWhenRecordingWrongAnswer() {
        LocalDateTime wrongAt = LocalDateTime.of(2026, 7, 26, 20, 0);

        service.recordWrong(7L, 11L, "  Momentum ", wrongAt, 301L);

        verify(mapper).upsertWrong(7L, 11L, "Momentum", "momentum", wrongAt, 301L);
    }

    @Test
    void linksEveryGeneratedWordToItsOwnBlankIndex() {
        LocalDateTime dueAt = LocalDateTime.of(2026, 7, 26, 20, 5);
        when(mapper.linkActiveSentence(eq(7L), any(), eq(91L), any(), eq(dueAt)))
                .thenReturn(1);

        service.linkGeneratedSentence(
                7L,
                91L,
                List.of("raw", "momentum", "fracture"),
                dueAt
        );

        verify(mapper).linkActiveSentence(7L, "raw", 91L, 0, dueAt);
        verify(mapper).linkActiveSentence(7L, "momentum", 91L, 1, dueAt);
        verify(mapper).linkActiveSentence(7L, "fracture", 91L, 2, dueAt);
    }

    @Test
    void advancesDueCorrectWordsAndResetsOnlyTheWrongWord() {
        LocalDateTime answeredAt = LocalDateTime.of(2026, 7, 26, 20, 10);
        SentenceClozeItem item = new SentenceClozeItem();
        item.setId(91L);
        item.setUserId(7L);
        item.setSource("word-agent");
        List<WrongWordReviewProgress> progresses = List.of(
                progress(1L, 0, 0, "raw", answeredAt.minusMinutes(1)),
                progress(2L, 1, 1, "momentum", answeredAt.plusDays(2)),
                progress(3L, 2, 2, "fracture", answeredAt.minusMinutes(1))
        );
        when(mapper.selectByActiveItem(7L, 91L)).thenReturn(progresses);
        when(mapper.advanceDueCorrect(
                eq(7L), eq(91L), eq(0), eq(401L), eq(answeredAt), any(), any()))
                .thenReturn(1);
        when(mapper.advanceDueCorrect(
                eq(7L), eq(91L), eq(2), eq(401L), eq(answeredAt), any(), any()))
                .thenReturn(1);

        ClozeAnswerComparison comparison = ClozeAnswerComparison.compare(
                List.of("raw", "wrong", "fracture"),
                List.of("raw", "momentum", "fracture")
        );
        WrongWordReviewProgressService.WordReviewUpdateResult result = service.applyAnswer(
                7L, item, 401L, comparison, answeredAt);

        verify(mapper).advanceDueCorrect(
                7L, 91L, 0, 401L, answeredAt, answeredAt.plusDays(7), answeredAt.plusDays(15));
        verify(mapper, never()).advanceDueCorrect(
                eq(7L), eq(91L), eq(1), eq(401L), eq(answeredAt), any(), any());
        verify(mapper).advanceDueCorrect(
                7L, 91L, 2, 401L, answeredAt, answeredAt.plusDays(7), answeredAt.plusDays(15));
        verify(mapper).upsertWrong(7L, 2L, "momentum", "momentum", answeredAt, 401L);
        assertEquals(List.of("fracture"), result.completedWords().stream()
                .map(WrongWordReviewProgressService.CompletedReviewWord::word)
                .toList());
    }

    @Test
    void doesNotAdvanceCorrectWordBeforeItIsDue() {
        LocalDateTime answeredAt = LocalDateTime.of(2026, 7, 26, 20, 10);
        SentenceClozeItem item = new SentenceClozeItem();
        item.setId(91L);
        item.setUserId(7L);
        item.setSource("word-agent");
        when(mapper.selectByActiveItem(7L, 91L)).thenReturn(List.of(
                progress(1L, 0, 1, "raw", answeredAt.plusDays(3))
        ));

        ClozeAnswerComparison comparison = ClozeAnswerComparison.compare(
                List.of("raw"), List.of("raw"));
        WrongWordReviewProgressService.WordReviewUpdateResult result = service.applyAnswer(
                7L, item, 402L, comparison, answeredAt);

        verify(mapper, never()).advanceDueCorrect(
                eq(7L), eq(91L), eq(0), eq(402L), eq(answeredAt), any(), any());
        assertEquals(List.of(), result.completedWords());
    }

    private WrongWordReviewProgress progress(
            Long wordId,
            int blankIndex,
            int reviewStage,
            String word,
            LocalDateTime nextReviewTime
    ) {
        WrongWordReviewProgress progress = new WrongWordReviewProgress();
        progress.setUserId(7L);
        progress.setWordId(wordId);
        progress.setWord(word);
        progress.setNormalizedWord(word);
        progress.setStatus(reviewStage == 0 ? "due" : "waiting");
        progress.setReviewStage(reviewStage);
        progress.setNextReviewTime(nextReviewTime);
        progress.setActiveClozeItemId(91L);
        progress.setActiveBlankIndex(blankIndex);
        return progress;
    }
}
