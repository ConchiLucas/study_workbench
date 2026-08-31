CREATE TABLE users (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  phone         TEXT NOT NULL UNIQUE,
  nickname      TEXT NOT NULL DEFAULT '',
  password_hash TEXT NOT NULL DEFAULT '',
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE children (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  name       TEXT NOT NULL,
  grade      TEXT NOT NULL DEFAULT '',
  avatar_url TEXT NOT NULL DEFAULT '',
  birthday   DATE,
  flowers    INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE parent_child (
  user_id  INTEGER NOT NULL REFERENCES users(id),
  child_id INTEGER NOT NULL REFERENCES children(id),
  relation TEXT NOT NULL DEFAULT '',
  role     TEXT NOT NULL DEFAULT 'owner',
  PRIMARY KEY (user_id, child_id)
);

CREATE TABLE subjects (
  id       INTEGER PRIMARY KEY AUTOINCREMENT,
  code     TEXT NOT NULL UNIQUE,
  name     TEXT NOT NULL,
  icon     TEXT NOT NULL DEFAULT '',
  order_no INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE modules (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  subject_id INTEGER NOT NULL REFERENCES subjects(id),
  code       TEXT NOT NULL,
  name       TEXT NOT NULL,
  order_no   INTEGER NOT NULL DEFAULT 0,
  UNIQUE (subject_id, code)
);

CREATE TABLE knowledge_points (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  module_id  INTEGER NOT NULL REFERENCES modules(id),
  code       TEXT NOT NULL,
  title      TEXT NOT NULL,
  payload    TEXT NOT NULL DEFAULT '{}',
  difficulty INTEGER NOT NULL DEFAULT 1,
  order_no   INTEGER NOT NULL DEFAULT 0,
  UNIQUE (module_id, code)
);

CREATE TABLE questions (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  kp_id       INTEGER NOT NULL REFERENCES knowledge_points(id),
  type        TEXT NOT NULL,
  stem        TEXT NOT NULL,
  options     TEXT NOT NULL DEFAULT '[]',
  answer      TEXT NOT NULL DEFAULT '{}',
  media_url   TEXT NOT NULL DEFAULT '',
  difficulty  INTEGER NOT NULL DEFAULT 1
);
CREATE INDEX idx_questions_kp ON questions(kp_id);

CREATE TABLE attempts (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  child_id    INTEGER NOT NULL REFERENCES children(id),
  kp_id       INTEGER NOT NULL REFERENCES knowledge_points(id),
  question_id INTEGER,
  is_correct  INTEGER NOT NULL,
  cost_ms     INTEGER NOT NULL DEFAULT 0,
  source      TEXT NOT NULL,
  client_id   TEXT NOT NULL,
  created_at  DATETIME NOT NULL
);
CREATE UNIQUE INDEX idx_attempts_idem ON attempts(child_id, client_id);
CREATE INDEX idx_attempts_child_kp_time ON attempts(child_id, kp_id, created_at DESC);
CREATE INDEX idx_attempts_child_time ON attempts(child_id, created_at DESC);

CREATE TABLE mastery_states (
  child_id      INTEGER NOT NULL REFERENCES children(id),
  kp_id         INTEGER NOT NULL REFERENCES knowledge_points(id),
  status        TEXT NOT NULL,
  attempts      INTEGER NOT NULL DEFAULT 0,
  correct       INTEGER NOT NULL DEFAULT 0,
  streak        INTEGER NOT NULL DEFAULT 0,
  best_streak   INTEGER NOT NULL DEFAULT 0,
  ease          REAL NOT NULL DEFAULT 2.5,
  interval_days INTEGER NOT NULL DEFAULT 0,
  due_at        DATETIME,
  first_seen_at DATETIME,
  mastered_at   DATETIME,
  updated_at    DATETIME NOT NULL,
  PRIMARY KEY (child_id, kp_id)
);
CREATE INDEX idx_mastery_child_status ON mastery_states(child_id, status);
CREATE INDEX idx_mastery_child_due ON mastery_states(child_id, due_at);

CREATE TABLE study_sessions (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  child_id     INTEGER NOT NULL,
  subject_id   INTEGER,
  started_at   DATETIME NOT NULL,
  ended_at     DATETIME,
  duration_sec INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE daily_stats (
  child_id       INTEGER NOT NULL,
  stat_date      DATE NOT NULL,
  practice_sec   INTEGER NOT NULL DEFAULT 0,
  attempts       INTEGER NOT NULL DEFAULT 0,
  correct        INTEGER NOT NULL DEFAULT 0,
  newly_mastered INTEGER NOT NULL DEFAULT 0,
  review_done    INTEGER NOT NULL DEFAULT 0,
  checked_in     INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (child_id, stat_date)
);

CREATE TABLE daily_tasks (
  -- 已弃用：日常打卡任务由 study_plans 取代。表保留以追溯 flower_ledger 历史。
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  child_id       INTEGER NOT NULL,
  task_date      DATE NOT NULL,
  title          TEXT NOT NULL,
  subject_id     INTEGER,
  target_count   INTEGER NOT NULL DEFAULT 1,
  done_count     INTEGER NOT NULL DEFAULT 0,
  reward_flowers INTEGER NOT NULL DEFAULT 1,
  status         TEXT NOT NULL DEFAULT 'pending'
);
CREATE INDEX idx_tasks_child_date ON daily_tasks(child_id, task_date);

CREATE TABLE rewards (
  id       INTEGER PRIMARY KEY AUTOINCREMENT,
  child_id INTEGER NOT NULL,
  name     TEXT NOT NULL,
  cost     INTEGER NOT NULL,
  stock    INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE flower_ledger (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  child_id   INTEGER NOT NULL,
  delta      INTEGER NOT NULL,
  reason     TEXT NOT NULL,
  ref_type   TEXT NOT NULL DEFAULT '',
  ref_id     INTEGER,
  created_at DATETIME NOT NULL
);
CREATE INDEX idx_flower_child_time ON flower_ledger(child_id, created_at DESC);
