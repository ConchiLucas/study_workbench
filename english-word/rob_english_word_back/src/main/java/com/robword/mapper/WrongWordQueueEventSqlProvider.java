package com.robword.mapper;

public final class WrongWordQueueEventSqlProvider {

    private static final String EVENT_CTES = """
            WITH game_events AS (
                SELECT
                    d.user_id,
                    CONCAT('game:', d.id::text) AS event_key,
                    BTRIM(d.word_content) AS word,
                    d.create_time AS answered_at,
                    '游戏答题'::text AS entry,
                    CASE COALESCE(g.mode, '')
                        WHEN 'match' THEN '正式匹配'
                        WHEN 'solo_training' THEN '单人训练'
                        ELSE NULLIF(g.mode, '')
                    END AS mode,
                    CASE
                        WHEN g.mode = 'solo_training' THEN COALESCE(g.training_difficulty_group, '')
                        ELSE COALESCE(g.match_difficulty_group, '')
                    END AS difficulty_group,
                    CASE
                        WHEN g.mode = 'solo_training' THEN COALESCE(g.training_difficulty_level, '')
                        ELSE COALESCE(g.match_difficulty_level, '')
                    END AS difficulty_level,
                    CASE
                        WHEN g.id IS NULL THEN '-'
                        WHEN g.mode = 'solo_training' THEN COALESCE(
                            NULLIF(g.training_difficulty_level, ''),
                            NULLIF(g.training_difficulty_group, ''),
                            '-'
                        )
                        ELSE COALESCE(
                            NULLIF(g.match_difficulty_label, ''),
                            NULLIF(g.match_difficulty_level, ''),
                            NULLIF(g.match_difficulty_group, ''),
                            '段位难度'
                        )
                    END AS difficulty_label,
                    d.word_difficulty,
                    d.answer_time_ms::bigint AS cost_ms,
                    CASE d.correct_answer_index
                        WHEN 1 THEN d.option_1
                        WHEN 2 THEN d.option_2
                        WHEN 3 THEN d.option_3
                        WHEN 4 THEN d.option_4
                        ELSE NULL
                    END AS correct_answer,
                    'game'::text AS source_type
                FROM game_answer_detail d
                LEFT JOIN game_record g ON g.id = d.record_id
                WHERE d.user_id = #{userId}
                  AND d.is_correct = 0
                  AND d.word_content IS NOT NULL
                  AND BTRIM(d.word_content) <> ''
            ),
            cloze_payload AS (
                SELECT
                    r.id,
                    r.user_id,
                    r.attempt_no,
                    r.cost_ms,
                    r.create_time,
                    COALESCE(NULLIF(r.answers_json, ''), '[]')::jsonb AS answers,
                    COALESCE(NULLIF(r.expected_words_json, ''), '[]')::jsonb AS expected_words,
                    i.source,
                    i.provider_label,
                    COALESCE(NULLIF(i.source_word_ids_json, ''), '[]')::jsonb AS source_word_ids
                FROM sentence_cloze_answer_record r
                JOIN sentence_cloze_item i ON i.id = r.cloze_item_id
                WHERE r.user_id = #{userId}
                  AND r.is_correct = false
            ),
            cloze_events AS (
                SELECT
                    p.user_id,
                    CONCAT('cloze:', p.id::text, ':', expected.ordinal::text) AS event_key,
                    BTRIM(expected.word) AS word,
                    p.create_time AS answered_at,
                    '挖空答题'::text AS entry,
                    CASE
                        WHEN p.source = 'best-sentence-practice' THEN '单独训练'
                        WHEN COALESCE(p.attempt_no, 1) > 1 THEN '错题复习'
                        ELSE '待练句子'
                    END AS mode,
                    ''::text AS difficulty_group,
                    ''::text AS difficulty_level,
                    CASE
                        WHEN p.source = 'best-sentence-practice' THEN COALESCE(
                            NULLIF(p.provider_label, ''),
                            '单独训练'
                        )
                        ELSE '外部错题造句'
                    END AS difficulty_label,
                    (
                        SELECT w.difficulty
                        FROM word w
                        WHERE (
                            p.source = 'word-agent'
                            AND source_word.word_id ~ '^[0-9]+$'
                            AND w.id = source_word.word_id::bigint
                        )
                           OR LOWER(BTRIM(w.word)) = LOWER(BTRIM(expected.word))
                        ORDER BY
                            CASE
                                WHEN p.source = 'word-agent'
                                 AND source_word.word_id ~ '^[0-9]+$'
                                 AND w.id = source_word.word_id::bigint
                                THEN 0
                                ELSE 1
                            END,
                            CASE WHEN w.status = 1 THEN 0 ELSE 1 END,
                            w.id
                        LIMIT 1
                    ) AS word_difficulty,
                    p.cost_ms::bigint AS cost_ms,
                    BTRIM(expected.word) AS correct_answer,
                    'cloze'::text AS source_type
                FROM cloze_payload p
                CROSS JOIN LATERAL jsonb_array_elements_text(p.expected_words)
                    WITH ORDINALITY AS expected(word, ordinal)
                LEFT JOIN LATERAL jsonb_array_elements_text(p.answers)
                    WITH ORDINALITY AS answer(word, ordinal)
                    ON answer.ordinal = expected.ordinal
                LEFT JOIN LATERAL jsonb_array_elements_text(p.source_word_ids)
                    WITH ORDINALITY AS source_word(word_id, ordinal)
                    ON source_word.ordinal = expected.ordinal
                WHERE BTRIM(expected.word) <> ''
                  AND (
                      jsonb_array_length(p.answers) <> jsonb_array_length(p.expected_words)
                      OR LOWER(BTRIM(COALESCE(answer.word, ''))) <> LOWER(BTRIM(expected.word))
                  )
            ),
            queue_events AS (
                SELECT * FROM game_events
                UNION ALL
                SELECT * FROM cloze_events
            ),
            latest_events AS (
                SELECT
                    queue_events.*,
                    ROW_NUMBER() OVER (
                        PARTITION BY queue_events.user_id, LOWER(BTRIM(queue_events.word))
                        ORDER BY queue_events.answered_at DESC, queue_events.event_key DESC
                    ) AS row_no
                FROM queue_events
            ),
            active_words AS (
                SELECT
                    CONCAT('progress:', progress.id::text) AS event_key,
                    CONCAT('progress:', progress.id::text) AS progress_key,
                    progress.word,
                    progress.last_wrong_time AS answered_at,
                    latest.entry,
                    latest.mode,
                    latest.difficulty_group,
                    latest.difficulty_level,
                    latest.difficulty_label,
                    latest.word_difficulty,
                    latest.cost_ms,
                    latest.correct_answer,
                    best_example.sentence AS example_sentence,
                    CASE
                        WHEN best_example.sentence IS NOT NULL THEN 'best_sentence'
                        ELSE 'none'
                    END AS example_source,
                    latest.source_type,
                    progress.wrong_count AS occurrence_count,
                    progress.status AS review_status,
                    progress.review_stage,
                    progress.next_review_time,
                    progress.id
                FROM wrong_word_review_progress progress
                LEFT JOIN latest_events latest
                  ON latest.user_id = progress.user_id
                 AND LOWER(BTRIM(latest.word)) = progress.normalized_word
                 AND latest.row_no = 1
                LEFT JOIN LATERAL (
                    SELECT NULLIF(BTRIM(best.sentence), '') AS sentence
                    FROM word_clean_best_sentence best
                    WHERE LOWER(BTRIM(best.word)) = progress.normalized_word
                      AND NULLIF(BTRIM(best.sentence), '') IS NOT NULL
                    ORDER BY best.id
                    LIMIT 1
                ) best_example ON true
                WHERE progress.user_id = #{userId}
                  AND progress.status <> 'completed'
            )
            """;

