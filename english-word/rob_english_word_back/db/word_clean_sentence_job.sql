-- Tracks per-word sentence generation jobs for public.word_clean.

CREATE TABLE IF NOT EXISTS public.word_clean_sentence_job (
    id bigserial PRIMARY KEY,
    word_clean_id bigint NOT NULL,
    word varchar(255) NOT NULL,
    model_name varchar(128) NOT NULL,
    status varchar(32) NOT NULL DEFAULT 'pending',
    retry_count integer NOT NULL DEFAULT 0,
    error_message text NOT NULL DEFAULT '',
    locked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

COMMENT ON TABLE public.word_clean_sentence_job IS 'word_clean 大模型造句任务状态表';
COMMENT ON COLUMN public.word_clean_sentence_job.id IS '主键 id';
COMMENT ON COLUMN public.word_clean_sentence_job.word_clean_id IS 'word_clean 表 id';
COMMENT ON COLUMN public.word_clean_sentence_job.word IS '英文单词';
COMMENT ON COLUMN public.word_clean_sentence_job.model_name IS '大模型名称';
COMMENT ON COLUMN public.word_clean_sentence_job.status IS '任务状态: pending/running/success/failed';
COMMENT ON COLUMN public.word_clean_sentence_job.retry_count IS '重试次数';
COMMENT ON COLUMN public.word_clean_sentence_job.error_message IS '最近一次错误信息';
COMMENT ON COLUMN public.word_clean_sentence_job.locked_at IS '任务被 worker 锁定时间';
COMMENT ON COLUMN public.word_clean_sentence_job.created_at IS '创建时间';
COMMENT ON COLUMN public.word_clean_sentence_job.updated_at IS '更新时间';

CREATE UNIQUE INDEX IF NOT EXISTS uk_word_clean_sentence_job_word_model
    ON public.word_clean_sentence_job(word_clean_id, model_name);

CREATE INDEX IF NOT EXISTS idx_word_clean_sentence_job_status
    ON public.word_clean_sentence_job(status);

CREATE INDEX IF NOT EXISTS idx_word_clean_sentence_job_model_status
    ON public.word_clean_sentence_job(model_name, status);

CREATE INDEX IF NOT EXISTS idx_word_clean_sentence_job_word
    ON public.word_clean_sentence_job(word);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_word_clean_sentence_job_word_clean'
    ) THEN
        ALTER TABLE public.word_clean_sentence_job
            ADD CONSTRAINT fk_word_clean_sentence_job_word_clean
            FOREIGN KEY (word_clean_id)
            REFERENCES public.word_clean(id)
            ON DELETE CASCADE;
    END IF;
END $$;
