CREATE SEQUENCE IF NOT EXISTS public.sentence_cloze_answer_record_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

COMMENT ON SEQUENCE public.sentence_cloze_answer_record_id_seq IS '挖空练习答题记录表id自增序列';

CREATE TABLE IF NOT EXISTS public.sentence_cloze_answer_record (
    id bigint PRIMARY KEY DEFAULT nextval('public.sentence_cloze_answer_record_id_seq'),
    user_id bigint NOT NULL,
    user_name varchar(100) NULL,
    cloze_item_id bigint NOT NULL,
    answer_text text NOT NULL DEFAULT '',
    answers_json text NOT NULL DEFAULT '[]',
    expected_words_json text NOT NULL DEFAULT '[]',
    submission_key varchar(64) NULL,
    practice_context varchar(16) NULL,
    action_type varchar(16) NULL,
    wrong_blank_indexes_json text NOT NULL DEFAULT '[]',
    is_correct boolean NOT NULL DEFAULT false,
    attempt_no int4 NOT NULL DEFAULT 1,
    cost_ms bigint NULL,
    create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE public.sentence_cloze_answer_record
    ADD COLUMN IF NOT EXISTS user_id bigint NOT NULL DEFAULT 0;
ALTER TABLE public.sentence_cloze_answer_record
    ADD COLUMN IF NOT EXISTS user_name varchar(100) NULL;
ALTER TABLE public.sentence_cloze_answer_record
    ADD COLUMN IF NOT EXISTS cloze_item_id bigint NOT NULL DEFAULT 0;
ALTER TABLE public.sentence_cloze_answer_record
    ADD COLUMN IF NOT EXISTS answer_text text NOT NULL DEFAULT '';
ALTER TABLE public.sentence_cloze_answer_record
    ADD COLUMN IF NOT EXISTS answers_json text NOT NULL DEFAULT '[]';
ALTER TABLE public.sentence_cloze_answer_record
    ADD COLUMN IF NOT EXISTS expected_words_json text NOT NULL DEFAULT '[]';
ALTER TABLE public.sentence_cloze_answer_record
    ADD COLUMN IF NOT EXISTS submission_key varchar(64) NULL;
ALTER TABLE public.sentence_cloze_answer_record
    ADD COLUMN IF NOT EXISTS practice_context varchar(16) NULL;
ALTER TABLE public.sentence_cloze_answer_record
    ADD COLUMN IF NOT EXISTS action_type varchar(16) NULL;
ALTER TABLE public.sentence_cloze_answer_record
    ADD COLUMN IF NOT EXISTS wrong_blank_indexes_json text NOT NULL DEFAULT '[]';
ALTER TABLE public.sentence_cloze_answer_record
    ADD COLUMN IF NOT EXISTS is_correct boolean NOT NULL DEFAULT false;
ALTER TABLE public.sentence_cloze_answer_record
    ADD COLUMN IF NOT EXISTS attempt_no int4 NOT NULL DEFAULT 1;
ALTER TABLE public.sentence_cloze_answer_record
    ADD COLUMN IF NOT EXISTS cost_ms bigint NULL;
ALTER TABLE public.sentence_cloze_answer_record
    ADD COLUMN IF NOT EXISTS create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE public.sentence_cloze_answer_record
    ADD COLUMN IF NOT EXISTS update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP;

COMMENT ON TABLE public.sentence_cloze_answer_record IS '挖空练习答题记录表';
COMMENT ON COLUMN public.sentence_cloze_answer_record.id IS '答题记录ID';
COMMENT ON COLUMN public.sentence_cloze_answer_record.user_id IS '答题用户ID';
COMMENT ON COLUMN public.sentence_cloze_answer_record.user_name IS '答题用户名快照';
COMMENT ON COLUMN public.sentence_cloze_answer_record.cloze_item_id IS '挖空题内容ID';
COMMENT ON COLUMN public.sentence_cloze_answer_record.answer_text IS '用户原始答案文本';
COMMENT ON COLUMN public.sentence_cloze_answer_record.answers_json IS '用户答案JSON';
COMMENT ON COLUMN public.sentence_cloze_answer_record.expected_words_json IS '期望答案JSON';
COMMENT ON COLUMN public.sentence_cloze_answer_record.submission_key IS '客户端提交幂等键';
COMMENT ON COLUMN public.sentence_cloze_answer_record.practice_context IS '实际答题入口：review或solo';
COMMENT ON COLUMN public.sentence_cloze_answer_record.action_type IS '提交动作：answer或reveal';
COMMENT ON COLUMN public.sentence_cloze_answer_record.wrong_blank_indexes_json IS '错误空位下标JSON';
COMMENT ON COLUMN public.sentence_cloze_answer_record.is_correct IS '是否答对';
COMMENT ON COLUMN public.sentence_cloze_answer_record.attempt_no IS '该用户对该题第几次作答';
COMMENT ON COLUMN public.sentence_cloze_answer_record.cost_ms IS '答题耗时毫秒';
COMMENT ON COLUMN public.sentence_cloze_answer_record.create_time IS '创建时间';
COMMENT ON COLUMN public.sentence_cloze_answer_record.update_time IS '更新时间';

ALTER SEQUENCE public.sentence_cloze_answer_record_id_seq
    OWNED BY public.sentence_cloze_answer_record.id;

CREATE INDEX IF NOT EXISTS idx_sentence_cloze_answer_user_time
    ON public.sentence_cloze_answer_record(user_id, create_time);
CREATE INDEX IF NOT EXISTS idx_sentence_cloze_answer_user_item
    ON public.sentence_cloze_answer_record(user_id, cloze_item_id);
CREATE INDEX IF NOT EXISTS idx_sentence_cloze_answer_user_correct
    ON public.sentence_cloze_answer_record(user_id, is_correct);
CREATE UNIQUE INDEX IF NOT EXISTS uk_sentence_cloze_answer_user_submission
    ON public.sentence_cloze_answer_record(user_id, submission_key)
    WHERE submission_key IS NOT NULL;

CREATE OR REPLACE FUNCTION public.update_sentence_cloze_answer_record_time()
RETURNS TRIGGER AS $$
BEGIN
    NEW.update_time = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_sentence_cloze_answer_record_update_time
    ON public.sentence_cloze_answer_record;
CREATE TRIGGER trg_sentence_cloze_answer_record_update_time
BEFORE UPDATE ON public.sentence_cloze_answer_record
FOR EACH ROW EXECUTE FUNCTION public.update_sentence_cloze_answer_record_time();
