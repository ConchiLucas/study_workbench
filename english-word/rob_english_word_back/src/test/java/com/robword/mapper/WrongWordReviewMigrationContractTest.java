package com.robword.mapper;

import org.junit.jupiter.api.Test;

import java.nio.file.Files;
import java.nio.file.Path;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

class WrongWordReviewMigrationContractTest {

    @Test
    void historicalJsonFieldsAreParsedAsArraysWithoutAbortingTheMigration() throws Exception {
        String sql = Files.readString(Path.of("db/wrong_word_review_progress.sql"));

        assertTrue(sql.contains("wrong_word_review_safe_jsonb_array"));
        assertTrue(sql.contains(
                "wrong_word_review_safe_jsonb_array(r.answers_json) AS answers"));
        assertTrue(sql.contains(
                "wrong_word_review_safe_jsonb_array(r.expected_words_json) AS expected_words"));
        assertTrue(sql.contains(
                "wrong_word_review_safe_jsonb_array(item.blank_words_json)"));
        assertTrue(sql.contains("IF jsonb_typeof(parsed) <> 'array'"));
        assertTrue(sql.contains("EXCEPTION WHEN OTHERS"));
        assertFalse(sql.contains("DROP FUNCTION IF EXISTS public.wrong_word_review_safe_jsonb_array"));
        assertFalse(sql.contains("COALESCE(NULLIF(r.answers_json, ''), '[]')::jsonb"));
    }
}
