-- 1. 创建自增序列
CREATE SEQUENCE IF NOT EXISTS public.room_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- 给序列添加注释
COMMENT ON SEQUENCE public.room_id_seq IS '房间表id自增序列';

-- 2. 创建房间表（PostgreSQL 标准字段定义格式）
CREATE TABLE IF NOT EXISTS public.room (
    -- 自增主键
    id bigint PRIMARY KEY DEFAULT nextval('public.room_id_seq'),
    -- 核心业务字段
    room_code varchar(8) NOT NULL,
    status int2 DEFAULT 0 NOT NULL,
    player1_id int8 NOT NULL,
    player2_id int8 NOT NULL,
    winner_id int8 NULL,
    create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at timestamp NULL
);

-- 3. 给表添加注释
COMMENT ON TABLE public.room IS '房间表';

-- 4. 给所有字段添加注释
COMMENT ON COLUMN public.room.id IS 'id';
COMMENT ON COLUMN public.room.room_code IS '房间邀请码';
COMMENT ON COLUMN public.room.status IS '状态(0:等待/1:游戏中/2:已结束)';
COMMENT ON COLUMN public.room.player1_id IS '玩家1ID';
COMMENT ON COLUMN public.room.player2_id IS '玩家2ID';
COMMENT ON COLUMN public.room.winner_id IS '获胜者ID';
COMMENT ON COLUMN public.room.create_time IS '创建时间';
COMMENT ON COLUMN public.room.update_time IS '更新时间';
COMMENT ON COLUMN public.room.ended_at IS '结束时间';

-- 5. 关联序列和主键字段（删除表时自动删除序列）
ALTER SEQUENCE public.room_id_seq OWNED BY public.room.id;

-- 6. 创建索引
CREATE UNIQUE INDEX IF NOT EXISTS idx_room_code ON public.room(room_code);
CREATE INDEX IF NOT EXISTS idx_room_player1 ON public.room(player1_id);
CREATE INDEX IF NOT EXISTS idx_room_player2 ON public.room(player2_id);
CREATE INDEX IF NOT EXISTS idx_room_status ON public.room(status);

-- 7. 创建更新时间触发器（实现MySQL的ON UPDATE CURRENT_TIMESTAMP效果）
CREATE OR REPLACE FUNCTION public.update_room_time()
RETURNS TRIGGER AS $$
BEGIN
    NEW.update_time = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_room_update_time
BEFORE UPDATE ON public.room
FOR EACH ROW EXECUTE FUNCTION public.update_room_time();
