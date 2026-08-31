CREATE TABLE IF NOT EXISTS public.user_word_state (
    id bigserial PRIMARY KEY,
    user_id int8 NOT NULL,
    word_id int8 NOT NULL,
    create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_user_word_state_user_word UNIQUE (user_id, word_id)
);

DROP TRIGGER IF EXISTS trg_user_word_state_update_time ON public.user_word_state;
DROP FUNCTION IF EXISTS public.update_user_word_state_time();
DROP INDEX IF EXISTS public.idx_user_word_state_user_status;

ALTER TABLE public.user_word_state DROP CONSTRAINT IF EXISTS ck_user_word_state_status;
ALTER TABLE public.user_word_state DROP COLUMN IF EXISTS status;
ALTER TABLE public.user_word_state DROP COLUMN IF EXISTS mastered_at;
ALTER TABLE public.user_word_state DROP COLUMN IF EXISTS hidden_at;
ALTER TABLE public.user_word_state DROP COLUMN IF EXISTS update_time;

COMMENT ON TABLE public.user_word_state IS '用户已掌握错词表';
COMMENT ON COLUMN public.user_word_state.user_id IS '用户ID';
COMMENT ON COLUMN public.user_word_state.word_id IS '单词ID';
COMMENT ON COLUMN public.user_word_state.create_time IS '标记已掌握时间';

CREATE INDEX IF NOT EXISTS idx_user_word_state_user_id ON public.user_word_state(user_id);
CREATE INDEX IF NOT EXISTS idx_user_word_state_word_id ON public.user_word_state(word_id);
