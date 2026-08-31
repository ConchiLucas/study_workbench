ALTER TABLE subjects ADD COLUMN quiz_enabled BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE subjects SET quiz_enabled = TRUE
WHERE code IN ('math', 'pinyin', 'literacy', 'english');

ALTER TABLE questions ADD COLUMN code   VARCHAR(40) NOT NULL DEFAULT '';
ALTER TABLE questions ADD COLUMN visual TEXT NOT NULL DEFAULT '{}';
ALTER TABLE questions ADD COLUMN speech TEXT NOT NULL DEFAULT '{}';

CREATE UNIQUE INDEX uq_questions_kp_code ON questions(kp_id, code);
