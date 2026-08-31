-- 删掉今天的计划，便于重新生成（会带上新学科的掌握度选题）。
DELETE FROM plan_items
WHERE plan_id IN (
  SELECT id FROM study_plans
  WHERE child_id = 1 AND plan_date = CURRENT_DATE
);
DELETE FROM study_plans
WHERE child_id = 1 AND plan_date = CURRENT_DATE;
