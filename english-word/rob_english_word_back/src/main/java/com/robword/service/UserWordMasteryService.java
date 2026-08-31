package com.robword.service;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.core.conditions.update.LambdaUpdateWrapper;
import com.robword.entity.GameAnswerDetail;
import com.robword.entity.UserWordMasteryProgress;
import com.robword.entity.Word;
import com.robword.mapper.UserWordMasteryProgressMapper;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.LocalDateTime;
import java.util.List;

@Service
@RequiredArgsConstructor
@Slf4j
public class UserWordMasteryService {

    private static final String MATCH_MODE = "match";
    private static final String SOLO_TRAINING_MODE = "solo_training";
    private static final String STATUS_LEARNING = "learning";
    private static final String STATUS_MASTERED = "mastered";
    private static final int STAGE_NOT_STARTED = 0;
    private static final int STAGE_FIRST_CORRECT = 1;
    private static final int STAGE_DAY1_CORRECT = 2;
    private static final int STAGE_MASTERED = 3;

    private final UserWordMasteryProgressMapper masteryProgressMapper;

    @Transactional
    public void recordAnswer(GameAnswerDetail detail, String mode) {
        if (detail == null || detail.getUserId() == null || detail.getWordId() == null) {
            return;
        }
        if (!SOLO_TRAINING_MODE.equals(mode) && !MATCH_MODE.equals(mode)) {
            return;
        }
        if (detail.getUserId() <= 0) {
            return;
        }

        boolean correct = detail.getIsCorrect() != null && detail.getIsCorrect() == 1;
        if (MATCH_MODE.equals(mode) && correct) {
            recordMatchCorrectAnswer(detail);
        } else if (correct) {
            recordCorrectAnswer(detail);
        } else {
            resetLearningProgress(detail);
        }
    }

    public List<Word> dueReviewWords(Long userId, List<String> libraryNames, int limit) {
        if (userId == null || userId <= 0 || limit <= 0) {
            return List.of();
        }
        return masteryProgressMapper.selectDueReviewWords(userId, libraryNames, limit);
    }

    @Transactional
    public void markMasteredFromCloze(Long userId,
                                      Long wordId,
                                      String wordContent,
                                      String correctMeaning,
                                      Long clozeAnswerRecordId) {
        if (userId == null || userId <= 0 || wordId == null || wordId <= 0) {
            return;
        }

        LocalDateTime now = LocalDateTime.now();
        UserWordMasteryProgress progress = findProgressByWord(userId, wordId, wordContent);
        if (progress == null) {
            UserWordMasteryProgress next = new UserWordMasteryProgress();
            next.setUserId(userId);
            next.setWordId(wordId);
            next.setWordContent(wordContent);
            next.setCorrectMeaning(correctMeaning);
            next.setStatus(STATUS_MASTERED);
            next.setStage(STAGE_MASTERED);
            next.setCorrectCount(1);
            next.setFirstCorrectTime(now);
            next.setDay1CorrectTime(now);
            next.setDay7CorrectTime(now);
            next.setLastCorrectTime(now);
            next.setMasteredTime(now);
            next.setLastAnswerDetailId(clozeAnswerRecordId);
            masteryProgressMapper.insert(next);
            return;
        }

        String nextWordContent = nonBlank(wordContent, progress.getWordContent());
        String nextCorrectMeaning = nonBlank(correctMeaning, progress.getCorrectMeaning());
        masteryProgressMapper.update(
                null,
                new LambdaUpdateWrapper<UserWordMasteryProgress>()
                        .eq(UserWordMasteryProgress::getId, progress.getId())
                        .set(UserWordMasteryProgress::getWordContent, nextWordContent)
                        .set(UserWordMasteryProgress::getCorrectMeaning, nextCorrectMeaning)
                        .set(UserWordMasteryProgress::getStatus, STATUS_MASTERED)
                        .set(UserWordMasteryProgress::getStage, STAGE_MASTERED)
                        .set(UserWordMasteryProgress::getCorrectCount, safeInt(progress.getCorrectCount()) + 1)
                        .set(UserWordMasteryProgress::getFirstCorrectTime, progress.getFirstCorrectTime() == null ? now : progress.getFirstCorrectTime())
                        .set(UserWordMasteryProgress::getDay1CorrectTime, progress.getDay1CorrectTime() == null ? now : progress.getDay1CorrectTime())
                        .set(UserWordMasteryProgress::getDay7CorrectTime, now)
                        .set(UserWordMasteryProgress::getNextReviewTime, null)
                        .set(UserWordMasteryProgress::getLastCorrectTime, now)
                        .set(UserWordMasteryProgress::getMasteredTime, progress.getMasteredTime() == null ? now : progress.getMasteredTime())
                        .set(UserWordMasteryProgress::getLastAnswerDetailId, clozeAnswerRecordId)
                        .set(UserWordMasteryProgress::getUpdateTime, now)
        );
    }

