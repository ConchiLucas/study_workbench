# 最佳例句挖空字段回填 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 `word_clean_best_sentence` 新增 `cloze_sentence`、`cloze_answer` 并用句中真实词形完成全量回填。

**Architecture:** 在 word-agent 中增加纯函数词形定位器，统一供最佳例句 upsert 和一次性回填脚本调用。回填先计算并校验全量结果，只有全部记录可信时才在单个事务中修改表结构和数据。

**Tech Stack:** Python 3.12、pytest、psycopg 3、PostgreSQL。

---

### Task 1: 词形定位与挖空纯函数

**Files:**
- Create: `word_select_dashboard/word-agent/tests/test_best_sentence_cloze.py`
- Create: `word_select_dashboard/word-agent/src/word_agent/services/best_sentence_cloze.py`

- [x] **Step 1: 写失败测试**

覆盖原词、大小写、第三人称、过去式、复数、进行时、词组锚点、重复出现和候选冲突。

- [x] **Step 2: 运行测试并确认因模块不存在而失败**

Run: `.venv/bin/pytest tests/test_best_sentence_cloze.py -q`
Expected: collection error，找不到 `best_sentence_cloze`。

- [x] **Step 3: 实现最小定位器**

实现 `build_best_sentence_cloze(word, sentence)`，返回 `cloze_sentence`、句中真实 `answer`、匹配类型和挖空次数；无法唯一定位返回 `None`。

- [x] **Step 4: 运行测试并确认通过**

Run: `.venv/bin/pytest tests/test_best_sentence_cloze.py -q`
Expected: all tests pass。

### Task 2: 表结构和最佳例句写入链路

**Files:**
- Modify: `rob_english_word_back/db/word_clean_best_sentence.sql`
- Modify: `word_select_dashboard/word-agent/src/word_agent/services/word_clean_sentence_score.py`

- [x] **Step 1: 在建表 SQL 增加两个非空文本字段及注释**

字段为 `cloze_sentence text NOT NULL DEFAULT ''` 和 `cloze_answer text NOT NULL DEFAULT ''`。

- [x] **Step 2: 在运行时初始化中幂等增加字段**

对已有表执行 `ADD COLUMN IF NOT EXISTS`，新建表定义同步包含字段。

- [x] **Step 3: 最佳例句 upsert 同步计算并写入挖空结果**

每次 sentence 变化都使用纯函数重算，无法定位时写空字符串，避免保留旧值。

- [x] **Step 4: 运行 word-agent 全量测试**

Run: `.venv/bin/pytest -q`
Expected: all tests pass。

### Task 3: 全量回填与质量闸门

**Files:**
- Create: `word_select_dashboard/word-agent/scripts/backfill_best_sentence_cloze.py`
- Create: `outputs/word_clean_best_sentence_cloze_unmatched.csv`（仅在存在失败记录时）

- [x] **Step 1: 实现 dry-run 和事务回填**

脚本加载现有 word-agent 数据库配置，统计 exact、inflected、phrase-anchor、multi 和 unmatched；任何 unmatched 默认阻止写入并输出 CSV。

- [x] **Step 2: 对 22,375 条实时数据执行 dry-run**

Run: `.venv/bin/python scripts/backfill_best_sentence_cloze.py --dry-run`
Expected: `unmatched=0`，所有记录可还原原句。

- [x] **Step 3: 执行实际回填**

Run: `.venv/bin/python scripts/backfill_best_sentence_cloze.py`
Expected: 事务更新 22,375 条。

- [x] **Step 4: 执行数据库验收**

检查字段存在、空值与空字符串为 0、挖空标记缺失为 0、答案仍暴露为 0，并抽样 `value/values`、过去式、大小写和重复词。

### Task 4: 最终验证

**Files:**
- Verify: `word_select_dashboard/word-agent`
- Verify: `rob_english_word_back/db/word_clean_best_sentence.sql`

- [x] **Step 1: 运行目标测试和全量测试**

Run: `.venv/bin/pytest -q`
Expected: all tests pass。

- [x] **Step 2: 检查格式和差异**

Run: `.venv/bin/ruff check src tests scripts/backfill_best_sentence_cloze.py`
Expected: no lint errors。

Run: `git diff --check`
Expected: no whitespace errors。
