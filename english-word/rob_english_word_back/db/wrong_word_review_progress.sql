CREATE SEQUENCE IF NOT EXISTS public.wrong_word_review_progress_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE IF NOT EXISTS public.wrong_word_review_progress (
    id bigint PRIMARY KEY DEFAULT nextval('public.wrong_word_review_progress_id_seq'),
    user_id bigint NOT NULL,
    word_id bigint NULL,
    word varchar(100) NOT NULL,
    normalized_word varchar(100) NOT NULL,
    status varchar(32) NOT NULL DEFAULT 'waiting_sentence',
    review_stage int4 NOT NULL DEFAULT 0,
    next_review_time timestamp NULL,
    active_cloze_item_id bigint NULL,
    active_blank_index int4 NULL,
    wrong_count int4 NOT NULL DEFAULT 1,
    first_wrong_time timestamp NOT NULL,
    last_wrong_time timestamp NOT NULL,
    last_answer_record_id bigint NULL,
    completed_time timestamp NULL,
    create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT ck_wrong_word_review_status
        CHECK (status IN ('waiting_sentence', 'due', 'waiting', 'completed')),
    CONSTRAINT ck_wrong_word_review_stage
        CHECK (review_stage BETWEEN 0 AND 3)
);

ALTER SEQUENCE public.wrong_word_review_progress_id_seq
    OWNED BY public.wrong_word_review_progress.id;

CREATE UNIQUE INDEX IF NOT EXISTS uk_wrong_word_review_user_word
    ON public.wrong_word_review_progress(user_id, normalized_word);
CREATE INDEX IF NOT EXISTS idx_wrong_word_review_user_status_time
    ON public.wrong_word_review_progress(user_id, status, next_review_time);
CREATE INDEX IF NOT EXISTS idx_wrong_word_review_active_item
    ON public.wrong_word_review_progress(active_cloze_item_id, active_blank_index);

COMMENT ON TABLE public.wrong_word_review_progress IS '用户错词的单词级三阶段复习进度';
COMMENT ON COLUMN public.wrong_word_review_progress.normalized_word IS '小写并去除首尾空白后的用户内单词唯一键';
COMMENT ON COLUMN public.wrong_word_review_progress.review_stage IS '0立即、1等待7天、2等待15天、3已完成';
COMMENT ON COLUMN public.wrong_word_review_progress.active_blank_index IS '当前复习句子标准答案数组的0基下标';

CREATE OR REPLACE FUNCTION public.update_wrong_word_review_progress_time()
RETURNS TRIGGER AS $$
BEGIN
    NEW.update_time = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_wrong_word_review_progress_update_time
    ON public.wrong_word_review_progress;
CREATE TRIGGER trg_wrong_word_review_progress_update_time
BEFORE UPDATE ON public.wrong_word_review_progress
FOR EACH ROW EXECUTE FUNCTION public.update_wrong_word_review_progress_time();

CREATE OR REPLACE FUNCTION public.wrong_word_review_safe_jsonb_array(raw_value text)
RETURNS jsonb AS $$
DECLARE
    parsed jsonb;
BEGIN
    IF raw_value IS NULL OR BTRIM(raw_value) = '' THEN
        RETURN '[]'::jsonb;
    END IF;

    BEGIN
        parsed := raw_value::jsonb;
    EXCEPTION WHEN OTHERS THEN
        RETURN '[]'::jsonb;
    END;

    IF jsonb_typeof(parsed) <> 'array' THEN
        RETURN '[]'::jsonb;
    END IF;
    RETURN parsed;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