    private void recordCorrectAnswer(GameAnswerDetail detail) {
        LocalDateTime now = LocalDateTime.now();
        UserWordMasteryProgress progress = findProgress(detail.getUserId(), detail.getWordId());

        if (progress == null) {
            UserWordMasteryProgress next = new UserWordMasteryProgress();
            next.setUserId(detail.getUserId());
            next.setWordId(detail.getWordId());
            next.setWordContent(detail.getWordContent());
            next.setCorrectMeaning(resolveCorrectMeaning(detail));
            next.setStatus(STATUS_LEARNING);
            next.setStage(STAGE_FIRST_CORRECT);
            next.setCorrectCount(1);
            next.setFirstCorrectTime(now);
            next.setNextReviewTime(now.plusDays(1));
            next.setLastCorrectTime(now);
            next.setLastAnswerDetailId(detail.getId());
            masteryProgressMapper.insert(next);
            return;
        }

        progress.setWordContent(nonBlank(detail.getWordContent(), progress.getWordContent()));
        progress.setCorrectMeaning(nonBlank(resolveCorrectMeaning(detail), progress.getCorrectMeaning()));
        progress.setCorrectCount(safeInt(progress.getCorrectCount()) + 1);
        progress.setLastCorrectTime(now);
        progress.setLastAnswerDetailId(detail.getId());

        if (STATUS_MASTERED.equals(progress.getStatus())) {
            masteryProgressMapper.updateById(progress);
            return;
        }

        int stage = safeInt(progress.getStage());
        if (stage <= STAGE_NOT_STARTED) {
            progress.setStatus(STATUS_LEARNING);
            progress.setStage(STAGE_FIRST_CORRECT);
            progress.setFirstCorrectTime(now);
            progress.setDay1CorrectTime(null);
            progress.setDay7CorrectTime(null);
            progress.setMasteredTime(null);
            progress.setNextReviewTime(now.plusDays(1));
        } else if (stage == STAGE_FIRST_CORRECT && due(progress.getNextReviewTime(), now)) {
            progress.setStage(STAGE_DAY1_CORRECT);
            progress.setDay1CorrectTime(now);
            progress.setNextReviewTime(now.plusDays(7));
        } else if (stage == STAGE_DAY1_CORRECT && due(progress.getNextReviewTime(), now)) {
            progress.setStage(STAGE_MASTERED);
            progress.setStatus(STATUS_MASTERED);
            progress.setDay7CorrectTime(now);
            progress.setMasteredTime(now);
            progress.setNextReviewTime(null);
        }

        masteryProgressMapper.updateById(progress);
    }

