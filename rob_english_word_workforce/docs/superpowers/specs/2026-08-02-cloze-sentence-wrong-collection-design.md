# 完形句子错题集与统一复习进度设计

## 目标

为完形练习增加用户侧句子错题集，并统一覆盖两个答题入口：

- 首页“开始答题”；
- “单独训练”中的句子练习。

一道题包含多个空位时，只要任意一个目标单词拼写错误，整道句子就进入错题集一条记录。
句子复习沿用单词错题的节奏：立即复习、7 天复习、15 天复习，三个到期阶段全部答对后完成。

本设计只复用单词错题的状态机思想，不复制其逐词数据模型。句子以
`(user_id, cloze_item_id)` 为稳定主体，错误次数按错误提交次数累计，不按错误空位数累计。

## 当前实现与缺口

两种入口共用 `POST /api/cloze-practice/answers`。后端会按空位顺序比较全部答案，任意一项
不匹配时把整条 `sentence_cloze_answer_record.is_correct` 写为 `false`，因此现有整句判定已经
满足入集规则。

当前复习状态分成两套：

- `source = word-agent` 的句子由 `wrong_word_review_progress` 按单词、空位独立推进；
- `source = best-sentence-practice` 的单独训练句子由
  `sentence_cloze_review_schedule` 按整句推进。

现有首页 `review-due` 只读取第一类任务。单独训练答错后虽然会建立整句计划，但不会进入首页
到期复习队列；整句计划完成三轮后还会被物理删除，导致“从未答错”和“已经完成”无法区分。

当前答题请求也没有提交幂等键。网络重试或重复点击可能重复插入答题流水、累计错误次数或推进
复习阶段。实现错题集时一并补齐该边界。

## 方案选择

### 方案一：动态拼接现有两套状态

列表临时 UNION 单词进度和整句计划，不修改数据模型。

优点是开发量小；缺点是同一列表中的阶段、完成、错误次数和唯一主体语义不同，无法形成可靠的
句子错题事实，不采用。

### 方案二：新增第三张句子错题进度表

新增 `sentence_wrong_review_progress`，再逐步停用旧整句计划。

边界清楚，但会在切换期形成两张相同主体、相同调度职责的表，增加双写和历史迁移风险，不采用。

### 方案三：升级现有整句计划为持久进度

扩展 `sentence_cloze_review_schedule`，让所有句子来源在答错后统一维护整句进度。完成后保留
`completed` 记录，不再删除。

现有表已经具备 `(user_id, cloze_item_id)` 唯一键、阶段、下次复习时间和最近答题关联，升级成本
和迁移风险最低，采用此方案。

## 领域边界

### 句子进度

句子进度负责：

- 是否出现在句子错题集；
- 整句错误次数、首次和最近答错时间；
- 整句立即、7 天、15 天的复习阶段；
- 是否完成句子复习。

### 单词进度

`wrong_word_review_progress` 继续负责：

- 单词错题集；
- 一条多空句中每个单词的独立阶段；
- 单词完成后的掌握状态。

`word-agent` 句子提交时，句子进度与单词进度在同一事务内分别更新。整句任一空错误会重置
句子进度；单词进度仍只重置错误空位，并推进已经到期且答对的空位。两者语义不同，不互相覆盖。

## 数据模型

### `sentence_cloze_review_schedule`

保留现有主键和唯一键，新增或调整以下字段：

| 字段 | 规则 |
| --- | --- |
| `status` | `active` 或 `completed` |
| `review_stage` | 0 立即、1 等待 7 天、2 等待 15 天、3 已完成 |
| `next_review_time` | 活跃阶段的下次到期时间；完成时为空 |
| `wrong_count` | 整句错误提交次数，每次错误提交只增加 1 |
| `first_wrong_time` | 首次进入错题集时间 |
| `last_wrong_time` | 最近错误提交时间 |
| `last_wrong_answer_record_id` | 最近一次错误答题流水 |
| `last_answer_record_id` | 最近一次成功更新进度的答题流水 |
| `last_correct_time` | 最近一次有效正确复习时间 |
| `completed_time` | 完成第三阶段时间 |

现有 `correct_streak` 暂时保留并与 `review_stage` 同步，避免一次迁移同时扩大到所有旧查询；新代码
以 `review_stage` 为阶段事实。增加索引：

- `(user_id, status, next_review_time)`；
- `(user_id, last_wrong_time DESC)`。

### `sentence_cloze_answer_record`

新增：

| 字段 | 规则 |
| --- | --- |
| `submission_key` | 客户端每次提交生成的 UUID |
| `practice_context` | `review` 或 `solo`，记录实际入口 |
| `action_type` | `answer` 或 `reveal` |
| `wrong_blank_indexes_json` | 本次错误空位下标快照 |

