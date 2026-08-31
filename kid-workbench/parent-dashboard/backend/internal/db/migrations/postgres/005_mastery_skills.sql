CREATE TABLE mastery_skills (
  child_id      BIGINT NOT NULL REFERENCES children(id),
  kp_id         BIGINT NOT NULL REFERENCES knowledge_points(id),
  skill_code    VARCHAR(40) NOT NULL,
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
  PRIMARY KEY (child_id, kp_id, skill_code)
);
CREATE INDEX idx_mastery_skills_child_kp ON mastery_skills(child_id, kp_id);
