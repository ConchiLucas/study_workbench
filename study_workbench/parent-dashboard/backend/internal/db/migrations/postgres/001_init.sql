CREATE TABLE users (
  id            BIGSERIAL PRIMARY KEY,
  phone         VARCHAR(20) NOT NULL UNIQUE,
  nickname      VARCHAR(50) NOT NULL DEFAULT '',
  password_hash VARCHAR(100) NOT NULL DEFAULT '',
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE children (
  id         BIGSERIAL PRIMARY KEY,
  name       VARCHAR(50) NOT NULL,
  grade      VARCHAR(20) NOT NULL DEFAULT '',
  avatar_url VARCHAR(255) NOT NULL DEFAULT '',
  birthday   DATE,
  flowers    INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE parent_child (
  user_id  BIGINT NOT NULL REFERENCES users(id),
  child_id BIGINT NOT NULL REFERENCES children(id),
  relation VARCHAR(20) NOT NULL DEFAULT '',
  role     VARCHAR(20) NOT NULL DEFAULT 'owner',
  PRIMARY KEY (user_id, child_id)
);

CREATE TABLE subjects (
  id       BIGSERIAL PRIMARY KEY,
  code     VARCHAR(30) NOT NULL UNIQUE,
  name     VARCHAR(30) NOT NULL,
  icon     VARCHAR(20) NOT NULL DEFAULT '',
  order_no INT NOT NULL DEFAULT 0
);

CREATE TABLE modules (
  id         BIGSERIAL PRIMARY KEY,
  subject_id BIGINT NOT NULL REFERENCES subjects(id),
  code       VARCHAR(50) NOT NULL,
  name       VARCHAR(50) NOT NULL,
  order_no   INT NOT NULL DEFAULT 0,
  UNIQUE (subject_id, code)
);

CREATE TABLE knowledge_points (
  id         BIGSERIAL PRIMARY KEY,
  module_id  BIGINT NOT NULL REFERENCES modules(id),
  code       VARCHAR(80) NOT NULL,
  title      VARCHAR(80) NOT NULL,
  payload    TEXT NOT NULL DEFAULT '{}',
  difficulty SMALLINT NOT NULL DEFAULT 1,
  order_no   INT NOT NULL DEFAULT 0,
  UNIQUE (module_id, code)
);

CREATE TABLE questions (
  id          BIGSERIAL PRIMARY KEY,
  kp_id       BIGINT NOT NULL REFERENCES knowledge_points(id),
  type        VARCHAR(20) NOT NULL,
  stem        TEXT NOT NULL,
  options     TEXT NOT NULL DEFAULT '[]',
  answer      TEXT NOT NULL DEFAULT '{}',
  media_url   VARCHAR(255) NOT NULL DEFAULT '',
  difficulty  SMALLINT NOT NULL DEFAULT 1
);
CREATE INDEX idx_questions_kp ON questions(kp_id);

CREATE TABLE attempts (
  id          BIGSERIAL PRIMARY KEY,
  child_id    BIGINT NOT NULL REFERENCES children(id),
  kp_id       BIGINT NOT NULL REFERENCES knowledge_points(id),
  question_id BIGINT,
  is_correct  BOOLEAN NOT NULL,
  cost_ms     INT NOT NULL DEFAULT 0,
  source      VARCHAR(20) NOT NULL,
  client_id   VARCHAR(64) NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL
);
CREATE UNIQUE INDEX idx_attempts_idem ON attempts(child_id, client_id);
CREATE INDEX idx_attempts_child_kp_time ON attempts(child_id, kp_id, created_at DESC);
CREATE INDEX idx_attempts_child_time ON attempts(child_id, created_at DESC);

CREATE TABLE mastery_states (
  child_id      BIGINT NOT NULL REFERENCES children(id),
  kp_id         BIGINT NOT NULL REFERENCES knowledge_points(id),
  status        VARCHAR(20) NOT NULL,
  attempts      INT NOT NULL DEFAULT 0,
  correct       INT NOT NULL DEFAULT 0,
  streak        INT NOT NULL DEFAULT 0,
  best_streak   INT NOT NULL DEFAULT 0,
  ease          REAL NOT NULL DEFAULT 2.5,
  interval_days INT NOT NULL DEFAULT 0,
  due_at        TIMESTAMPTZ,
  first_seen_at TIMESTAMPTZ,
  mastered_at   TIMESTAMPTZ,
  updated_at    TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (child_id, kp_id)
);
CREATE INDEX idx_mastery_child_status ON mastery_states(child_id, status);
CREATE INDEX idx_mastery_child_due ON mastery_states(child_id, due_at);

CREATE TABLE study_sessions (
  id           BIGSERIAL PRIMARY KEY,
  child_id     BIGINT NOT NULL,
  subject_id   BIGINT,
  started_at   TIMESTAMPTZ NOT NULL,
  ended_at     TIMESTAMPTZ,
  duration_sec INT NOT NULL DEFAULT 0
);

CREATE TABLE daily_stats (
  child_id       BIGINT NOT NULL,
  stat_date      DATE NOT NULL,
  practice_sec   INT NOT NULL DEFAULT 0,
  attempts       INT NOT NULL DEFAULT 0,
  correct        INT NOT NULL DEFAULT 0,
  newly_mastered INT NOT NULL DEFAULT 0,
  review_done    INT NOT NULL DEFAULT 0,
  checked_in     BOOLEAN NOT NULL DEFAULT false,
  PRIMARY KEY (child_id, stat_date)
);

CREATE TABLE daily_tasks (
  -- 已弃用：日常打卡任务由 study_plans 取代。表保留以追溯 flower_ledger 历史。
  id             BIGSERIAL PRIMARY KEY,
  child_id       BIGINT NOT NULL,
  task_date      DATE NOT NULL,
  title          VARCHAR(80) NOT NULL,
  subject_id     BIGINT,
  target_count   INT NOT NULL DEFAULT 1,
  done_count     INT NOT NULL DEFAULT 0,
  reward_flowers INT NOT NULL DEFAULT 1,
  status         VARCHAR(20) NOT NULL DEFAULT 'pending'
);
CREATE INDEX idx_tasks_child_date ON daily_tasks(child_id, task_date);

CREATE TABLE rewards (
  id       BIGSERIAL PRIMARY KEY,
  child_id BIGINT NOT NULL,
  name     VARCHAR(50) NOT NULL,
  cost     INT NOT NULL,
  stock    INT NOT NULL DEFAULT 1
);

CREATE TABLE flower_ledger (
  id         BIGSERIAL PRIMARY KEY,
  child_id   BIGINT NOT NULL,
  delta      INT NOT NULL,
  reason     VARCHAR(50) NOT NULL,
  ref_type   VARCHAR(20) NOT NULL DEFAULT '',
  ref_id     BIGINT,
  created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_flower_child_time ON flower_ledger(child_id, created_at DESC);
