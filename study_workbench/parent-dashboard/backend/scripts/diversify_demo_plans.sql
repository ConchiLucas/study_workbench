-- 让 10 份任务的题目和学科组合各不相同（可重复执行）
BEGIN;

DELETE FROM plan_items WHERE plan_id IN (
  SELECT id FROM study_plans WHERE child_id = 1
);

CREATE TEMP TABLE plan_recipe (
  plan_id INT,
  seq INT,
  subject_code TEXT,
  bucket TEXT
) ON COMMIT DROP;

INSERT INTO plan_recipe (plan_id, seq, subject_code, bucket) VALUES
(1,1,'math','review'),(1,2,'math','review'),(1,3,'math','shaky'),(1,4,'math','learning'),
(1,5,'chengyu','new'),(1,6,'chengyu','learning'),(1,7,'phrase','new'),
(1,8,'pinyin','review'),(1,9,'pinyin','shaky'),(1,10,'pinyin','new'),
(2,1,'english','review'),(2,2,'english','review'),(2,3,'english','shaky'),(2,4,'english','learning'),
(2,5,'chengyu','new'),(2,6,'phrase','new'),(2,7,'poem','review'),
(2,8,'science','shaky'),(2,9,'science','learning'),(2,10,'science','new'),
(3,1,'logic','review'),(3,2,'logic','review'),(3,3,'logic','shaky'),(3,4,'logic','learning'),
(3,5,'math','new'),(3,6,'math','review'),(3,7,'math','shaky'),
(3,8,'literacy','review'),(3,9,'literacy','learning'),(3,10,'literacy','new'),
(4,1,'pinyin','review'),(4,2,'pinyin','shaky'),(4,3,'pinyin','learning'),(4,4,'pinyin','new'),(4,5,'pinyin','review'),
(4,6,'english','shaky'),(4,7,'english','learning'),(4,8,'english','new'),(4,9,'english','review'),(4,10,'english','shaky'),
(5,1,'poem','review'),(5,2,'poem','review'),(5,3,'poem','shaky'),(5,4,'poem','new'),
(5,5,'science','learning'),(5,6,'science','new'),(5,7,'science','review'),
(5,8,'chengyu','shaky'),(5,9,'phrase','learning'),(5,10,'logic','new'),
(6,1,'math','review'),(6,2,'math','shaky'),(6,3,'math','learning'),
(6,4,'english','new'),(6,5,'english','review'),(6,6,'english','shaky'),
(6,7,'literacy','review'),(6,8,'literacy','new'),
(6,9,'pinyin','learning'),(6,10,'pinyin','shaky'),
(7,1,'science','review'),(7,2,'science','shaky'),(7,3,'science','learning'),(7,4,'science','new'),
(7,5,'logic','review'),(7,6,'logic','shaky'),(7,7,'logic','learning'),
(7,8,'math','new'),(7,9,'math','review'),(7,10,'math','shaky'),
(8,1,'literacy','review'),(8,2,'literacy','shaky'),(8,3,'literacy','learning'),(8,4,'literacy','new'),
(8,5,'poem','review'),(8,6,'poem','shaky'),(8,7,'poem','new'),
(8,8,'pinyin','review'),(8,9,'pinyin','learning'),(8,10,'pinyin','shaky'),
(9,1,'english','review'),(9,2,'english','shaky'),(9,3,'english','learning'),(9,4,'english','new'),(9,5,'english','review'),
(9,6,'logic','shaky'),(9,7,'logic','learning'),(9,8,'logic','new'),(9,9,'logic','review'),(9,10,'logic','shaky'),
(10,1,'math','review'),(10,2,'math','shaky'),(10,3,'math','learning'),(10,4,'math','new'),
(10,5,'poem','review'),(10,6,'poem','shaky'),(10,7,'poem','new'),
(10,8,'chengyu','learning'),(10,9,'phrase','new'),(10,10,'science','review');

CREATE TEMP TABLE question_pool (
  subject_code TEXT,
  question_id BIGINT,
  kp_id BIGINT,
  rn INT
) ON COMMIT DROP;

INSERT INTO question_pool
SELECT s.code, q.id, kp.id,
       ROW_NUMBER() OVER (PARTITION BY s.code ORDER BY q.id)
FROM questions q
JOIN knowledge_points kp ON kp.id = q.kp_id
JOIN modules m ON m.id = kp.module_id
JOIN subjects s ON s.id = m.subject_id
WHERE s.quiz_enabled = TRUE;

-- 同一学科内按 plan_id+seq 错开取题
CREATE TEMP TABLE question_pick AS
SELECT
  pr.plan_id,
  pr.seq,
  pr.bucket,
  qp.kp_id,
  qp.question_id
FROM plan_recipe pr
JOIN LATERAL (
  SELECT kp_id, question_id
  FROM question_pool qp2
  WHERE qp2.subject_code = pr.subject_code
  ORDER BY qp2.rn
  OFFSET ((pr.plan_id * 13 + pr.seq * 5) % 30)
  LIMIT 1
) qp ON TRUE;

INSERT INTO plan_items (plan_id, seq, kp_id, question_id, bucket, status, tries, cost_ms, picks)
SELECT
  qp.plan_id,
  qp.seq,
  qp.kp_id,
  qp.question_id,
  qp.bucket,
  CASE
    WHEN sp.status = 'pending' THEN 'pending'
    WHEN sp.status = 'doing' AND qp.seq < sp.done_count THEN 'correct'
    WHEN sp.status = 'doing' AND qp.seq = sp.done_count THEN 'pending'
    WHEN sp.status = 'done' AND qp.seq <= sp.correct_count THEN 'correct'
    WHEN sp.status = 'done' AND qp.seq <= sp.done_count THEN 'wrong'
    ELSE 'pending'
  END,
  CASE
    WHEN sp.status = 'pending' THEN 0
    WHEN sp.status = 'doing' AND qp.seq > sp.done_count THEN 0
    WHEN sp.status = 'done' AND qp.seq > sp.correct_count AND qp.seq <= sp.done_count THEN 2
    ELSE 1
  END,
  CASE
    WHEN sp.status = 'pending' THEN 0
    WHEN sp.status = 'doing' AND qp.seq > sp.done_count THEN 0
    ELSE 2500 + (qp.plan_id * 137 + qp.seq * 419) % 5000
  END,
  CASE
    WHEN sp.status IN ('done','doing') AND qp.seq < sp.done_count AND qp.seq % 4 = 0 THEN '2'
    ELSE ''
  END
FROM question_pick qp
JOIN study_plans sp ON sp.id = qp.plan_id
WHERE sp.child_id = 1;

COMMIT;

SELECT sp.id, sp.plan_date, sp.seq_no, sp.status,
       string_agg(x.icon || x.name || '×' || x.cnt::text, '  ' ORDER BY x.order_no) AS subjects
FROM study_plans sp
JOIN (
  SELECT pi.plan_id, s.icon, s.name, s.order_no, COUNT(*) AS cnt
  FROM plan_items pi
  JOIN knowledge_points kp ON kp.id = pi.kp_id
  JOIN modules m ON m.id = kp.module_id
  JOIN subjects s ON s.id = m.subject_id
  GROUP BY pi.plan_id, s.icon, s.name, s.order_no
) x ON x.plan_id = sp.id
WHERE sp.child_id = 1
GROUP BY sp.id, sp.plan_date, sp.seq_no, sp.status
ORDER BY sp.plan_date DESC, sp.seq_no;
