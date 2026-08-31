-- 孩子作答时选过的选项下标，按顺序逗号分隔（如 "3,1"）。
-- 家长端复盘要看"选了什么"，而不仅仅是对错。
ALTER TABLE plan_items ADD COLUMN picks VARCHAR(24) NOT NULL DEFAULT '';