WITH game_wrong_events AS (
    SELECT
        d.user_id,
        d.word_id,
        BTRIM(d.word_content) AS word,
        LOWER(BTRIM(d.word_content)) AS normalized_word,
        d.id AS answer_record_id,
        d.create_time AS wrong_time
    FROM public.game_answer_detail d
    WHERE d.is_correct = 0
      AND d.user_id > 0
      AND d.word_content IS NOT NULL
      AND BTRIM(d.word_content) <> ''
),
cloze_wrong_payload AS (
    SELECT
        r.id,
        r.user_id,
        r.create_time,
        public.wrong_word_review_safe_jsonb_array(r.answers_json) AS answers,
        public.wrong_word_review_safe_jsonb_array(r.expected_words_json) AS expected_words
    FROM public.sentence_cloze_answer_record r
    WHERE r.is_correct = false
      AND r.user_id > 0
),
cloze_wrong_events AS (
    SELECT
        payload.user_id,
        NULL::bigint AS word_id,
        BTRIM(expected.word) AS word,
        LOWER(BTRIM(expected.word)) AS normalized_word,
        payload.id AS answer_record_id,
        payload.create_time AS wrong_time
    FROM cloze_wrong_payload payload
    CROSS JOIN LATERAL jsonb_array_elements_text(payload.expected_words)
        WITH ORDINALITY AS expected(word, ordinal)
    LEFT JOIN LATERAL jsonb_array_elements_text(payload.answers)
        WITH ORDINALITY AS answer(word, ordinal)
        ON answer.ordinal = expected.ordinal
    WHERE BTRIM(expected.word) <> ''
      AND (
          jsonb_array_length(payload.answers) <> jsonb_array_length(payload.expected_words)
          OR LOWER(BTRIM(COALESCE(answer.word, ''))) <> LOWER(BTRIM(expected.word))
      )
),
all_wrong_events AS (
    SELECT * FROM game_wrong_events
    UNION ALL
    SELECT * FROM cloze_wrong_events
),
grouped_wrong_events AS (
    SELECT
        user_id,
        (ARRAY_AGG(word_id ORDER BY wrong_time DESC, answer_record_id DESC)
            FILTER (WHERE word_id IS NOT NULL))[1] AS word_id,
        (ARRAY_AGG(word ORDER BY wrong_time DESC, answer_record_id DESC))[1] AS word,
        normalized_word,
        COUNT(*)::int AS wrong_count,
        MIN(wrong_time) AS first_wrong_time,
        MAX(wrong_time) AS last_wrong_time,
        (ARRAY_AGG(answer_record_id ORDER BY wrong_time DESC, answer_record_id DESC))[1]
            AS last_answer_record_id
    FROM all_wrong_events
    GROUP BY user_id, normalized_word
),
latest_generated_sentence AS (
    SELECT DISTINCT ON (item.user_id, LOWER(BTRIM(blank.word)))
        item.user_id,
        LOWER(BTRIM(blank.word)) AS normalized_word,
        item.id AS cloze_item_id,
        (blank.ordinal - 1)::int AS blank_index
    FROM public.sentence_cloze_item item
    CROSS JOIN LATERAL jsonb_array_elements_text(
        public.wrong_word_review_safe_jsonb_array(item.blank_words_json)
    ) WITH ORDINALITY AS blank(word, ordinal)
    WHERE item.source = 'word-agent'
      AND item.user_id > 0
      AND BTRIM(blank.word) <> ''
    ORDER BY
        item.user_id,
        LOWER(BTRIM(blank.word)),
        item.create_time DESC,
        item.id DESC
)
INSERT INTO public.wrong_word_review_progress (
    user_id,
    word_id,
    word,
    normalized_word,
    status,
    review_stage,
    next_review_time,
    active_cloze_item_id,
    active_blank_index,
    wrong_count,
    first_wrong_time,
    last_wrong_time,
    last_answer_record_id
)
SELECT
    grouped.user_id,
    grouped.word_id,
    grouped.word,
    grouped.normalized_word,
    CASE WHEN generated.cloze_item_id IS NULL THEN 'waiting_sentence' ELSE 'due' END,
    0,
    CASE WHEN generated.cloze_item_id IS NULL THEN NULL ELSE CURRENT_TIMESTAMP END,
    generated.cloze_item_id,
    generated.blank_index,
    grouped.wrong_count,
    grouped.first_wrong_time,
    grouped.last_wrong_time,
    grouped.last_answer_record_id
FROM grouped_wrong_events grouped
LEFT JOIN latest_generated_sentence generated
  ON generated.user_id = grouped.user_id
 AND generated.normalized_word = grouped.normalized_word
ON CONFLICT (user_id, normalized_word) DO NOTHING;
