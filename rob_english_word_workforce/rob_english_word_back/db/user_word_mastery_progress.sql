CREATE TABLE IF NOT EXISTS public.user_word_mastery_progress (
    id bigserial PRIMARY KEY,
    user_id int8 NOT NULL,
    word_id int8 NOT NULL,
    word_content varchar(255) NULL,
    correct_meaning text NULL,
    status varchar(20) NOT NULL DEFAULT 'learning',
    stage int2 NOT NULL DEFAULT 0,
    correct_count int4 NOT NULL DEFAULT 0,
    first_correct_time timestamp NULL,
    day1_correct_time timestamp NULL,
    day7_correct_time timestamp NULL,
    next_review_time timestamp NULL,
    last_correct_time timestamp NULL,
    mastered_time timestamp NULL,
    last_answer_detail_id int8 NULL,
    create_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_user_word_mastery_user_word UNIQUE (user_id, word_id),
    CONSTRAINT ck_user_word_mastery_status CHECK (status IN ('learning', 'mastered'))
);

COMMENT ON TABLE public.user_word_mastery_progress IS '用户单人训练单词掌握进度表';
COMMENT ON COLUMN public.user_word_mastery_progress.status IS 'learning-复习中 mastered-已掌握';
COMMENT ON COLUMN public.user_word_mastery_progress.stage IS '0-未开始 1-首次答对待1天复习 2-1天复习答对待7天复习 3-已掌握';
COMMENT ON COLUMN public.user_word_mastery_progress.next_review_time IS '下一次复习到期时间';

CREATE INDEX IF NOT EXISTS idx_user_word_mastery_user_status ON public.user_word_mastery_progress(user_id, status);
CREATE INDEX IF NOT EXISTS idx_user_word_mastery_user_next_review ON public.user_word_mastery_progress(user_id, next_review_time);
CREATE INDEX IF NOT EXISTS idx_user_word_mastery_word_id ON public.user_word_mastery_progress(word_id);
