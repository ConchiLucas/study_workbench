package com.robword.service;

import com.robword.entity.SentenceClozeItem;
import com.robword.entity.WrongWordReviewProgress;
import com.robword.mapper.WrongWordReviewProgressMapper;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;

@Service
@RequiredArgsConstructor
public class WrongWordReviewProgressService {

    private final WrongWordReviewProgressMapper mapper;

    @Transactional
    public void recordWrong(
            Long userId,
            Long wordId,
            String rawWord,
            LocalDateTime wrongAt,
            Long answerRecordId
    ) {
        String word = cleanWord(rawWord);
        if (userId == null || userId <= 0 || word.isEmpty()) {
            return;
        }
        LocalDateTime effectiveWrongAt = wrongAt == null ? LocalDateTime.now() : wrongAt;
        mapper.upsertWrong(
                userId,
                wordId,
                word,
                normalizeWord(word),
                effectiveWrongAt,
                answerRecordId
        );
    }

    @Transactional
    public void linkGeneratedSentence(
            Long userId,
            Long clozeItemId,
            List<String> rawWords,
            LocalDateTime dueAt
    ) {
        if (userId == null || userId <= 0 || clozeItemId == null || clozeItemId <= 0
                || rawWords == null || rawWords.isEmpty()) {
            return;
        }
        LocalDateTime effectiveDueAt = dueAt == null ? LocalDateTime.now() : dueAt;
        for (int index = 0; index < rawWords.size(); index++) {
            String word = cleanWord(rawWords.get(index));
            if (word.isEmpty()) {
                continue;
            }
            String normalizedWord = normalizeWord(word);
            int updated = mapper.linkActiveSentence(
                    userId,
                    normalizedWord,
                    clozeItemId,
                    index,
                    effectiveDueAt
            );
            if (updated > 0) {
                continue;
            }

            WrongWordReviewProgress existing = mapper.selectByUserAndNormalizedWord(
                    userId,
                    normalizedWord
            );
            if (existing != null && "completed".equals(existing.getStatus())) {
                continue;
            }
            if (existing == null) {
                recordWrong(userId, null, word, effectiveDueAt, null);
            }
            mapper.linkActiveSentence(
                    userId,
                    normalizedWord,
                    clozeItemId,
                    index,
                    effectiveDueAt
            );
        }
    }

    @Transactional
    public WordReviewUpdateResult applyAnswer(
            Long userId,
            SentenceClozeItem item,
            Long answerRecordId,
            ClozeAnswerComparison comparison,
            LocalDateTime answeredAt
    ) {
        if (userId == null || item == null || item.getId() == null
                || comparison == null || comparison.expectedWords().isEmpty()) {
            return new WordReviewUpdateResult(List.of());
        }

        LocalDateTime effectiveAnsweredAt = answeredAt == null ? LocalDateTime.now() : answeredAt;
        List<String> expectedWords = comparison.expectedWords();
        Map<Integer, WrongWordReviewProgress> progressByIndex = new HashMap<>();
        for (WrongWordReviewProgress progress : mapper.selectByActiveItem(userId, item.getId())) {
            if (progress.getActiveBlankIndex() != null) {
                progressByIndex.put(progress.getActiveBlankIndex(), progress);
            }
        }

        List<CompletedReviewWord> completedWords = new ArrayList<>();
        for (int index = 0; index < expectedWords.size(); index++) {
            String expectedWord = cleanWord(expectedWords.get(index));
            if (expectedWord.isEmpty()) {
                continue;
            }
            WrongWordReviewProgress progress = progressByIndex.get(index);

            if (comparison.wrongIndexes().contains(index)) {
                recordWrong(
                        userId,
                        progress == null ? null : progress.getWordId(),
                        expectedWord,
                        effectiveAnsweredAt,
                        answerRecordId
                );
                continue;
            }
            if (progress == null || progress.getNextReviewTime() == null
                    || progress.getNextReviewTime().isAfter(effectiveAnsweredAt)) {
                continue;
            }

            int updated = mapper.advanceDueCorrect(
                    userId,
                    item.getId(),
                    index,
                    answerRecordId,
                    effectiveAnsweredAt,
                    effectiveAnsweredAt.plusDays(7),
                    effectiveAnsweredAt.plusDays(15)
            );
            if (updated > 0 && Integer.valueOf(2).equals(progress.getReviewStage())) {
                completedWords.add(new CompletedReviewWord(progress.getWordId(), expectedWord));
            }
        }
        return new WordReviewUpdateResult(List.copyOf(completedWords));
    }

    private String cleanWord(String value) {
        return value == null ? "" : value.trim();
    }

    private String normalizeWord(String value) {
        return cleanWord(value).toLowerCase(Locale.ROOT);
    }

    public record CompletedReviewWord(Long wordId, String word) {
    }

    public record WordReviewUpdateResult(List<CompletedReviewWord> completedWords) {
    }
}
