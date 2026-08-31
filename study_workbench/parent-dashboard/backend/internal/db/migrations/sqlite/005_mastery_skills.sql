CREATE TABLE mastery_skills (
  child_id      INTEGER NOT NULL REFERENCES children(id),
  kp_id         INTEGER NOT NULL REFERENCES knowledge_points(id),
  skill_code    TEXT NOT NULL,
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
  PRIMARY KEY (child_id, kp_id, skill_code)
);
CREATE INDEX idx_mastery_skills_child_kp ON mastery_skills(child_id, kp_id);