    private WrongWordQueueEventSqlProvider() {
    }

    public static String selectEvents() {
        return EVENT_CTES + """
                SELECT
                    event_key,
                    progress_key,
                    word,
                    answered_at,
                    entry,
                    mode,
                    difficulty_group,
                    difficulty_level,
                    difficulty_label,
                    word_difficulty,
                    cost_ms,
                    correct_answer,
                    example_sentence,
                    example_source,
                    source_type,
                    occurrence_count,
                    review_status,
                    review_stage,
                    next_review_time
                FROM active_words
                WHERE (
                    CAST(#{keyword} AS text) IS NULL
                    OR word ILIKE CONCAT('%', CAST(#{keyword} AS text), '%')
                )
                ORDER BY
                    CASE WHEN #{sort} = 'count' THEN occurrence_count END DESC,
                    answered_at DESC,
                    event_key DESC
                LIMIT #{size} OFFSET #{offset}
                """;
    }

    public static String countEvents() {
        return """
                SELECT COUNT(*)
                FROM wrong_word_review_progress
                WHERE user_id = #{userId}
                  AND status <> 'completed'
                  AND (
                      CAST(#{keyword} AS text) IS NULL
                      OR word ILIKE CONCAT('%', CAST(#{keyword} AS text), '%')
                  )
                """;
    }
}
