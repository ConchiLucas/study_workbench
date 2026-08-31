CREATE SEQUENCE IF NOT EXISTS public.sentence_cloze_review_schedule_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

COMMENT ON SEQUENCE public.sentence_cloze_review_schedule_id_seq IS '挖空句子错题复习计划表id自增序列';

CREATE TABLE IF NOT EXISTS public.sentence_cloze_review_schedule (
    id bigint PRIMARY KEY DEFAULT nextval('public.sentence_cloze_review_schedule_id_seq'),
    user_id bigint NOT NULL,
    cloze_item_id bigint NOT NULL,
    correct_streak int4 NOT NULL DEFAULT 0,
    review_stage int4 NOT NULL DEFAULT 0,
    status varchar(32) NOT NULL DEFAULT 'active',
    next_review_time timestamp NULL,
    wrong_count int4 NOT NULL DEFAULT 1,
    first_wrong_time timestamp NULL,
    last_answer_record_id bigint NULL,
    last_wrong_answer_record_id bigint NULL,
    last_wrong_time timestamp NULL,
    last_correct_time timestamp NULL,
    completed_time timestamp NULL,
    create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE public.sentence_cloze_review_schedule
    ADD COLUMN IF NOT EXISTS user_id bigint NOT NULL DEFAULT 0;
ALTER TABLE public.sentence_cloze_review_schedule
    ADD COLUMN IF NOT EXISTS cloze_item_id bigint NOT NULL DEFAULT 0;
ALTER TABLE public.sentence_cloze_review_schedule
    ADD COLUMN IF NOT EXISTS correct_streak int4 NOT NULL DEFAULT 0;
ALTER TABLE public.sentence_cloze_review_schedule
    ADD COLUMN IF NOT EXISTS review_stage int4 NOT NULL DEFAULT 0;
ALTER TABLE public.sentence_cloze_review_schedule
    ADD COLUMN IF NOT EXISTS status varchar(32) NOT NULL DEFAULT 'active';
ALTER TABLE public.sentence_cloze_review_schedule
    ADD COLUMN IF NOT EXISTS next_review_time timestamp NULL;
ALTER TABLE public.sentence_cloze_review_schedule
    ALTER COLUMN next_review_time DROP NOT NULL;
ALTER TABLE public.sentence_cloze_review_schedule
    ADD COLUMN IF NOT EXISTS wrong_count int4 NOT NULL DEFAULT 1;
ALTER TABLE public.sentence_cloze_review_schedule
    ADD COLUMN IF NOT EXISTS first_wrong_time timestamp NULL;
ALTER TABLE public.sentence_cloze_review_schedule
    ADD COLUMN IF NOT EXISTS last_answer_record_id bigint NULL;
ALTER TABLE public.sentence_cloze_review_schedule
    ADD COLUMN IF NOT EXISTS last_wrong_answer_record_id bigint NULL;
ALTER TABLE public.sentence_cloze_review_schedule
    ADD COLUMN IF NOT EXISTS last_wrong_time timestamp NULL;
ALTER TABLE public.sentence_cloze_review_schedule
    ADD COLUMN IF NOT EXISTS last_correct_time timestamp NULL;
ALTER TABLE public.sentence_cloze_review_schedule
    ADD COLUMN IF NOT EXISTS completed_time timestamp NULL;
ALTER TABLE public.sentence_cloze_review_schedule
    ADD COLUMN IF NOT EXISTS create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE public.sentence_cloze_review_schedule
    ADD COLUMN IF NOT EXISTS update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP;

COMMENT ON TABLE public.sentence_cloze_review_schedule IS '挖空句子错题复习计划表';
COMMENT ON COLUMN public.sentence_cloze_review_schedule.id IS '复习计划ID';
COMMENT ON COLUMN public.sentence_cloze_review_schedule.user_id IS '用户ID';
COMMENT ON COLUMN public.sentence_cloze_review_schedule.cloze_item_id IS '挖空题内容ID';
COMMENT ON COLUMN public.sentence_cloze_review_schedule.correct_streak IS '进入复习后连续答对次数';
COMMENT ON COLUMN public.sentence_cloze_review_schedule.review_stage IS '复习阶段：0立即，1七天，2十五天，3已完成';
COMMENT ON COLUMN public.sentence_cloze_review_schedule.status IS '复习状态：active或completed';
COMMENT ON COLUMN public.sentence_cloze_review_schedule.next_review_time IS '下次可复习时间';
COMMENT ON COLUMN public.sentence_cloze_review_schedule.wrong_count IS '整句错误提交次数';
COMMENT ON COLUMN public.sentence_cloze_review_schedule.first_wrong_time IS '首次答错时间';
COMMENT ON COLUMN public.sentence_cloze_review_schedule.last_answer_record_id IS '最近一次答题记录ID';
COMMENT ON COLUMN public.sentence_cloze_review_schedule.last_wrong_answer_record_id IS '最近一次错误答题记录ID';
COMMENT ON COLUMN public.sentence_cloze_review_schedule.last_wrong_time IS '最近一次答错时间';
COMMENT ON COLUMN public.sentence_cloze_review_schedule.last_correct_time IS '最近一次答对时间';
COMMENT ON COLUMN public.sentence_cloze_review_schedule.completed_time IS '完成三阶段复习时间';
COMMENT ON COLUMN public.sentence_cloze_review_schedule.create_time IS '创建时间';
COMMENT ON COLUMN public.sentence_cloze_review_schedule.update_time IS '更新时间';

ALTER SEQUENCE public.sentence_cloze_review_schedule_id_seq
    OWNED BY public.sentence_cloze_review_schedule.id;

CREATE UNIQUE INDEX IF NOT EXISTS uk_sentence_cloze_review_user_item
    ON public.sentence_cloze_review_schedule(user_id, cloze_item_id);
CREATE INDEX IF NOT EXISTS idx_sentence_cloze_review_user_next_time
    ON public.sentence_cloze_review_schedule(user_id, next_review_time);
CREATE INDEX IF NOT EXISTS idx_sentence_cloze_review_user_status_time
    ON public.sentence_cloze_review_schedule(user_id, status, next_review_time);
CREATE INDEX IF NOT EXISTS idx_sentence_cloze_review_user_last_wrong
    ON public.sentence_cloze_review_schedule(user_id, last_wrong_time DESC);
CREATE INDEX IF NOT EXISTS idx_sentence_cloze_review_item
    ON public.sentence_cloze_review_schedule(cloze_item_id);

ALTER TABLE public.sentence_cloze_review_schedule
    DROP CONSTRAINT IF EXISTS ck_sentence_cloze_review_status;
ALTER TABLE public.sentence_cloze_review_schedule
    ADD CONSTRAINT ck_sentence_cloze_review_status
    CHECK (status IN ('active', 'completed'));
ALTER TABLE public.sentence_cloze_review_schedule
    DROP CONSTRAINT IF EXISTS ck_sentence_cloze_review_stage;
ALTER TABLE public.sentence_cloze_review_schedule
    ADD CONSTRAINT ck_sentence_cloze_review_stage
    CHECK (review_stage BETWEEN 0 AND 3);

CREATE OR REPLACE FUNCTION public.update_sentence_cloze_review_schedule_time()
RETURNS TRIGGER AS $$
BEGIN
    NEW.update_time = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_sentence_cloze_review_schedule_update_time
    ON public.sentence_cloze_review_schedule;
CREATE TRIGGER trg_sentence_cloze_review_schedule_update_time
BEFORE UPDATE ON public.sentence_cloze_review_schedule
FOR EACH ROW EXECUTE FUNCTION public.update_sentence_cloze_review_schedule_time();

WITH answer_state AS (
    SELECT
        r.user_id,
        r.cloze_item_id,
        COUNT(*) FILTER (WHERE r.is_correct = false)::int4 AS wrong_count,
        MIN(r.create_time) FILTER (WHERE r.is_correct = false) AS first_wrong_time,
        (ARRAY_AGG(r.id ORDER BY r.create_time DESC, r.id DESC) FILTER (WHERE r.is_correct = false))[1] AS last_wrong_id,
        (ARRAY_AGG(r.create_time ORDER BY r.create_time DESC, r.id DESC) FILTER (WHERE r.is_correct = false))[1] AS last_wrong_time
    FROM public.sentence_cloze_answer_record r
    GROUP BY r.user_id, r.cloze_item_id
)
UPDATE public.sentence_cloze_review_schedule schedule
SET wrong_count = GREATEST(schedule.wrong_count, answer_state.wrong_count),
    first_wrong_time = COALESCE(
        LEAST(schedule.first_wrong_time, answer_state.first_wrong_time),
        schedule.first_wrong_time,
        answer_state.first_wrong_time
    ),
    last_wrong_answer_record_id = CASE
        WHEN schedule.last_wrong_time IS NULL
          OR answer_state.last_wrong_time >= schedule.last_wrong_time
            THEN answer_state.last_wrong_id
        ELSE schedule.last_wrong_answer_record_id
    END,
    last_wrong_time = COALESCE(
        GREATEST(schedule.last_wrong_time, answer_state.last_wrong_time),
        schedule.last_wrong_time,
        answer_state.last_wrong_time
    )
FROM answer_state
WHERE schedule.user_id = answer_state.user_id
  AND schedule.cloze_item_id = answer_state.cloze_item_id
  AND answer_state.last_wrong_id IS NOT NULL
  AND (
      schedule.wrong_count < answer_state.wrong_count
      OR schedule.first_wrong_time IS NULL
      OR answer_state.first_wrong_time < schedule.first_wrong_time
      OR schedule.last_wrong_time IS NULL
      OR answer_state.last_wrong_time > schedule.last_wrong_time
      OR (
          answer_state.last_wrong_time = schedule.last_wrong_time
          AND schedule.last_wrong_answer_record_id IS DISTINCT FROM answer_state.last_wrong_id
      )
  );

WITH answer_state AS (
    SELECT
        r.user_id,
        r.cloze_item_id,
        (ARRAY_AGG(r.id ORDER BY r.create_time DESC, r.id DESC))[1] AS latest_answer_record_id,
        (ARRAY_AGG(r.is_correct ORDER BY r.create_time DESC, r.id DESC))[1] AS latest_is_correct,
        COUNT(*) FILTER (WHERE r.is_correct = false)::int4 AS wrong_count,
        MIN(r.create_time) FILTER (WHERE r.is_correct = false) AS first_wrong_time,
        (ARRAY_AGG(r.id ORDER BY r.create_time DESC, r.id DESC) FILTER (WHERE r.is_correct = false))[1] AS last_wrong_id,
        (ARRAY_AGG(r.create_time ORDER BY r.create_time DESC, r.id DESC) FILTER (WHERE r.is_correct = false))[1] AS last_wrong_time
    FROM public.sentence_cloze_answer_record r
    GROUP BY r.user_id, r.cloze_item_id
)
INSERT INTO public.sentence_cloze_review_schedule (
    user_id,
    cloze_item_id,
    correct_streak,
    review_stage,
    status,
    next_review_time,
    wrong_count,
    first_wrong_time,
    last_answer_record_id,
    last_wrong_answer_record_id,
    last_wrong_time,
    last_correct_time,
    completed_time
)
SELECT
    user_id,
    cloze_item_id,
    0,
    0,
    'active',
    last_wrong_time,
    wrong_count,
    first_wrong_time,
    latest_answer_record_id,
    last_wrong_id,
    last_wrong_time,
    NULL,
    NULL
FROM answer_state
WHERE latest_is_correct = false
  AND last_wrong_id IS NOT NULL
ON CONFLICT (user_id, cloze_item_id) DO NOTHING;
