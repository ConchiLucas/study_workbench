# 识字字详情 · 三题型答题情况

**日期：** 2026-08-29  
**范围：** parent-dashboard 字详情侧栏（`KpDetailDrawer`）+ `GET .../knowledge-points/:kpId`  
**前提：** 矩阵已支持 `listen_glyph` / `glyph_sense` / `sense_char` 与 `mastery_skills`  
**Mockup：** `docs/superpowers/mockups/2026-08-29-literacy-kp-detail.html`

## 目标

家长点开某个字时，能看到：
1. 听 / 义 / 字 各自掌握状态与练习摘要  
2. 作答记录带题型标签  
3. 字级汇总与「标记已学会」行为保持现有语义（三技能全过才算完全掌握）

非识字学科详情 UI / 字段行为不变。

## 布局（仅 `subject_code === 'literacy'`）

自上而下：

1. 标题：学科 · 组 / 字  
2. 字级状态条：`RollupSkills` 后的 `status`（与矩阵字卡一致）  
3. **题型掌握**：三张精简卡（固定顺序 听→义→字）  
   - 题型短名 + 全称（听音选字 / 看字选义 / 看义选字）  
   - 状态标签  
   - `练 N 次 · 正确率 M%`  
4. **字级汇总**：现有六宫格（练习次数、正确率、当前连对、历史最佳、下次复习、首次掌握）  
5. 标记为已学会 / 撤销标记（逻辑不变：标记写满三技能并 rollup）  
6. **作答记录**：时间倒序；每行前缀 pill  
   - 有 `skill_code` → 听 / 义 / 字  
   - `source === 'parent_mark'` →「标」  
   - 其它无 skill → 不显示 pill

## API

扩展现有详情响应，不新增路由、不改表。

### `skills`（识字时必填数组；其它学科省略）

与矩阵 `MatrixSkill` 同形：

| 字段 | 说明 |
|------|------|
| `code` | `listen_glyph` \| `glyph_sense` \| `sense_char` |
| `status` | 展示态（含 `review_due`） |
| `attempts` | 该技能练习次数 |
| `accuracy` | 0–1 |

缺行按未开始填充；顺序固定为 `LiteracySkills`。

字级 `status` 在识字时由三技能 `RollupSkills` 覆盖查询到的 KP 行状态，避免与矩阵不一致。

### `history[]` 增补

| 字段 | 说明 |
|------|------|
| `skill_code` | 可选。`attempts.question_id` → `questions.code` → `SkillFromQuestionCode`；家长标记或无题 → 空 |

旧码 `listen1` / `listen2` 仍映射为 `listen_glyph`。

## 前端

- `types.ts`：`KpDetail.skills?`、`HistoryItem.skill_code?`  
- `KpDetailDrawer`：识字分支渲染题型卡 + 历史 pill；其它学科沿用现状  
- 标签映射与矩阵一致：听 / 义 / 字  

## 测试

- `KpDetail`：识字返回三条 `skills`；有作答时 history 带正确 `skill_code`；家长标记无 skill  
- 非识字：无 `skills`（或空），history 可无 `skill_code`  
- FE：有 skills 时渲染三卡与 pill（可用现有手动/冒烟）

## 非目标

- 不同步 content-admin 字图/义图到详情  
- 不在详情内嵌答题器  
- 不改矩阵、不改 kid-app 作答协议  

## 决策摘要

- 方案：扩展详情 API（非整页替换、非纯前端硬凑）  
- 题型卡信息量：精简（状态 + 次数 + 正确率）  
- 作答记录与题型状态卡都要  
