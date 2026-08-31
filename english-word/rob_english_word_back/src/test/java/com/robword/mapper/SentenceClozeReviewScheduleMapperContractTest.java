package com.robword.mapper;

import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Select;
import org.apache.ibatis.annotations.Update;
import org.junit.jupiter.api.Test;

import java.lang.reflect.Method;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.LocalDateTime;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

class SentenceClozeReviewScheduleMapperContractTest {

    @Test
    void schemaPersistsCompletedProgressAndSubmissionMetadata() throws Exception {
        String scheduleSql = Files.readString(Path.of("db/sentence_cloze_review_schedule.sql"));
        String answerSql = Files.readString(Path.of("db/sentence_cloze_answer_record.sql"));

        assertTrue(scheduleSql.contains("status varchar(32) NOT NULL DEFAULT 'active'"));
        assertTrue(scheduleSql.contains("wrong_count int4 NOT NULL DEFAULT 1"));
        assertTrue(scheduleSql.contains("first_wrong_time timestamp NULL"));
        assertTrue(scheduleSql.contains("last_wrong_answer_record_id bigint NULL"));
        assertTrue(scheduleSql.contains("completed_time timestamp NULL"));
        assertTrue(scheduleSql.contains("CHECK (status IN ('active', 'completed'))"));
        assertTrue(answerSql.contains("submission_key varchar(64) NULL"));
        assertTrue(answerSql.contains("practice_context varchar(16) NULL"));
        assertTrue(answerSql.contains("action_type varchar(16) NULL"));
        assertTrue(answerSql.contains("wrong_blank_indexes_json text NOT NULL DEFAULT '[]'"));
        assertTrue(answerSql.contains("uk_sentence_cloze_answer_user_submission"));
        assertTrue(scheduleSql.contains("latest_is_correct = false"));
        assertTrue(scheduleSql.contains("UPDATE public.sentence_cloze_review_schedule"));
    }

    @Test
    void completedTaskStatsIncludePersistentCompletedSchedules() throws Exception {
        Method method = SentenceClozeAnswerRecordMapper.class.getMethod(
                "countCompletedItems", Long.class
        );
        String sql = String.join("\n", method.getAnnotation(Select.class).value());

        assertTrue(sql.contains("review.id IS NULL"));
        assertTrue(sql.contains("review.status = 'completed'"));
    }

    @Test
    void wrongAnswerReactivatesOnePersistentSentenceProgress() throws Exception {
        Method method = SentenceClozeReviewScheduleMapper.class.getMethod(
                "upsertWrongSchedule",
                Long.class,
                Long.class,
                Long.class,
                LocalDateTime.class
        );
        String sql = String.join("\n", method.getAnnotation(Insert.class).value());

        assertTrue(sql.contains("ON CONFLICT (user_id, cloze_item_id)"));
        assertTrue(sql.contains("wrong_count = sentence_cloze_review_schedule.wrong_count + 1"));
        assertTrue(sql.contains("status = 'active'"));
        assertTrue(sql.contains("review_stage = 0"));
        assertTrue(sql.contains("completed_time = NULL"));
        assertTrue(sql.contains("last_wrong_answer_record_id"));
        assertTrue(sql.contains("CURRENT_TIMESTAMP"));
        assertFalse(sql.contains("#{answeredAt}"));
        assertTrue(sql.contains("EXCLUDED.last_answer_record_id >"));
        assertTrue(sql.contains("newer.id > #{recordId}"));
    }

    @Test
    void answerSubmissionLocksTheSentenceBeforeAllocatingItsAttempt() throws Exception {
        Method method = SentenceClozeItemMapper.class.getMethod(
                "selectOwnedByIdForUpdate", Long.class, Long.class
        );
        String sql = String.join("\n", method.getAnnotation(Select.class).value());

        assertTrue(sql.contains("WHERE id = #{id}"));
        assertTrue(sql.contains("AND user_id = #{userId}"));
        assertTrue(sql.contains("FOR UPDATE"));
    }

    @Test
    void dueCorrectAnswerAdvancesAtomicallyAndKeepsCompletedRow() throws Exception {
        Method method = SentenceClozeReviewScheduleMapper.class.getMethod(
                "advanceDueCorrectSchedule",
                Long.class,
                Long.class,
                Long.class,
                LocalDateTime.class,
                LocalDateTime.class,
                LocalDateTime.class
        );
        String sql = String.join("\n", method.getAnnotation(Update.class).value());

        assertTrue(sql.contains("next_review_time <= CURRENT_TIMESTAMP"));
        assertTrue(sql.contains("CURRENT_TIMESTAMP + INTERVAL '7 days'"));
        assertTrue(sql.contains("CURRENT_TIMESTAMP + INTERVAL '15 days'"));
        assertTrue(sql.contains("WHEN review_stage >= 2 THEN 'completed'"));
        assertTrue(sql.contains("WHEN review_stage >= 2 THEN 3"));
        assertTrue(sql.contains("next_review_time = CASE"));
        assertTrue(sql.contains("ELSE NULL"));
        assertTrue(sql.contains("last_answer_record_id <> #{recordId}"));
        assertFalse(sql.contains("DELETE FROM sentence_cloze_review_schedule"));
    }
}
