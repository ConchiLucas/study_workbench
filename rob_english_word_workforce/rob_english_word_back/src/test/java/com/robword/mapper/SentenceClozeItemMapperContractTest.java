package com.robword.mapper;

import org.apache.ibatis.annotations.Select;
import org.junit.jupiter.api.Test;

import java.lang.reflect.Method;

import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.junit.jupiter.api.Assertions.assertFalse;

class SentenceClozeItemMapperContractTest {

    @Test
    void generationKeyLookupUsesUniqueBusinessKey() throws Exception {
        Method method = SentenceClozeItemMapper.class.getMethod(
                "selectByGenerationKey", String.class
        );
        String sql = String.join("\n", method.getAnnotation(Select.class).value());

        assertTrue(sql.contains("generation_key = #{generationKey}"));
        assertTrue(sql.contains("LIMIT 1"));
    }

    @Test
    void dueReviewQueryMergesSentenceAndWordProgressThenDeduplicates() throws Exception {
        Method method = SentenceClozeItemMapper.class.getMethod(
                "selectDueWrongReviewItems", Long.class, Integer.class
        );
        String sql = String.join("\n", method.getAnnotation(Select.class).value());

        assertTrue(sql.contains("sentence_due AS"));
        assertTrue(sql.contains("word_due AS"));
        assertTrue(sql.contains("combined_due AS"));
        assertTrue(sql.contains("deduped_due AS"));
        assertTrue(sql.contains("UNION ALL"));
        assertTrue(sql.contains("GROUP BY cloze_item_id"));
        assertTrue(sql.contains("MIN(next_review_time)"));
        assertTrue(sql.contains("wrong_word_review_progress"));
        assertFalse(sql.contains("latest.is_correct = false"));
        assertTrue(sql.contains("i.user_id = #{userId}"));
        assertFalse(sql.contains("i.source = 'word-agent'"));
        assertTrue(sql.contains("LIMIT #{limit}"));
    }

    @Test
    void dueReviewCountUsesTheSameDeduplicatedProjectionWithoutLimit() throws Exception {
        Method method = SentenceClozeItemMapper.class.getMethod(
                "countDueReviewItems", Long.class
        );
        String sql = String.join("\n", method.getAnnotation(Select.class).value());

        assertTrue(sql.contains("sentence_due AS"));
        assertTrue(sql.contains("word_due AS"));
        assertTrue(sql.contains("UNION ALL"));
        assertTrue(sql.contains("GROUP BY cloze_item_id"));
        assertTrue(sql.contains("COUNT(*)"));
        assertFalse(sql.contains("LIMIT"));
    }
}
