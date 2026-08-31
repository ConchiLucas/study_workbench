-- 1. 创建自增序列
CREATE SEQUENCE IF NOT EXISTS public.game_answer_detail_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- 给序列添加注释
COMMENT ON SEQUENCE public.game_answer_detail_id_seq IS '比赛答题详情表id自增序列';

-- 2. 创建比赛答题详情表（PostgreSQL 标准字段定义格式）
CREATE TABLE IF NOT EXISTS public.game_answer_detail (
    -- 自增主键
    id bigint PRIMARY KEY DEFAULT nextval('public.game_answer_detail_id_seq'),
    -- 核心业务字段
    record_id int8 NOT NULL,
    user_id int8 NOT NULL,
    user_name varchar(50) NULL,
    round_no int4 NOT NULL,
    word_id int8 NOT NULL,
    word_content varchar(100) NULL,
    word_difficulty int4 NULL,
    option_1 varchar(255) NULL,
    option_2 varchar(255) NULL,
    option_3 varchar(255) NULL,
    option_4 varchar(255) NULL,
    correct_answer_index int4 NOT NULL,
    selected_answer_index int4 NULL,
    is_correct int2 DEFAULT 0 NULL,
    score int4 DEFAULT 0 NULL,
    answer_time_ms int4 NULL,
    create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 3. 给表添加注释
COMMENT ON TABLE public.game_answer_detail IS '比赛答题详情表';

-- 4. 给所有字段添加注释
COMMENT ON COLUMN public.game_answer_detail.id IS '详情ID';
COMMENT ON COLUMN public.game_answer_detail.record_id IS '关联的game_record.id';
COMMENT ON COLUMN public.game_answer_detail.user_id IS '答题用户ID';
COMMENT ON COLUMN public.game_answer_detail.user_name IS '答题用户名称';
COMMENT ON COLUMN public.game_answer_detail.round_no IS '轮次序号(1-8)';
COMMENT ON COLUMN public.game_answer_detail.word_id IS '单词ID';
COMMENT ON COLUMN public.game_answer_detail.word_content IS '单词内容';
COMMENT ON COLUMN public.game_answer_detail.word_difficulty IS '单词难度分';
COMMENT ON COLUMN public.game_answer_detail.option_1 IS '选项1内容';
COMMENT ON COLUMN public.game_answer_detail.option_2 IS '选项2内容';
COMMENT ON COLUMN public.game_answer_detail.option_3 IS '选项3内容';
COMMENT ON COLUMN public.game_answer_detail.option_4 IS '选项4内容';
COMMENT ON COLUMN public.game_answer_detail.correct_answer_index IS '正确答案序号(1-4)';
COMMENT ON COLUMN public.game_answer_detail.selected_answer_index IS '玩家选择的答案序号(1-4，未答为null)';
COMMENT ON COLUMN public.game_answer_detail.is_correct IS '是否答对 0-错 1-对';
COMMENT ON COLUMN public.game_answer_detail.score IS '本题记分';
COMMENT ON COLUMN public.game_answer_detail.answer_time_ms IS '答题用时(毫秒)';
COMMENT ON COLUMN public.game_answer_detail.create_time IS '创建时间';

-- 5. 关联序列和主键字段（删除表时自动删除序列）
ALTER SEQUENCE public.game_answer_detail_id_seq OWNED BY public.game_answer_detail.id;

-- 6. 创建索引
CREATE INDEX IF NOT EXISTS idx_game_answer_detail_record_id ON public.game_answer_detail(record_id);
CREATE INDEX IF NOT EXISTS idx_game_answer_detail_user_id ON public.game_answer_detail(user_id);
CREATE INDEX IF NOT EXISTS idx_game_answer_detail_record_round ON public.game_answer_detail(record_id, round_no);
CREATE INDEX IF NOT EXISTS idx_game_answer_detail_record_user ON public.game_answer_detail(record_id, user_id);
