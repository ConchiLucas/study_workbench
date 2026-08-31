ALTER TABLE subjects ADD COLUMN quiz_enabled INTEGER NOT NULL DEFAULT 0;

UPDATE subjects SET quiz_enabled = 1
WHERE code IN ('math', 'pinyin', 'literacy', 'english');

ALTER TABLE questions ADD COLUMN code   TEXT NOT NULL DEFAULT '';
ALTER TABLE questions ADD COLUMN visual TEXT NOT NULL DEFAULT '{}';
ALTER TABLE questions ADD COLUMN speech TEXT NOT NULL DEFAULT '{}';

CREATE UNIQUE INDEX uq_questions_kp_code ON questions(kp_id, code);