增加唯一约束 `(user_id, submission_key)`。同一 key 重试时返回已有答题结果，不重复写流水、不重复
通知 Agent，也不重复推进任何进度。

前端正常提交和“显示答案”都必须传 key。“显示答案”按未掌握处理，`action_type = reveal`，整句进入
或重置错题集。

## 答案比较

后端建立一个共享的按下标比较结果，供整句判定、`wrong_blank_indexes_json` 和逐词进度共同使用：

- 保留答案数组原始下标，不过滤中间空字符串；
- 对每个字符串执行 `trim + NFKC + Locale.ROOT lowercase`；
- 答案数量不同或任一位置不匹配时整句错误；
- 每个错误位置只记录一次；
- 单词进度与整句进度必须消费同一份比较结果，不能使用不同规范化规则。

前端仍要求所有输入框填写后才能正常提交；服务端规则负责处理直接 API 调用和旧客户端。

## 状态机

| 当前状态 | 事件 | 结果 |
| --- | --- | --- |
| 无进度 | 整句正确 | 不创建错题进度 |
| 无进度 | 任意空错误或显示答案 | active、stage 0、立即到期、wrong_count=1 |
| active stage 0 且已到期 | 整句正确 | stage 1，7 天后到期 |
| active stage 1 且已到期 | 整句正确 | stage 2，15 天后到期 |
| active stage 2 且已到期 | 整句正确 | completed、stage 3、next_review_time 为空 |
| active 但尚未到期 | 整句正确 | 不推进 |
| 任意 active 阶段 | 整句错误 | 重置 stage 0，立即到期，wrong_count+1 |
| completed | 整句错误 | 原记录重新激活为 stage 0，wrong_count+1 |
| completed | 整句正确 | 不改变完成状态 |

所有时间和到期判断统一使用数据库时钟。错误 upsert、正确推进和完成都使用带阶段、到期时间、
上一答题流水条件的单条原子 SQL，避免“先 select 再 update/delete”的竞态。

## 到期任务合并

首页“开始答题”不能只改为查询整句进度，因为现有 `word-agent` 句子在第一次完形作答前，仍需要
作为单词错题的承载题出现。

`GET /api/cloze-practice/tasks/review-due` 合并两个集合：

1. `sentence_cloze_review_schedule.status = active` 且已经到期的整句；
2. `wrong_word_review_progress` 中已经到期的单词所关联的 `word-agent` 句子。

按 `cloze_item_id` 去重，取两个集合中更早的到期时间，再按到期时间和题目 ID 稳定排序。这样：

- 尚未在完形中答错的单词复习句仍可正常出现；
- “开始答题”中答错的句子立即进入句子错题集；
- 单独训练答错的句子也能进入首页到期复习；
- 同一题同时因单词和句子到期时只出现一次。

## API

### 错题列表

新增 `GET /api/cloze-practice/wrong-sentences`：

- `status=active|completed`，默认 `active`；
- `source=all|review|solo`；
- `availability=all|due|waiting`；
- `keyword` 匹配英文句子、翻译和目标词；
- `sort=nextReview|recent|wrongCount`；
- `page`、`size`，size 上限 100。

响应包含 `items/total/current/pages/summary`。`summary` 至少返回：

- `activeCount`；
- `dueCount`；
- `stage1Count`；
- `stage2Count`；
- `completedCount`。

列表项包含：

- `progressId`、`clozeItemId`；
- 挖空句、完整句、中文翻译；
- 目标词数组、最近错误空位、错误空位数量；
- 最近错误入口和内容来源；
- 难度标签；
- `status/reviewStage/nextReviewTime`；
- `wrongCount/firstWrongTime/lastWrongTime`；
- 最近耗时。

### 错题详情

新增 `GET /api/cloze-practice/wrong-sentences/{progressId}`，返回：

- 列表项全部内容；
- 每个空位的正确单词、最近正误、可用时的词义与单词复习阶段；
- 最近五次答题的时间、整句结果、耗时、入口和操作类型；
- 固定的立即、7 天、15 天、完成阶段信息。

用户页面不返回或展示内部模型、Provider、事件 ID、跨库追溯字段。原始用户答案继续保存在流水中，
但错题集 UI 不展示“你的答案”。

### 答题提交

扩展 `POST /api/cloze-practice/answers` 请求：

- `submissionKey`；
- `practiceContext`；
- `actionType`。

部署兼容期允许旧客户端不传 key，并按旧方式处理；新前端发布后再把 key 改为必填契约。

### 统计

扩展 `/api/cloze-practice/stats`：

- `activeWrongSentences`；
- `dueReviewTasks`，按合并并去重后的到期题精确计数。

首页不再使用最多 10 条的到期数组长度冒充真实到期数量。

## 前端设计

### 首页

保留“开始答题”为主按钮。下方改为两个等宽次级按钮：

