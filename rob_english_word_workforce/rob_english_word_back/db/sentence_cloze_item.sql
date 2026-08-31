CREATE SEQUENCE IF NOT EXISTS public.sentence_cloze_item_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

COMMENT ON SEQUENCE public.sentence_cloze_item_id_seq IS '挖空造句内容表id自增序列';

CREATE TABLE IF NOT EXISTS public.sentence_cloze_item (
    id bigint PRIMARY KEY DEFAULT nextval('public.sentence_cloze_item_id_seq'),
    user_id bigint NULL,
    user_name varchar(100) NULL,
    word varchar(100) NOT NULL,
    words_json text NOT NULL,
    blank_words_json text NOT NULL,
    sentence text NOT NULL,
    best_sentence_id bigint NULL,
    sentence_audio_url text NOT NULL DEFAULT '',
    translation_zh text NOT NULL,
    explanation_zh text NULL,
    cloze_sentence text NOT NULL,
    provider_id varchar(120) NULL,
    provider_label varchar(160) NULL,
    model varchar(160) NULL,
    source varchar(64) DEFAULT 'word-agent' NOT NULL,
    source_event_ids_json text NOT NULL DEFAULT '[]',
    source_answer_detail_ids_json text NOT NULL DEFAULT '[]',
    source_record_ids_json text NOT NULL DEFAULT '[]',
    source_word_ids_json text NOT NULL DEFAULT '[]',
    generation_key varchar(160) NULL,
    create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS user_id bigint NULL;
ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS user_name varchar(100) NULL;
ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS word varchar(100) NOT NULL DEFAULT '';
ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS words_json text NOT NULL DEFAULT '[]';
ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS blank_words_json text NOT NULL DEFAULT '[]';
ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS sentence text NOT NULL DEFAULT '';
ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS best_sentence_id bigint NULL;
ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS sentence_audio_url text NOT NULL DEFAULT '';
ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS translation_zh text NOT NULL DEFAULT '';
ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS explanation_zh text NULL;
ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS cloze_sentence text NOT NULL DEFAULT '';
ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS provider_id varchar(120) NULL;
ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS provider_label varchar(160) NULL;
ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS model varchar(160) NULL;
ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS source varchar(64) DEFAULT 'word-agent' NOT NULL;
ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS source_event_ids_json text NOT NULL DEFAULT '[]';
ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS source_answer_detail_ids_json text NOT NULL DEFAULT '[]';
ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS source_record_ids_json text NOT NULL DEFAULT '[]';
ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS source_word_ids_json text NOT NULL DEFAULT '[]';
ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS generation_key varchar(160) NULL;
ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE public.sentence_cloze_item
    ADD COLUMN IF NOT EXISTS update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP;

COMMENT ON TABLE public.sentence_cloze_item IS '挖空造句内容表';
COMMENT ON COLUMN public.sentence_cloze_item.id IS '内容ID';
COMMENT ON COLUMN public.sentence_cloze_item.user_id IS '归属用户ID';
COMMENT ON COLUMN public.sentence_cloze_item.user_name IS '归属用户名快照';
COMMENT ON COLUMN public.sentence_cloze_item.word IS '主挖空单词';
COMMENT ON COLUMN public.sentence_cloze_item.words_json IS '请求单词JSON';
COMMENT ON COLUMN public.sentence_cloze_item.blank_words_json IS '需要挖空的单词JSON';
COMMENT ON COLUMN public.sentence_cloze_item.sentence IS '生成的英文句子';
COMMENT ON COLUMN public.sentence_cloze_item.best_sentence_id IS '来源最佳例句ID';
COMMENT ON COLUMN public.sentence_cloze_item.sentence_audio_url IS '最佳例句TTS音频地址';
COMMENT ON COLUMN public.sentence_cloze_item.translation_zh IS '句子中文翻译';
COMMENT ON COLUMN public.sentence_cloze_item.explanation_zh IS '中文解释';
COMMENT ON COLUMN public.sentence_cloze_item.cloze_sentence IS '挖空句子';
COMMENT ON COLUMN public.sentence_cloze_item.provider_id IS '模型配置ID';
COMMENT ON COLUMN public.sentence_cloze_item.provider_label IS '模型配置名称';
COMMENT ON COLUMN public.sentence_cloze_item.model IS '模型名称';
COMMENT ON COLUMN public.sentence_cloze_item.source IS '内容来源';
COMMENT ON COLUMN public.sentence_cloze_item.source_event_ids_json IS '来源Python错题事件ID JSON';
COMMENT ON COLUMN public.sentence_cloze_item.source_answer_detail_ids_json IS '来源答题明细ID JSON';
COMMENT ON COLUMN public.sentence_cloze_item.source_record_ids_json IS '来源答题记录ID JSON';
COMMENT ON COLUMN public.sentence_cloze_item.source_word_ids_json IS '来源单词ID JSON';
COMMENT ON COLUMN public.sentence_cloze_item.generation_key IS '外部生成请求幂等键';
COMMENT ON COLUMN public.sentence_cloze_item.create_time IS '创建时间';
COMMENT ON COLUMN public.sentence_cloze_item.update_time IS '更新时间';

ALTER SEQUENCE public.sentence_cloze_item_id_seq OWNED BY public.sentence_cloze_item.id;

CREATE INDEX IF NOT EXISTS idx_sentence_cloze_item_word ON public.sentence_cloze_item(word);
CREATE INDEX IF NOT EXISTS idx_sentence_cloze_item_best_sentence_id ON public.sentence_cloze_item(best_sentence_id);
CREATE INDEX IF NOT EXISTS idx_sentence_cloze_item_user_id ON public.sentence_cloze_item(user_id);
CREATE INDEX IF NOT EXISTS idx_sentence_cloze_item_user_create_time ON public.sentence_cloze_item(user_id, create_time);
CREATE INDEX IF NOT EXISTS idx_sentence_cloze_item_model ON public.sentence_cloze_item(model);
CREATE INDEX IF NOT EXISTS idx_sentence_cloze_item_create_time ON public.sentence_cloze_item(create_time);
CREATE UNIQUE INDEX IF NOT EXISTS uk_sentence_cloze_item_generation_key
    ON public.sentence_cloze_item(generation_key)
    WHERE generation_key IS NOT NULL;

CREATE OR REPLACE FUNCTION public.update_sentence_cloze_item_time()
RETURNS TRIGGER AS $$
BEGIN
    NEW.update_time = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_sentence_cloze_item_update_time ON public.sentence_cloze_item;
CREATE TRIGGER trg_sentence_cloze_item_update_time
BEFORE UPDATE ON public.sentence_cloze_item
FOR EACH ROW EXECUTE FUNCTION public.update_sentence_cloze_item_time();
