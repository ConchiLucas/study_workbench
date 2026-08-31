package com.robword.mapper;

import org.apache.ibatis.annotations.Select;
import org.junit.jupiter.api.Test;

import java.lang.reflect.Method;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

class ClozeWrongSentenceQueryMapperContractTest {

    @Test
    void listQueryIsProgressDrivenOwnedFilteredAndStable() throws Exception {
        Method method = ClozeWrongSentenceQueryMapper.class.getMethod(
                "selectWrongSentences",
                Long.class,
                String.class,
                String.class,
                String.class,
                String.class,
                String.class,
                Integer.class,
                Integer.class
        );
        String sql = String.join("\n", method.getAnnotation(Select.class).value());

        assertTrue(sql.contains("FROM sentence_cloze_review_schedule progress"));
        assertTrue(sql.contains("progress.status = #{status}"));
        assertTrue(sql.contains("last_wrong_answer_record_id"));
        assertTrue(sql.contains("item.user_id = #{userId}"));
        assertTrue(sql.contains("#{keyword}"));
        assertTrue(sql.contains("#{source}"));
        assertTrue(sql.contains("#{availability}"));
        assertTrue(sql.contains("jsonb_array_elements_text"));
        assertTrue(sql.contains("item.blank_words_json"));
        assertTrue(sql.contains("WHEN item.source = 'best-sentence-practice'"));
        assertTrue(sql.contains("ORDER BY"));
        assertTrue(sql.contains("progress.id DESC"));
        assertTrue(sql.contains("LIMIT #{size} OFFSET #{offset}"));
        assertFalse(sql.contains("answer_text"));
        assertFalse(sql.contains("provider_id"));
        assertFalse(sql.contains("model"));

        Method count = ClozeWrongSentenceQueryMapper.class.getMethod(
                "countWrongSentences",
                Long.class,
                String.class,
                String.class,
                String.class,
                String.class
        );
        String countSql = String.join("\n", count.getAnnotation(Select.class).value());
        assertTrue(countSql.contains("jsonb_array_elements_text"));
        assertTrue(countSql.contains("item.blank_words_json"));
    }

    @Test
    void detailAndAttemptQueriesEnforceOwnershipAndBoundHistory() throws Exception {
        Method detail = ClozeWrongSentenceQueryMapper.class.getMethod(
                "selectWrongSentenceById", Long.class, Long.class
        );
        String detailSql = String.join("\n", detail.getAnnotation(Select.class).value());
        assertTrue(detailSql.contains("progress.id = #{progressId}"));
        assertTrue(detailSql.contains("progress.user_id = #{userId}"));
        assertTrue(detailSql.contains("item.user_id = #{userId}"));
        assertTrue(detailSql.contains("WHEN item.source = 'best-sentence-practice'"));

        Method attempts = ClozeWrongSentenceQueryMapper.class.getMethod(
                "selectRecentAttempts", Long.class, Long.class, Integer.class
        );
        String attemptsSql = String.join("\n", attempts.getAnnotation(Select.class).value());
        assertTrue(attemptsSql.contains("record.user_id = #{userId}"));
        assertTrue(attemptsSql.contains("record.cloze_item_id = #{clozeItemId}"));
        assertTrue(attemptsSql.contains("ORDER BY record.create_time DESC, record.id DESC"));
        assertTrue(attemptsSql.contains("LIMIT #{limit}"));
        assertFalse(attemptsSql.contains("answer_text"));
        assertFalse(attemptsSql.contains("answers_json"));
    }
}
