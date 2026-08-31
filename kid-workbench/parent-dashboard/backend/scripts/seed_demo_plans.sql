-- 补 8 份历史任务 demo（幂等：按日期跳过已存在的）
DO $$
DECLARE
  rec RECORD;
  new_id BIGINT;
  src_plan_id BIGINT := 1;
  specs RECORD;
BEGIN
  FOR specs IN
    SELECT * FROM (VALUES
      ('2026-08-21'::date, 'done'::varchar,   10, 10,  9, 3, 420),
      ('2026-08-20'::date, 'done'::varchar,   10, 10,  8, 2, 380),
      ('2026-08-19'::date, 'done'::varchar,   10, 10,  7, 2, 510),
      ('2026-08-18'::date, 'done'::varchar,   10, 10,  5, 1, 600),
      ('2026-08-17'::date, 'doing'::varchar,  10,  6,  5, 0, 240),
      ('2026-08-16'::date, 'doing'::varchar,  10,  3,  2, 0, 120),
      ('2026-08-15'::date, 'pending'::varchar,10,  0,  0, 0,   0),
      ('2026-08-14'::date, 'pending'::varchar,10,  0,  0, 0,   0)
    ) AS t(plan_date, status, target_count, done_count, correct_count, stars, duration_sec)
  LOOP
    IF EXISTS (
      SELECT 1 FROM study_plans
      WHERE child_id = 1 AND plan_date = specs.plan_date AND seq_no = 1
    ) THEN
      CONTINUE;
    END IF;

    INSERT INTO study_plans (
      child_id, plan_date, seq_no, status, target_count, done_count,
      correct_count, stars, duration_sec, created_at, completed_at
    ) VALUES (
      1, specs.plan_date, 1, specs.status, specs.target_count, specs.done_count,
      specs.correct_count, specs.stars, specs.duration_sec,
      specs.plan_date::timestamptz + time '18:00',
      CASE WHEN specs.status = 'done' THEN specs.plan_date::timestamptz + time '18:10' ELSE NULL END
    ) RETURNING id INTO new_id;

    INSERT INTO plan_items (plan_id, seq, kp_id, question_id, bucket, status, tries, cost_ms, picks)
    SELECT
      new_id,
      pi.seq,
      pi.kp_id,
      pi.question_id,
      pi.bucket,
      CASE
        WHEN specs.status = 'pending' THEN 'pending'
        WHEN specs.status = 'doing' AND pi.seq < specs.done_count THEN 'correct'
        WHEN specs.status = 'doing' AND pi.seq = specs.done_count THEN 'pending'
        WHEN specs.status = 'done' AND pi.seq <= specs.correct_count THEN 'correct'
        WHEN specs.status = 'done' AND pi.seq <= specs.done_count THEN 'wrong'
        ELSE 'pending'
      END,
      CASE
        WHEN specs.status = 'pending' THEN 0
        WHEN specs.status = 'doing' AND pi.seq > specs.done_count THEN 0
        ELSE GREATEST(pi.tries, 1)
      END,
      CASE WHEN specs.status = 'pending' THEN 0 ELSE pi.cost_ms END,
      CASE WHEN specs.status IN ('done', 'doing') AND pi.seq < specs.done_count THEN pi.picks ELSE '' END
    FROM plan_items pi
    WHERE pi.plan_id = src_plan_id;
  END LOOP;
END $$;

SELECT id, plan_date, seq_no, status, done_count, correct_count, stars
FROM study_plans WHERE child_id = 1
ORDER BY plan_date DESC, seq_no;