```text
[                 开始答题                 ]
[          错题集 N          ][      单独训练      ]
```

窄屏下三个按钮垂直排列。“错题集”始终可进入，数量为 activeCount；没有错题时展示空状态。

### 错题集覆盖层

新增独立 `WrongSentenceCollection` 组件，不把完整列表继续写入 `App.tsx`。组件负责查询参数、加载、
分页、展开状态、空状态和错误状态；`App.tsx` 只管理打开与关闭。

桌面主表字段：

1. 最近答错时间；
2. 句子与翻译；
3. 错词数；
4. 入集来源；
5. 复习阶段；
6. 下次复习；
7. 错误次数；
8. 展开/收起。

展开区包含：

- 还原后的完整英文句子和中文翻译；
- 逐空正确词、最近正误、词义和可用时的单词复习状态；
- 当前阶段高亮的“立即 → 7 天 → 15 天 → 完成”时间轴；
- 首次答错、最近答错、最近耗时、来源难度；
- 最近五次答题的时间、结果、耗时和入口。

移动端使用卡片布局，不复用现有 980px 固定宽表横向滚动。第一版不提供行内“立即训练”；所有到期
复习仍从首页“开始答题”进入，避免提前作答误推进和新的返回层级。

### 关闭与导航

错题集是复习首页之上的全屏覆盖层：

- 叉号和 Escape 只关闭错题集并返回复习首页；
- 不改变单独训练、难度选择、答题结果的原有父子关系；
- 打开错题集时不能同时展示其他全屏覆盖层；
- 关闭时不修改当前难度或答题批次。

## 错误处理

- 列表加载失败保留当前筛选，显示重试按钮；不回退到旧 `answered?status=wrong` 数据。
- 详情加载失败只影响当前展开行，不关闭整个错题集。
- 重复 submission key 返回已有结果。
- 同一题不同 submission key 并发提交时，进度 SQL 保证一个阶段最多推进一次。
- 通知 Python Agent 失败不回滚已经提交的答题流水和句子进度，保持现有异步边界。
- 登录失效继续走统一退出逻辑。

## 历史迁移

迁移脚本幂等执行：

1. 现有 `sentence_cloze_review_schedule` 行标记为 active，保留已有阶段和时间；
2. 根据答题流水补齐 wrong_count、first_wrong_time、last_wrong_time 和最近错误流水；
3. 只为“最新一次作答仍为错误、但没有整句进度”的历史题建立 stage 0 立即到期进度；
4. 不把所有曾经答错的题重新激活，避免已经掌握的旧题批量回流；
5. 无法可靠推断的历史完成状态不猜测，继续保留在答题历史中；
6. 修正 DDL 注释和前端旧文案，把“1 天、7 天、15 天”统一为“立即、7 天、15 天”。

## 测试

### Java 与 PostgreSQL

- 两种来源任一空错误都会创建同一个句子进度；
- 多空句错一个或多个空位都只增加一次 wrong_count；
- 正确答案不会首次创建错题进度；
- stage 0/1/2 按立即、7 天、15 天推进，stage 3 保留完成行；
- 未到期正确不推进，任意错误立即重置；
- 已完成再次错误原地复活；
- 相同 submission key 不重复插入或推进；
- 不同 key 并发提交不能越级推进；
- 句子比较和逐词比较使用同一规范化结果；
- 到期查询合并两种来源并按 cloze_item_id 去重；
- 列表分页总数、筛选、排序和详情字段正确；
- 历史迁移重复执行结果不变。

### React

- 首页显示主按钮和两个等宽次按钮；
- 错题集数量来自精确统计；
- 覆盖层加载、筛选、分页、展开、空状态和局部错误正常；
- 页面不展示“你的答案”、模型或内部追溯字段；
- 叉号和 Escape 返回复习首页；
- 桌面表格和移动卡片均可用；
- submission key 在未知网络结果重试时复用，成功后下一次提交换新 key；
- 两种练习模式仍提交正确的 practiceContext。

### 回归

- 单词错题三阶段和逐词推进不变；
- `word-agent` 句子首次练习仍能从首页出现；
- 单独训练难度、句子列表和答题结果不受影响；
- 答题历史和正确率保持原口径；
- Java 全量测试、React 全量测试、生产构建和真实 PostgreSQL 行为测试通过；
- 按工作空间 Context Router 或降级部署脚本完成运行态验证。

## 实施拆分

1. 数据库字段、幂等约束和原子进度 Mapper；
2. 共享答案比较、提交事务和两种进度并行更新；
3. 合并到期查询、错题分页/详情/统计接口；
4. 首页入口和独立错题集前端组件；
5. 历史迁移、完整测试、构建和运行态验收。

本设计不包含管理后台错题集重构、通用多类型复习引擎、行内单题训练或删除历史答题流水。
