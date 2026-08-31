package com.robword.mapper;

import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Update;
import org.junit.jupiter.api.Test;

import java.lang.reflect.Method;
import java.time.LocalDateTime;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

class WrongWordReviewProgressMapperContractTest {

    @Test
    void wrongAnswerUpsertReactivatesAndResetsOneNormalizedWord() throws Exception {
        Method method = WrongWordReviewProgressMapper.class.getMethod(
                "upsertWrong",
                Long.class,
                Long.class,
                String.class,
                String.class,
                LocalDateTime.class,
                Long.class
        );
        String sql = String.join("\n", method.getAnnotation(Insert.class).value());

        assertTrue(sql.contains("ON CONFLICT (user_id, normalized_word)"));
        assertTrue(sql.contains("wrong_count = wrong_word_review_progress.wrong_count + 1"));
        assertTrue(sql.contains("review_stage = 0"));
        assertTrue(sql.contains("completed_time = NULL"));
        assertTrue(sql.contains("active_cloze_item_id IS NULL"));
        assertTrue(sql.contains(
                "EXCLUDED.last_wrong_time >= wrong_word_review_progress.last_wrong_time"));
        assertTrue(sql.contains("EXCLUDED.last_answer_record_id IS DISTINCT FROM"));
        assertTrue(sql.contains("wrong_word_review_progress.last_answer_record_id"));
    }

    @Test
    void generatedSentenceLinksEachWordByBlankIndexAndMakesItDue() throws Exception {
        Method method = WrongWordReviewProgressMapper.class.getMethod(
                "linkActiveSentence",
                Long.class,
                String.class,
                Long.class,
                Integer.class,
                LocalDateTime.class
        );
        String sql = String.join("\n", method.getAnnotation(Update.class).value());

        assertTrue(sql.contains("active_cloze_item_id = #{clozeItemId}"));
        assertTrue(sql.contains("active_blank_index = #{blankIndex}"));
        assertTrue(sql.contains("status = 'due'"));
        assertTrue(sql.contains("next_review_time = #{dueTime}"));
        assertTrue(sql.contains("status <> 'completed'"));
        assertTrue(sql.contains("active_cloze_item_id IS NULL"));
    }

    @Test
    void dueCorrectAnswerAdvancesSevenFifteenThenCompletesAtomically() throws Exception {
        Method method = WrongWordReviewProgressMapper.class.getMethod(
                "advanceDueCorrect",
                Long.class,
                Long.class,
                Integer.class,
                Long.class,
                LocalDateTime.class,
                LocalDateTime.class,
                LocalDateTime.class
        );
        String sql = String.join("\n", method.getAnnotation(Update.class).value());

        assertTrue(sql.contains("next_review_time <= #{answeredAt}"));
        assertTrue(sql.contains("THEN 3"));
        assertTrue(sql.contains("THEN 'completed'"));
        assertTrue(sql.contains("#{sevenDaysAt}"));
        assertTrue(sql.contains("#{fifteenDaysAt}"));
        assertFalse(sql.contains("last_answer_record_id = #{answerRecordId}"));
    }
}
