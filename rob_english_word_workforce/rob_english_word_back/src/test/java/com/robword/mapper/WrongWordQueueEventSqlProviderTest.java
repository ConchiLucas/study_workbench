package com.robword.mapper;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

class WrongWordQueueEventSqlProviderTest {

    @Test
    void projectsLatestMetadataForEveryUnfinishedUniqueWord() {
        String sql = WrongWordQueueEventSqlProvider.selectEvents();

        assertTrue(sql.contains("FROM game_answer_detail d"));
        assertTrue(sql.contains("d.is_correct = 0"));
        assertTrue(sql.contains("BTRIM(d.word_content) <> ''"));
        assertFalse(sql.contains("d.word_id IS NOT NULL"));
        assertTrue(sql.contains("WHEN g.id IS NULL THEN '-'"));
        assertTrue(sql.contains("FROM sentence_cloze_answer_record r"));
        assertTrue(sql.contains("jsonb_array_elements_text"));
        assertTrue(sql.contains("WITH ORDINALITY"));
        assertTrue(sql.contains("jsonb_array_length"));
        assertTrue(sql.contains("source_word_ids_json"));
        assertTrue(sql.contains("source_word.ordinal = expected.ordinal"));
        assertTrue(sql.contains("w.id = source_word.word_id::bigint"));
        assertTrue(sql.contains("UNION ALL"));
        assertTrue(sql.contains("FROM wrong_word_review_progress progress"));
        assertTrue(sql.contains("progress.status <> 'completed'"));
        assertTrue(sql.contains("ROW_NUMBER() OVER"));
        assertTrue(sql.contains(
                "PARTITION BY queue_events.user_id, LOWER(BTRIM(queue_events.word))"));
        assertTrue(sql.contains("progress.wrong_count AS occurrence_count"));
        assertTrue(sql.contains("progress.id"));
        assertTrue(sql.contains("CAST(#{keyword} AS text) IS NULL"));
        assertFalse(sql.contains("WHERE (#{keyword} IS NULL"));
        assertTrue(sql.contains("word_clean_best_sentence"));
        assertTrue(sql.contains("best_example.sentence AS example_sentence"));
        assertTrue(sql.contains("WHEN best_example.sentence IS NOT NULL THEN 'best_sentence'"));
        assertTrue(sql.contains("ELSE 'none'"));
        assertFalse(sql.contains("dictionary_example"));
        assertFalse(sql.contains("FROM word exact_word"));
        assertFalse(sql.contains("FROM word fallback_word"));
        assertFalse(sql.contains("THEN 'word'"));
        assertTrue(sql.contains("LIMIT #{size} OFFSET #{offset}"));
    }

    @Test
    void countUsesTheSameUnfinishedUniqueWordSet() {
        String sql = WrongWordQueueEventSqlProvider.countEvents();

        assertTrue(sql.contains("wrong_word_review_progress"));
        assertTrue(sql.contains("status <> 'completed'"));
        assertTrue(sql.contains("COUNT(*)"));
        assertTrue(sql.contains("CAST(#{keyword} AS text) IS NULL"));
        assertFalse(sql.contains("#{keyword} IS NULL"));
        assertFalse(sql.contains("COUNT(*) FROM ranked_events"));
    }
}
