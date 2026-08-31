-- 1. 创建自增序列
CREATE SEQUENCE IF NOT EXISTS public.word_library_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- 给序列添加注释
COMMENT ON SEQUENCE public.word_library_id_seq IS '词库表id自增序列';

-- 2. 创建词库表（PostgreSQL 标准字段定义格式）
CREATE TABLE IF NOT EXISTS public.word_library (
    -- 自增主键
    id bigint PRIMARY KEY DEFAULT nextval('public.word_library_id_seq'),
    -- 核心业务字段
    library_name varchar(100) NOT NULL,
    library_meaning varchar(200) NULL,
    status int2 DEFAULT 1 NOT NULL,
    word_count int4 DEFAULT 0 NOT NULL,
    created_by int8 NULL,
    create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 3. 给表添加注释
ALTER TABLE public.word_library ADD COLUMN IF NOT EXISTS library_meaning varchar(200) NULL;

COMMENT ON TABLE public.word_library IS '词库表';

-- 4. 给所有字段添加注释
COMMENT ON COLUMN public.word_library.id IS 'id';
COMMENT ON COLUMN public.word_library.library_name IS '词库名称';
COMMENT ON COLUMN public.word_library.library_meaning IS '词库名称中文说明';
COMMENT ON COLUMN public.word_library.status IS '状态(0:禁用/1:启用)';
COMMENT ON COLUMN public.word_library.word_count IS '单词数量';
COMMENT ON COLUMN public.word_library.created_by IS '创建者ID';
COMMENT ON COLUMN public.word_library.create_time IS '创建时间';
COMMENT ON COLUMN public.word_library.update_time IS '更新时间';

-- 5. 关联序列和主键字段（删除表时自动删除序列）
ALTER SEQUENCE public.word_library_id_seq OWNED BY public.word_library.id;

-- 6. 创建索引
CREATE INDEX IF NOT EXISTS idx_word_library_created_by ON public.word_library(created_by);
CREATE INDEX IF NOT EXISTS idx_word_library_status ON public.word_library(status);

-- 7. 创建更新时间触发器（实现MySQL的ON UPDATE CURRENT_TIMESTAMP效果）
CREATE OR REPLACE FUNCTION public.update_word_library_time()
RETURNS TRIGGER AS $$
BEGIN
    NEW.update_time = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_word_library_update_time
BEFORE UPDATE ON public.word_library
FOR EACH ROW EXECUTE FUNCTION public.update_word_library_time();
