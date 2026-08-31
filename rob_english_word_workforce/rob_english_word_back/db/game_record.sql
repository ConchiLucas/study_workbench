-- 1. 创建自增序列
CREATE SEQUENCE IF NOT EXISTS public.game_record_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- 给序列添加注释
COMMENT ON SEQUENCE public.game_record_id_seq IS '比赛记录表id自增序列';

-- 2. 创建比赛记录表（PostgreSQL 标准字段定义格式）
CREATE TABLE IF NOT EXISTS public.game_record (
    -- 自增主键
    id bigint PRIMARY KEY DEFAULT nextval('public.game_record_id_seq'),
    -- 核心业务字段
    room_id int8 NOT NULL,
    mode varchar(32) NOT NULL DEFAULT 'match',
    player1_id int8 NOT NULL,
    player1_name varchar(50) NULL,
    player1_score int4 DEFAULT 0 NULL,
    player1_correct_count int4 DEFAULT 0 NULL,
    player1_total_count int4 DEFAULT 0 NULL,
    player2_id int8 NOT NULL,
    player2_name varchar(50) NULL,
    player2_score int4 DEFAULT 0 NULL,
    player2_correct_count int4 DEFAULT 0 NULL,
    player2_total_count int4 DEFAULT 0 NULL,
    winner_id int8 NULL,
    is_draw int2 DEFAULT 0 NULL,
    start_time timestamp NULL,
    end_time timestamp NULL,
    duration_seconds int4 NULL,
    match_difficulty_group varchar(64) NULL,
    match_difficulty_level varchar(64) NULL,
    match_difficulty_label varchar(128) NULL,
    training_exp_change int4 DEFAULT 0 NULL,
    training_rank_after int4 NULL,
    training_difficulty_group varchar(64) NULL,
    training_difficulty_level varchar(64) NULL,
    robot_tier varchar(32) NULL,
    robot_aptitude int4 NULL,
    robot_growth numeric(5, 2) NULL,
    robot_profile_json text NULL,
    create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 3. 兼容已存在的旧表
ALTER TABLE public.game_record ADD COLUMN IF NOT EXISTS mode varchar(32) NOT NULL DEFAULT 'match';
ALTER TABLE public.game_record ADD COLUMN IF NOT EXISTS match_difficulty_group varchar(64) NULL;
ALTER TABLE public.game_record ADD COLUMN IF NOT EXISTS match_difficulty_level varchar(64) NULL;
ALTER TABLE public.game_record ADD COLUMN IF NOT EXISTS match_difficulty_label varchar(128) NULL;
ALTER TABLE public.game_record ADD COLUMN IF NOT EXISTS training_exp_change int4 DEFAULT 0 NULL;
ALTER TABLE public.game_record ADD COLUMN IF NOT EXISTS training_rank_after int4 NULL;
ALTER TABLE public.game_record ADD COLUMN IF NOT EXISTS training_difficulty_group varchar(64) NULL;
ALTER TABLE public.game_record ADD COLUMN IF NOT EXISTS training_difficulty_level varchar(64) NULL;
ALTER TABLE public.game_record ADD COLUMN IF NOT EXISTS robot_tier varchar(32) NULL;
ALTER TABLE public.game_record ADD COLUMN IF NOT EXISTS robot_aptitude int4 NULL;
ALTER TABLE public.game_record ADD COLUMN IF NOT EXISTS robot_growth numeric(5, 2) NULL;
ALTER TABLE public.game_record ADD COLUMN IF NOT EXISTS robot_profile_json text NULL;

-- 4. 给表添加注释
COMMENT ON TABLE public.game_record IS '比赛概览记录表';

-- 5. 给所有字段添加注释
COMMENT ON COLUMN public.game_record.id IS '记录ID';
COMMENT ON COLUMN public.game_record.room_id IS '房间ID';
COMMENT ON COLUMN public.game_record.mode IS '记录模式 match-正式匹配 solo_training-单人训练';
COMMENT ON COLUMN public.game_record.player1_id IS '玩家1ID';
COMMENT ON COLUMN public.game_record.player1_name IS '玩家1名称';
COMMENT ON COLUMN public.game_record.player1_score IS '玩家1总得分';
COMMENT ON COLUMN public.game_record.player1_correct_count IS '玩家1答对题数';
COMMENT ON COLUMN public.game_record.player1_total_count IS '玩家1总答题数';
COMMENT ON COLUMN public.game_record.player2_id IS '玩家2ID';
COMMENT ON COLUMN public.game_record.player2_name IS '玩家2名称';
COMMENT ON COLUMN public.game_record.player2_score IS '玩家2总得分';
COMMENT ON COLUMN public.game_record.player2_correct_count IS '玩家2答对题数';
COMMENT ON COLUMN public.game_record.player2_total_count IS '玩家2总答题数';
COMMENT ON COLUMN public.game_record.winner_id IS '获胜者ID，平局为NULL';
COMMENT ON COLUMN public.game_record.is_draw IS '是否平局 0-否 1-是';
COMMENT ON COLUMN public.game_record.start_time IS '比赛开始时间';
COMMENT ON COLUMN public.game_record.end_time IS '比赛结束时间';
COMMENT ON COLUMN public.game_record.duration_seconds IS '比赛持续时间(秒)';
COMMENT ON COLUMN public.game_record.match_difficulty_group IS '正式匹配选择难度父级';
COMMENT ON COLUMN public.game_record.match_difficulty_level IS '正式匹配选择难度';
COMMENT ON COLUMN public.game_record.match_difficulty_label IS '正式匹配难度展示名称';
COMMENT ON COLUMN public.game_record.training_exp_change IS '单人训练经验变化';
COMMENT ON COLUMN public.game_record.training_rank_after IS '单人训练结束后的训练等级';
COMMENT ON COLUMN public.game_record.training_difficulty_group IS '单人训练选择难度父级';
COMMENT ON COLUMN public.game_record.training_difficulty_level IS '单人训练选择难度';
COMMENT ON COLUMN public.game_record.robot_tier IS '训练机器人档位';
COMMENT ON COLUMN public.game_record.robot_aptitude IS '训练机器人资质';
COMMENT ON COLUMN public.game_record.robot_growth IS '训练机器人成长';
COMMENT ON COLUMN public.game_record.robot_profile_json IS '训练机器人完整面板快照';
COMMENT ON COLUMN public.game_record.create_time IS '创建时间';
COMMENT ON COLUMN public.game_record.update_time IS '更新时间';

-- 6. 关联序列和主键字段（删除表时自动删除序列）
ALTER SEQUENCE public.game_record_id_seq OWNED BY public.game_record.id;

-- 7. 创建索引
CREATE INDEX IF NOT EXISTS idx_game_record_room_id ON public.game_record(room_id);
CREATE INDEX IF NOT EXISTS idx_game_record_player1_id ON public.game_record(player1_id);
CREATE INDEX IF NOT EXISTS idx_game_record_player2_id ON public.game_record(player2_id);
CREATE INDEX IF NOT EXISTS idx_game_record_mode ON public.game_record(mode);
CREATE INDEX IF NOT EXISTS idx_game_record_create_time ON public.game_record(create_time);

-- 8. 创建更新时间触发器（实现MySQL的ON UPDATE CURRENT_TIMESTAMP效果）
CREATE OR REPLACE FUNCTION public.update_game_record_time()
RETURNS TRIGGER AS $$
BEGIN
    NEW.update_time = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_game_record_update_time
BEFORE UPDATE ON public.game_record
FOR EACH ROW EXECUTE FUNCTION public.update_game_record_time();
