CREATE TABLE study_plans (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  child_id      INTEGER NOT NULL REFERENCES children(id),
  plan_date     DATE NOT NULL,
  seq_no        INTEGER NOT NULL DEFAULT 1,
  status        TEXT NOT NULL DEFAULT 'pending',
  target_count  INTEGER NOT NULL,
  done_count    INTEGER NOT NULL DEFAULT 0,
  correct_count INTEGER NOT NULL DEFAULT 0,
  stars         INTEGER NOT NULL DEFAULT 0,
  duration_sec  INTEGER NOT NULL DEFAULT 0,
  created_at    DATETIME NOT NULL,
  started_at    DATETIME,
  completed_at  DATETIME,
  UNIQUE (child_id, plan_date, seq_no)
);
CREATE INDEX idx_study_plans_child_date ON study_plans(child_id, plan_date);

CREATE TABLE plan_items (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  plan_id     INTEGER NOT NULL REFERENCES study_plans(id) ON DELETE CASCADE,
  seq         INTEGER NOT NULL,
  kp_id       INTEGER NOT NULL REFERENCES knowledge_points(id),
  question_id INTEGER NOT NULL REFERENCES questions(id),
  bucket      TEXT NOT NULL,
  status      TEXT NOT NULL DEFAULT 'pending',
  tries       INTEGER NOT NULL DEFAULT 0,
  cost_ms     INTEGER NOT NULL DEFAULT 0,
  answered_at DATETIME,
  UNIQUE (plan_id, seq)
);
CREATE INDEX idx_plan_items_plan ON plan_items(plan_id, seq);
