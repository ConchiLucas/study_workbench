-- Tracks TTS generation jobs for public.word_clean_best_sentence.

CREATE TABLE IF NOT EXISTS public.word_clean_sentence_tts_job (
    id bigserial PRIMARY KEY,
    best_sentence_id bigint NOT NULL,
    word_clean_id bigint NOT NULL,
    word varchar(255) NOT NULL,
    source_sentence_id bigint NOT NULL,
    sentence text NOT NULL,
    sentence_translation text NOT NULL DEFAULT '',
    provider varchar(64) NOT NULL DEFAULT 'mimo',
    tts_model varchar(128) NOT NULL,
    voice varchar(128) NOT NULL,
    audio_format varchar(32) NOT NULL DEFAULT 'wav',
    status varchar(32) NOT NULL DEFAULT 'pending',
    local_file_name varchar(255) NOT NULL DEFAULT '',
    local_file_path text NOT NULL DEFAULT '',
    content_type varchar(128) NOT NULL DEFAULT '',
    file_size bigint NULL,
    duration_ms integer NULL,
    tts_bucket varchar(128) NOT NULL DEFAULT '',
    tts_object_key text NOT NULL DEFAULT '',
    tts_object_url text NOT NULL DEFAULT '',
    retry_count integer NOT NULL DEFAULT 0,
    error_message text NOT NULL DEFAULT '',
    locked_at timestamptz,
    generated_at timestamptz NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

COMMENT ON TABLE public.word_clean_sentence_tts_job IS 'word_clean 最佳例句 TTS 生成任务状态表';
COMMENT ON COLUMN public.word_clean_sentence_tts_job.best_sentence_id IS 'word_clean_best_sentence 表 id';
COMMENT ON COLUMN public.word_clean_sentence_tts_job.word_clean_id IS 'word_clean 表 id';
COMMENT ON COLUMN public.word_clean_sentence_tts_job.source_sentence_id IS '来源 word_clean_sentence 表 id';
COMMENT ON COLUMN public.word_clean_sentence_tts_job.provider IS 'TTS 服务商';
COMMENT ON COLUMN public.word_clean_sentence_tts_job.tts_model IS 'TTS 模型';
COMMENT ON COLUMN public.word_clean_sentence_tts_job.voice IS 'TTS 音色';
COMMENT ON COLUMN public.word_clean_sentence_tts_job.status IS '任务状态: pending/running/success/failed';
COMMENT ON COLUMN public.word_clean_sentence_tts_job.local_file_path IS '本地生成音频路径，MinIO 接入前用于验证';
COMMENT ON COLUMN public.word_clean_sentence_tts_job.tts_bucket IS 'MinIO bucket，后续接入时写入';
COMMENT ON COLUMN public.word_clean_sentence_tts_job.tts_object_key IS 'MinIO object key，后续接入时写入';
COMMENT ON COLUMN public.word_clean_sentence_tts_job.tts_object_url IS 'MinIO 访问地址或签名地址，后续接入时写入';

CREATE UNIQUE INDEX IF NOT EXISTS uk_word_clean_sentence_tts_job_best_provider
    ON public.word_clean_sentence_tts_job(best_sentence_id, provider, tts_model, voice, audio_format);

CREATE INDEX IF NOT EXISTS idx_word_clean_sentence_tts_job_status
    ON public.word_clean_sentence_tts_job(status);

CREATE INDEX IF NOT EXISTS idx_word_clean_sentence_tts_job_provider_status
    ON public.word_clean_sentence_tts_job(provider, status);

CREATE INDEX IF NOT EXISTS idx_word_clean_sentence_tts_job_word
    ON public.word_clean_sentence_tts_job(word);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_word_clean_sentence_tts_job_best_sentence'
    ) THEN
        ALTER TABLE public.word_clean_sentence_tts_job
            ADD CONSTRAINT fk_word_clean_sentence_tts_job_best_sentence
            FOREIGN KEY (best_sentence_id)
            REFERENCES public.word_clean_best_sentence(id)
            ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_word_clean_sentence_tts_job_word_clean'
    ) THEN
        ALTER TABLE public.word_clean_sentence_tts_job
            ADD CONSTRAINT fk_word_clean_sentence_tts_job_word_clean
            FOREIGN KEY (word_clean_id)
            REFERENCES public.word_clean(id)
            ON DELETE CASCADE;
    END IF;
END $$;
