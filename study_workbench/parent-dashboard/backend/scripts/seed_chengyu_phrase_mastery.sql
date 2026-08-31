-- 为成语、英语短句灌掌握度 demo（幂等 upsert，不清空其他学科数据）。
WITH ranked AS (
  SELECT
    kp.id AS kp_id,
    s.code AS subject_code,
    ROW_NUMBER() OVER (PARTITION BY s.code ORDER BY m.order_no, kp.order_no, kp.id) AS rn
  FROM knowledge_points kp
  JOIN modules m ON m.id = kp.module_id
  JOIN subjects s ON s.id = m.subject_id
  WHERE s.code IN ('chengyu', 'phrase')
),
picked AS (
  SELECT
    kp_id,
    subject_code,
    rn,
    CASE (rn - 1) % 5
      WHEN 0 THEN 'mastered'
      WHEN 1 THEN 'learning'
      WHEN 2 THEN 'shaky'
      WHEN 3 THEN 'review_due'
      ELSE 'not_started'
    END AS status,
    CASE (rn - 1) % 5
      WHEN 0 THEN 8
      WHEN 1 THEN 4
      WHEN 2 THEN 6
      WHEN 3 THEN 5
      ELSE 0
    END AS attempts,
    CASE (rn - 1) % 5
      WHEN 0 THEN 7
      WHEN 1 THEN 2
      WHEN 2 THEN 2
      WHEN 3 THEN 3
      ELSE 0
    END AS correct
  FROM ranked
  WHERE rn <= 12
)
INSERT INTO mastery_states (
  child_id, kp_id, status, attempts, correct, streak, best_streak,
  ease, interval_days, due_at, first_seen_at, mastered_at, updated_at
)
SELECT
  1,
  p.kp_id,
  p.status,
  p.attempts,
  p.correct,
  CASE p.status WHEN 'mastered' THEN 3 WHEN 'learning' THEN 1 ELSE 0 END,
  CASE p.status WHEN 'mastered' THEN 3 ELSE 1 END,
  2.5,
  CASE p.status WHEN 'mastered' THEN 7 WHEN 'review_due' THEN 3 ELSE 0 END,
  CASE
    WHEN p.status = 'review_due' THEN NOW() - INTERVAL '2 days'
    ELSE NULL
  END,
  NOW() - ((p.rn + 3) || ' days')::interval,
  CASE WHEN p.status = 'mastered' THEN NOW() - INTERVAL '10 days' ELSE NULL END,
  NOW()
FROM picked p
WHERE p.status != 'not_started'
ON CONFLICT (child_id, kp_id) DO UPDATE SET
  status = EXCLUDED.status,
  attempts = EXCLUDED.attempts,
  correct = EXCLUDED.correct,
  streak = EXCLUDED.streak,
  best_streak = EXCLUDED.best_streak,
  due_at = EXCLUDED.due_at,
  first_seen_at = EXCLUDED.first_seen_at,
  mastered_at = EXCLUDED.mastered_at,
  updated_at = EXCLUDED.updated_at;

SELECT s.code, s.name, ms.status, COUNT(1) AS cnt
FROM mastery_states ms
JOIN knowledge_points kp ON kp.id = ms.kp_id
JOIN modules m ON m.id = kp.module_id
JOIN subjects s ON s.id = m.subject_id
WHERE ms.child_id = 1 AND s.code IN ('chengyu', 'phrase')
GROUP BY s.code, s.name, ms.status
ORDER BY s.code, ms.status;
