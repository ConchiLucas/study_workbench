-- 1. 创建自增序列
CREATE SEQUENCE IF NOT EXISTS public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- 给序列添加注释
COMMENT ON SEQUENCE public.users_id_seq IS '用户表id自增序列';

-- 2. 创建用户表（PostgreSQL 标准字段定义格式）
CREATE TABLE IF NOT EXISTS public.users (
    -- 自增主键
    id bigint PRIMARY KEY DEFAULT nextval('public.users_id_seq'),
    -- 核心业务字段
    username varchar(50) NOT NULL,
    "password" varchar(128) NOT NULL,
    nickname varchar(50) NULL,
    avatar varchar(255) NULL,
    "rank" int4 DEFAULT 0 NULL,
    "exp" int4 DEFAULT 0 NULL,
    total_wins int4 DEFAULT 0 NULL,
    total_games int4 DEFAULT 0 NULL,
    current_win_streak int4 DEFAULT 0 NULL,
    training_rank int4 DEFAULT 1 NULL,
    training_exp int4 DEFAULT 0 NULL,
    training_total_wins int4 DEFAULT 0 NULL,
    training_total_games int4 DEFAULT 0 NULL,
    solo_difficulty_group varchar(64) NOT NULL DEFAULT 'junior',
    solo_difficulty_level varchar(64) NOT NULL DEFAULT 'junior',
    create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE public.users
    ADD COLUMN IF NOT EXISTS current_win_streak int4 DEFAULT 0 NULL;
ALTER TABLE public.users
    ADD COLUMN IF NOT EXISTS training_rank int4 DEFAULT 1 NULL;
ALTER TABLE public.users
    ADD COLUMN IF NOT EXISTS training_exp int4 DEFAULT 0 NULL;
ALTER TABLE public.users
    ADD COLUMN IF NOT EXISTS training_total_wins int4 DEFAULT 0 NULL;
ALTER TABLE public.users
    ADD COLUMN IF NOT EXISTS training_total_games int4 DEFAULT 0 NULL;
ALTER TABLE public.users
    ADD COLUMN IF NOT EXISTS solo_difficulty_group varchar(64) NOT NULL DEFAULT 'junior';
ALTER TABLE public.users
    ADD COLUMN IF NOT EXISTS solo_difficulty_level varchar(64) NOT NULL DEFAULT 'junior';
UPDATE public.users
SET solo_difficulty_group = 'junior', solo_difficulty_level = 'junior'
WHERE solo_difficulty_group IS NULL OR solo_difficulty_group = ''
   OR solo_difficulty_level IS NULL OR solo_difficulty_level = '';

-- 3. 给表添加注释
COMMENT ON TABLE public.users IS '用户信息表';

-- 4. 给所有字段添加注释
COMMENT ON COLUMN public.users.id IS '用户ID，自增主键';
COMMENT ON COLUMN public.users.username IS '用户名';
COMMENT ON COLUMN public.users."password" IS '密码';
COMMENT ON COLUMN public.users.nickname IS '昵称';
COMMENT ON COLUMN public.users.avatar IS '头像地址';
COMMENT ON COLUMN public.users."rank" IS '排名';
COMMENT ON COLUMN public.users."exp" IS '经验值';
COMMENT ON COLUMN public.users.total_wins IS '总胜利次数';
COMMENT ON COLUMN public.users.total_games IS '总游戏次数';
COMMENT ON COLUMN public.users.current_win_streak IS '当前有效连胜数';
COMMENT ON COLUMN public.users.training_rank IS '单人训练等级';
COMMENT ON COLUMN public.users.training_exp IS '单人训练经验值';
COMMENT ON COLUMN public.users.training_total_wins IS '单人训练胜利次数';
COMMENT ON COLUMN public.users.training_total_games IS '单人训练总场次';
COMMENT ON COLUMN public.users.solo_difficulty_group IS '挖空练习单独训练难度分组';
COMMENT ON COLUMN public.users.solo_difficulty_level IS '挖空练习单独训练难度';
COMMENT ON COLUMN public.users.create_time IS '创建时间';
COMMENT ON COLUMN public.users.update_time IS '更新时间';

-- 5. 关联序列和主键字段（删除表时自动删除序列）
ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;

-- 6. 创建索引（用户名唯一索引，对应MySQL的UNIQUE KEY）
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON public.users(username);

-- 7. 创建更新时间触发器（实现MySQL的ON UPDATE CURRENT_TIMESTAMP效果）
CREATE OR REPLACE FUNCTION public.update_users_time()
RETURNS TRIGGER AS $$
BEGIN
    NEW.update_time = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_update_time
BEFORE UPDATE ON public.users
FOR EACH ROW EXECUTE FUNCTION public.update_users_time();
