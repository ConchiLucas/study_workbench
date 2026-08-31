-- Tracks per-word sentence scoring jobs for public.word_clean_sentence.

CREATE TABLE IF NOT EXISTS public.word_clean_sentence_score_job (
    id bigserial PRIMARY KEY,
    word_clean_id bigint NOT NULL,
    word varchar(255) NOT NULL,
    judge_model varchar(128) NOT NULL,
    status varchar(32) NOT NULL DEFAULT 'pending',
    candidate_count integer NOT NULL DEFAULT 0,
    scored_count integer NOT NULL DEFAULT 0,
    best_sentence_id bigint NULL,
    best_score integer NULL,
    best_source_model_name varchar(160) NOT NULL DEFAULT '',
    retry_count integer NOT NULL DEFAULT 0,
    error_message text NOT NULL DEFAULT '',
    locked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

COMMENT ON TABLE public.word_clean_sentence_score_job IS 'word_clean 大模型造句评分任务状态表';
COMMENT ON COLUMN public.word_clean_sentence_score_job.word_clean_id IS 'word_clean 表 id';
COMMENT ON COLUMN public.word_clean_sentence_score_job.word IS '英文单词';
COMMENT ON COLUMN public.word_clean_sentence_score_job.judge_model IS '执行评分的裁判模型';
COMMENT ON COLUMN public.word_clean_sentence_score_job.status IS '任务状态: pending/running/success/failed';
COMMENT ON COLUMN public.word_clean_sentence_score_job.candidate_count IS '该单词可评分候选句数量';
COMMENT ON COLUMN public.word_clean_sentence_score_job.scored_count IS '该单词已评分候选句数量';
COMMENT ON COLUMN public.word_clean_sentence_score_job.best_sentence_id IS '评分后落表的最佳句子 id';
COMMENT ON COLUMN public.word_clean_sentence_score_job.best_score IS '最佳句子评分';
COMMENT ON COLUMN public.word_clean_sentence_score_job.best_source_model_name IS '最佳句子来源造句模型';
COMMENT ON COLUMN public.word_clean_sentence_score_job.retry_count IS '重试次数';
COMMENT ON COLUMN public.word_clean_sentence_score_job.error_message IS '最近一次错误信息';
COMMENT ON COLUMN public.word_clean_sentence_score_job.locked_at IS '任务被 worker 锁定时间';

CREATE UNIQUE INDEX IF NOT EXISTS uk_word_clean_sentence_score_job_word_model
    ON public.word_clean_sentence_score_job(word_clean_id, judge_model);

CREATE INDEX IF NOT EXISTS idx_word_clean_sentence_score_job_status
    ON public.word_clean_sentence_score_job(status);

CREATE INDEX IF NOT EXISTS idx_word_clean_sentence_score_job_model_status
    ON public.word_clean_sentence_score_job(judge_model, status);

CREATE INDEX IF NOT EXISTS idx_word_clean_sentence_score_job_word
    ON public.word_clean_sentence_score_job(word);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_word_clean_sentence_score_job_word_clean'
    ) THEN
        ALTER TABLE public.word_clean_sentence_score_job
            ADD CONSTRAINT fk_word_clean_sentence_score_job_word_clean
            FOREIGN KEY (word_clean_id)
            REFERENCES public.word_clean(id)
            ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_word_clean_sentence_score_job_best_sentence'
    ) THEN
        ALTER TABLE public.word_clean_sentence_score_job
            ADD CONSTRAINT fk_word_clean_sentence_score_job_best_sentence
            FOREIGN KEY (best_sentence_id)
            REFERENCES public.word_clean_best_sentence(id)
            ON DELETE SET NULL;
    END IF;
END $$;
