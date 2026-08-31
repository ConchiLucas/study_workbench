---
title: 英语完形任务、答题与复习链路
summary: 说明完形前端从获取任务、提交答案到复习进度、历史统计和后台结果查看的完整流转。
---

# 英语完形任务、答题与复习链路

## 主路径

```text
React 完形前端
  -> /api/cloze-practice/tasks/next | pending | review-due | difficulty-batch
  -> ClozePracticeController / Service / Mapper
  -> rob_english_word 完形任务与复习进度
  -> 用户填写挖空答案
  -> POST /api/cloze-practice/answers
  -> 按空位统一比较并写入幂等答题流水
  -> 更新句子错题进度；word-agent 同时更新独立单词进度
  -> stats / history / wrong-sentences 展示
  -> Go 管理服务跨库读取结果
  -> React 运营后台展示 cloze-results
```

## 练习模式

- 支持下一题、待练、到期复习、已答列表和按难度批量练习。
- 用户单人难度偏好通过 `preferences` 接口读取和保存。
- 前端用 `practiceMode.ts`、`soloDifficulty.ts` 和 `PracticeLaunchers.tsx` 管理模式与入口。
- 句子和单词音频是增强能力，不应改变答案与复习进度事实。
- “开始答题”和“单独训练”统一提交到 `/answers`，由 `practiceContext=review|solo`
  记录实际入口；“显示答案”使用 `actionType=reveal` 并按整句错误处理。

## 答题判定与幂等

- 后端按答案数组下标比较，每个值统一执行 `trim + NFKC + Locale.ROOT lowercase`。
- 答案数量不同或任一空位不匹配时，整句判错并进入或重置句子错题进度；错误空位下标
  固化到答题流水，句子进度的 `wrong_count` 每次错误提交只增加一次。
- 新前端为每次提交生成 `submissionKey`。未知网络结果重试复用同一个 key，成功后下一次作答
  更换 key；数据库以 `(user_id, submission_key)` 的非空唯一索引保证不重复写流水、通知或推进。
- 兼容期允许旧客户端不传 key。

## 句子与单词进度边界

- `sentence_cloze_review_schedule` 是全来源句子错题的权威进度，以
  `(user_id, cloze_item_id)` 唯一，完成后保留 `status=completed, review_stage=3`。
- 任一空位错误都会把句子重置为 active stage 0 并立即到期；已完成句子再次答错会重新激活。
- `word-agent` 题目仍并行维护 `wrong_word_review_progress`。每个单词按自己的空位独立推进，
  只有逐词第三阶段完成时才写入单词掌握状态；句子进度与单词进度互不替代。
- 非 `word-agent` 题目没有逐词进度，只维护句子错题状态。

## 句子复习状态机

| 当前状态 | 有效事件 | 结果 |
| --- | --- | --- |
| 无句子进度 | 整句错误或显示答案 | active stage 0，立即到期 |
| active stage 0 且到期 | 整句正确 | stage 1，7 天后到期 |
| active stage 1 且到期 | 整句正确 | stage 2，15 天后到期 |
| active stage 2 且到期 | 整句正确 | completed stage 3，不再到期 |
| active 尚未到期 | 整句正确 | 不提前推进 |
| active 任意阶段 | 整句错误 | 重置 stage 0，错误次数加一 |
| completed | 整句错误 | 重新激活 stage 0 |
| completed | 整句正确 | 保持 completed |

正确推进和错误重置均由带到期时间、阶段和上一答题流水条件的原子 SQL 完成，数据库时钟
决定是否到期。

## 到期任务与错题集

- `/tasks/review-due` 合并两类任务：已经到期的句子进度，以及已经到期的逐词进度所关联的
  `word-agent` 载体句；按 `cloze_item_id` 去重，取更早到期时间后稳定排序并限量。
- `/stats` 返回 `activeWrongSentences` 和去重后的 `dueReviewTasks` 精确数量，首页不再用最多
  10 条的任务数组长度充当真实数量。
- `/wrong-sentences` 只以持久句子进度为主集合，支持 active/completed、入口、到期性、关键词、
  排序和分页；`/wrong-sentences/{progressId}` 在校验用户所有权后返回逐空状态和最近五次结果。
- 用户错题集不返回或展示原始用户答案、模型、Provider、事件 ID 或跨库追踪字段。

## 数据边界

- Java 是答题判定和复习进度的权威写入方。
- Go 管理服务以运营查看为主，不应绕过 Java 修改用户练习状态。
- 复习进度、错词事件和句子生成之间的关联应使用稳定业务键，不依赖页面顺序。

## 证据路径

- `rob_english_word_cloze_web/src/App.tsx`
- `rob_english_word_cloze_web/src/lib/api.ts`
- `rob_english_word_back/src/main/java/com/robword/controller/ClozePracticeController.java`
- `rob_english_word_back/src/main/java/com/robword/service/ClozePracticeService.java`
- `rob_english_word_back/db/wrong_word_review_progress.sql`
- `word_select_dashboard/server/router/system/sys_cloze_result.go`
- `word_select_dashboard/web-react/src/lib/clozeResultApi.ts`

## 运行时待核对

- 实际部署环境的数据库时区与前端本地时间展示是否一致。
- PostgreSQL 历史回填行为测试需在设置 `RUN_POSTGRES_INTEGRATION=true` 和测试库连接时执行；
  默认全量单测会安全跳过该外部依赖用例。
