# 题目管理（识字任务题包）设计

日期：2026-08-29  
状态：待实现  
范围：study-content-admin 第一版；共享库 `study_workbench`

## 背景

内容后台已有识字 / 拼音 / 算术等**素材**管理；孩子端每日计划由 workbench 按掌握度自动拼 **10 道题**。运营侧缺少「按任务维度」预组题目的能力。

本设计在内容后台新增 **题目管理**：以**任务（题包）**为单位，从正式题库按识字组抽出固定 10 道不重复题，供后续孩子端挂载。第一版**只做后台**，不接孩子端。

## 目标

- 顶栏菜单「题目管理」，可创建 / 列表 / 预览 / 发布识字任务。
- 每个任务绑定一个识字组，自动抽满 **10** 道题，**题目不重复**。
- 题型沿用现网识字题库（`listen1` / `listen2` 等已入库题），不引入三种技能新题型。
- 数据落共享库，为拼音 / 算术 / 英语与孩子端接入预留字段。

## 非目标（v1）

- 拼音、算术、英语任务
- 孩子端入口、替代或并存今日计划
- 跨组混抽、人工逐题勾选
- 改写 / 重新 seed `questions` 表
- 三种技能题（听字图 / 字图→义图 / 义图→字）写入正式题库

## 决策摘要

| 项 | 选择 |
|---|---|
| 任务形态 | 内容后台题包（方案 A），独立表，不复用 `study_plans` |
| 抽题范围 | 单个识字 `module_code`（组） |
| 题量 | 固定 10 |
| 去重 | `question_id` 在同一 task 内唯一 |
| 题源 | 现网 `questions`（识字 listen 变体） |
| 孩子端 | v1 不做 |

## 数据模型

### `question_tasks`

| 字段 | 类型 | 说明 |
|---|---|---|
| id | BIGSERIAL PK | |
| subject_code | VARCHAR(30) NOT NULL | v1 固定 `literacy` |
| title | VARCHAR(80) NOT NULL | 默认「识字 · {组名}」，可改 |
| module_code | VARCHAR(50) NOT NULL | 识字组，如 `g3` |
| module_name | VARCHAR(50) NOT NULL DEFAULT '' | 冗余展示 |
| target_count | INT NOT NULL DEFAULT 10 | v1 恒为 10 |
| status | VARCHAR(20) NOT NULL | `draft` \| `published` |
| created_at / updated_at | TIMESTAMPTZ | |

索引建议：`(subject_code, status)`、`(module_code)`。

### `question_task_items`

| 字段 | 类型 | 说明 |
|---|---|---|
| id | BIGSERIAL PK | |
| task_id | BIGINT NOT NULL FK → question_tasks ON DELETE CASCADE | |
| seq | INT NOT NULL | 1…target_count |
| kp_id | BIGINT NOT NULL | 知识点 |
| question_id | BIGINT NOT NULL | `questions.id` |
| UNIQUE(task_id, seq) | | |
| UNIQUE(task_id, question_id) | | 同一任务题目不重复 |

不在 items 上冗余 stem/options：详情接口 join `questions`（及需要的 KP 标题）。

## 抽题规则

1. 输入：`subject_code=literacy` + `module_code`。
2. 查询该组下全部 `knowledge_points`，再取关联 `questions`（`JOIN` 学科为 literacy）。
3. 候选集合按 `question_id` 去重。
4. 打乱后取前 `target_count`（10）条；**不足 10** → 失败，返回明确错误（含当前可出题数 N）。
5. 写入 `question_task_items`，`seq` 从 1 递增；任务 `status=draft`。
6. **重新抽题**（仅 draft）：删除旧 items，按同上规则重抽并写回。
7. **发布**：`draft` → `published`；published 禁止 reshuffle / 改 items；可 **撤回** → `draft` 后再抽。
8. **删除**：仅允许 `draft`。

说明：同一字的 `listen1` 与 `listen2` 是不同 `question_id`，允许同时进入同一任务；同一 `question_id` 绝不重复。

## API

前缀：`/api/v1/question-tasks`（content-admin）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/` | 列表；query：`subject`、`status` 可选 |
| POST | `/` | body：`{ subjectCode, moduleCode, title? }` → 抽题建 draft |
| GET | `/:id` | 详情 + items（含题目展示字段） |
| POST | `/:id/reshuffle` | draft 重抽 |
| POST | `/:id/publish` | → published |
| POST | `/:id/unpublish` | → draft |
| DELETE | `/:id` | 仅 draft |

详情 item 建议返回：`seq, kpId, questionId, charText, code, stem, options, answerIndex, speech`（从 `questions` / KP 组装），便于后台预览播放。

## 后台 UI

- 路由：`/questions`；顶栏 NavLink「题目管理」。
- **列表**：标题、学科、组、题数、状态、更新时间；操作查看 / 重抽 / 发布 / 撤回 / 删除（按状态显隐）。
- **新建**：选识字组 → 可选改标题 →「生成 10 题」→ 进详情。
- **详情**：按序预览 10 题（题干、选项、正确项、变体 code、读音按钮若有 speech）；draft 显示「重新抽题」。

视觉与交互对齐现有 literacy / math 列表页风格（暗色 admin），不做独立设计系统。

## 与现有系统关系

| 系统 | 关系 |
|---|---|
| `questions` / `knowledge_points` | **只读**抽题来源 |
| `literacy_assets` | 不依赖；预览以正式题为准 |
| `study_plans` / kid-app | v1 **不接入**；后续可读 `published` 任务发题 |
| 识字三种技能预览 | 仍为本地预览，与本功能无关 |

## 后续扩展（非 v1）

- `subject_code` ∈ {pinyin, math, english}，按对应 module 抽题。
- 孩子端：独立任务列表或替换今日计划，读取 `published` + items。
- 人工微调 items、跨组池、技能题型入库。

## 验收标准

1. 顶栏可进入题目管理；可对某识字组生成 draft 任务且恰有 10 条 item。
2. 同一 task 内无重复 `question_id`。
3. 组内可出题 &lt; 10 时创建失败并提示 N。
4. draft 可重抽、可删；published 只读，可撤回后再改。
5. 详情能展示听音点字题面（stem / options / 正确项）。
6. 不修改现有每日计划与孩子端行为。
