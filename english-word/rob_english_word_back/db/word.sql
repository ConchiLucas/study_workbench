-- 1. 创建自增序列
CREATE SEQUENCE IF NOT EXISTS public.word_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- 给序列添加注释
COMMENT ON SEQUENCE public.word_id_seq IS '单词表id自增序列';

-- 2. 创建单词表（PostgreSQL 标准字段定义格式）
CREATE TABLE IF NOT EXISTS public.word (
    -- 自增主键
    id bigint PRIMARY KEY DEFAULT nextval('public.word_id_seq'),
    -- 核心业务字段
    library_id int8 NOT NULL,
    word varchar(100) NOT NULL,
    meaning varchar(200) NOT NULL,
    pronunciation_us varchar(100) NULL,
    pronunciation_uk varchar(100) NULL,
    frequency int2 DEFAULT 50 NOT NULL,
    difficulty int2 DEFAULT 50 NOT NULL,
    status int2 DEFAULT 1 NOT NULL,
    phrase varchar(500) DEFAULT ''::character varying NULL,
    phrase_translation varchar(500) DEFAULT ''::character varying NULL,
    sentence varchar(800) DEFAULT ''::character varying NULL,
    sentence_translation varchar(800) DEFAULT ''::character varying NULL,
    create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 3. 给表添加注释
COMMENT ON TABLE public.word IS '单词表';

-- 4. 给所有字段添加注释
COMMENT ON COLUMN public.word.id IS 'id';
COMMENT ON COLUMN public.word.library_id IS '词库ID';
COMMENT ON COLUMN public.word.word IS '英文单词';
COMMENT ON COLUMN public.word.meaning IS '中文含义';
COMMENT ON COLUMN public.word.pronunciation_us IS '美式发音音标';
COMMENT ON COLUMN public.word.pronunciation_uk IS '英式发音音标';
COMMENT ON COLUMN public.word.frequency IS '使用频率(1-100)';
COMMENT ON COLUMN public.word.difficulty IS '难度分(1-100)';
COMMENT ON COLUMN public.word.status IS '状态(1:可用, 0:禁用)';
COMMENT ON COLUMN public.word.phrase IS '常用短语';
COMMENT ON COLUMN public.word.phrase_translation IS '短语翻译';
COMMENT ON COLUMN public.word.sentence IS '例句';
COMMENT ON COLUMN public.word.sentence_translation IS '例句翻译';
COMMENT ON COLUMN public.word.create_time IS '创建时间';
COMMENT ON COLUMN public.word.update_time IS '更新时间';

-- 5. 关联序列和主键字段（删除表时自动删除序列）
ALTER SEQUENCE public.word_id_seq OWNED BY public.word.id;

-- 6. 创建索引
CREATE INDEX IF NOT EXISTS idx_word_library_id ON public.word(library_id);
CREATE INDEX IF NOT EXISTS idx_word_frequency ON public.word(frequency);
CREATE INDEX IF NOT EXISTS idx_word_difficulty ON public.word(difficulty);

-- 7. 创建更新时间触发器（实现MySQL的ON UPDATE CURRENT_TIMESTAMP效果）
CREATE OR REPLACE FUNCTION public.update_word_time()
RETURNS TRIGGER AS $$
BEGIN
    NEW.update_time = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_word_update_time
BEFORE UPDATE ON public.word
FOR EACH ROW EXECUTE FUNCTION public.update_word_time();
