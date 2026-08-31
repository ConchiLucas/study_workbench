-- 把孩子端补做窗口（今天/昨天/前天）内的 demo 任务摆好。
-- 幂等：按稳定条件更新，避免 (child_id, plan_date, seq_no) 冲突。

-- 删掉 API 临时生成的今日计划（非 curated demo）
DELETE FROM plan_items
WHERE plan_id IN (
  SELECT id FROM study_plans
  WHERE child_id = 1 AND plan_date = CURRENT_DATE AND id > 10
);
DELETE FROM study_plans
WHERE child_id = 1 AND plan_date = CURRENT_DATE AND id > 10;

-- plan 10 含成语/短句，挪到今天做主任务
UPDATE study_plans
SET plan_date = CURRENT_DATE,
    seq_no = 1,
    status = 'pending',
    done_count = 0,
    correct_count = 0,
    stars = 0,
    duration_sec = 0,
    completed_at = NULL
WHERE child_id = 1 AND id = 10;

UPDATE plan_items
SET status = 'pending', tries = 0, cost_ms = 0, picks = ''
WHERE plan_id = 10;

-- 昨天：doing 且做了 6 题的那份（plan 7）
UPDATE study_plans
SET plan_date = CURRENT_DATE - 1,
    seq_no = 2
WHERE child_id = 1 AND status = 'doing' AND done_count = 6
  AND plan_date < CURRENT_DATE - 2
  AND NOT EXISTS (
    SELECT 1 FROM study_plans x
    WHERE x.child_id = 1 AND x.plan_date = CURRENT_DATE - 1 AND x.seq_no = 2 AND x.id <> study_plans.id
  );

-- 前天：doing 且做了 3 题的那份（plan 8）
UPDATE study_plans
SET plan_date = CURRENT_DATE - 2,
    seq_no = 2
WHERE child_id = 1 AND status = 'doing' AND done_count = 3
  AND plan_date < CURRENT_DATE - 2
  AND NOT EXISTS (
    SELECT 1 FROM study_plans x
    WHERE x.child_id = 1 AND x.plan_date = CURRENT_DATE - 2 AND x.seq_no = 2 AND x.id <> study_plans.id
  );

-- 再塞一份前天 pending 补做（plan 9）
UPDATE study_plans
SET plan_date = CURRENT_DATE - 2,
    seq_no = 3
WHERE child_id = 1 AND id = 9 AND status = 'pending'
  AND plan_date < CURRENT_DATE - 2
  AND NOT EXISTS (
    SELECT 1 FROM study_plans x
    WHERE x.child_id = 1 AND x.plan_date = CURRENT_DATE - 2 AND x.seq_no = 3 AND x.id <> 9
  );

SELECT id, plan_date, seq_no, status, done_count, target_count
FROM study_plans
WHERE child_id = 1 AND status IN ('pending', 'doing')
  AND plan_date >= CURRENT_DATE - 2
ORDER BY plan_date DESC, seq_no;