    private void recordMatchCorrectAnswer(GameAnswerDetail detail) {
        LocalDateTime now = LocalDateTime.now();
        UserWordMasteryProgress progress = findProgress(detail.getUserId(), detail.getWordId());

        if (progress == null) {
            UserWordMasteryProgress next = new UserWordMasteryProgress();
            next.setUserId(detail.getUserId());
            next.setWordId(detail.getWordId());
            next.setWordContent(detail.getWordContent());
            next.setCorrectMeaning(resolveCorrectMeaning(detail));
            next.setStatus(STATUS_LEARNING);
            next.setStage(STAGE_FIRST_CORRECT);
            next.setCorrectCount(1);
            next.setFirstCorrectTime(now);
            next.setLastCorrectTime(now);
            next.setLastAnswerDetailId(detail.getId());
            masteryProgressMapper.insert(next);
            return;
        }

        progress.setWordContent(nonBlank(detail.getWordContent(), progress.getWordContent()));
        progress.setCorrectMeaning(nonBlank(resolveCorrectMeaning(detail), progress.getCorrectMeaning()));
        progress.setCorrectCount(safeInt(progress.getCorrectCount()) + 1);
        progress.setLastCorrectTime(now);
        progress.setLastAnswerDetailId(detail.getId());

        if (STATUS_MASTERED.equals(progress.getStatus())) {
            masteryProgressMapper.updateById(progress);
            return;
        }

        int nextStage = Math.min(STAGE_MASTERED, safeInt(progress.getStage()) + 1);
        progress.setStatus(nextStage >= STAGE_MASTERED ? STATUS_MASTERED : STATUS_LEARNING);
        progress.setStage(nextStage);
        progress.setFirstCorrectTime(progress.getFirstCorrectTime() == null ? now : progress.getFirstCorrectTime());
        progress.setNextReviewTime(null);

        if (nextStage >= STAGE_DAY1_CORRECT && progress.getDay1CorrectTime() == null) {
            progress.setDay1CorrectTime(now);
        }
        if (nextStage >= STAGE_MASTERED) {
            progress.setDay7CorrectTime(now);
            progress.setMasteredTime(now);
        }

        masteryProgressMapper.updateById(progress);
    }

    private void resetLearningProgress(GameAnswerDetail detail) {
        UserWordMasteryProgress progress = findProgress(detail.getUserId(), detail.getWordId());
        if (progress == null || STATUS_MASTERED.equals(progress.getStatus())) {
            return;
        }

        masteryProgressMapper.update(
                null,
                new LambdaUpdateWrapper<UserWordMasteryProgress>()
                        .eq(UserWordMasteryProgress::getId, progress.getId())
                        .set(UserWordMasteryProgress::getWordContent, nonBlank(detail.getWordContent(), progress.getWordContent()))
                        .set(UserWordMasteryProgress::getCorrectMeaning, nonBlank(resolveCorrectMeaning(detail), progress.getCorrectMeaning()))
                        .set(UserWordMasteryProgress::getStage, STAGE_NOT_STARTED)
                        .set(UserWordMasteryProgress::getNextReviewTime, null)
                        .set(UserWordMasteryProgress::getFirstCorrectTime, null)
                        .set(UserWordMasteryProgress::getDay1CorrectTime, null)
                        .set(UserWordMasteryProgress::getDay7CorrectTime, null)
                        .set(UserWordMasteryProgress::getMasteredTime, null)
                        .set(UserWordMasteryProgress::getLastAnswerDetailId, detail.getId())
                        .set(UserWordMasteryProgress::getUpdateTime, LocalDateTime.now())
        );
    }

    private UserWordMasteryProgress findProgress(Long userId, Long wordId) {
        return masteryProgressMapper.selectOne(
                new LambdaQueryWrapper<UserWordMasteryProgress>()
                        .eq(UserWordMasteryProgress::getUserId, userId)
                        .eq(UserWordMasteryProgress::getWordId, wordId)
                        .last("LIMIT 1")
        );
    }

    private UserWordMasteryProgress findProgressByWord(Long userId, Long wordId, String wordContent) {
        UserWordMasteryProgress progress = findProgress(userId, wordId);
        if (progress != null || wordContent == null || wordContent.isBlank()) {
            return progress;
        }
        return masteryProgressMapper.selectOne(
                new LambdaQueryWrapper<UserWordMasteryProgress>()
                        .eq(UserWordMasteryProgress::getUserId, userId)
                        .apply("lower(btrim(word_content)) = lower(btrim({0}))", wordContent.trim())
                        .last("LIMIT 1")
        );
    }

    private boolean due(LocalDateTime reviewTime, LocalDateTime now) {
        return reviewTime == null || !now.isBefore(reviewTime);
    }

    private int safeInt(Integer value) {
        return value != null ? value : 0;
    }

    private String nonBlank(String next, String fallback) {
        return next != null && !next.isBlank() ? next : fallback;
    }

    private String resolveCorrectMeaning(GameAnswerDetail detail) {
        if (detail == null || detail.getCorrectAnswerIndex() == null) {
            return null;
        }
        return switch (detail.getCorrectAnswerIndex()) {
            case 1 -> detail.getOption1();
            case 2 -> detail.getOption2();
            case 3 -> detail.getOption3();
            case 4 -> detail.getOption4();
            default -> null;
        };
    }
}
