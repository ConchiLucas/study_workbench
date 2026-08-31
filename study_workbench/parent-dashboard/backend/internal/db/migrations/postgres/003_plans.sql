CREATE TABLE study_plans (
  id            BIGSERIAL PRIMARY KEY,
  child_id      BIGINT NOT NULL REFERENCES children(id),
  plan_date     DATE NOT NULL,
  seq_no        SMALLINT NOT NULL DEFAULT 1,
  status        VARCHAR(20) NOT NULL DEFAULT 'pending',
  target_count  INT NOT NULL,
  done_count    INT NOT NULL DEFAULT 0,
  correct_count INT NOT NULL DEFAULT 0,
  stars         SMALLINT NOT NULL DEFAULT 0,
  duration_sec  INT NOT NULL DEFAULT 0,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  started_at    TIMESTAMPTZ,
  completed_at  TIMESTAMPTZ,
  UNIQUE (child_id, plan_date, seq_no)
);
CREATE INDEX idx_study_plans_child_date ON study_plans(child_id, plan_date);

CREATE TABLE plan_items (
  id          BIGSERIAL PRIMARY KEY,
  plan_id     BIGINT NOT NULL REFERENCES study_plans(id) ON DELETE CASCADE,
  seq         SMALLINT NOT NULL,
  kp_id       BIGINT NOT NULL REFERENCES knowledge_points(id),
  question_id BIGINT NOT NULL REFERENCES questions(id),
  bucket      VARCHAR(20) NOT NULL,
  status      VARCHAR(20) NOT NULL DEFAULT 'pending',
  tries       SMALLINT NOT NULL DEFAULT 0,
  cost_ms     INT NOT NULL DEFAULT 0,
  answered_at TIMESTAMPTZ,
  UNIQUE (plan_id, seq)
);
CREATE INDEX idx_plan_items_plan ON plan_items(plan_id, seq);
