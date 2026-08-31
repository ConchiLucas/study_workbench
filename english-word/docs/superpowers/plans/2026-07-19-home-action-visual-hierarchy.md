# 7002 首页操作按钮视觉层级 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 仅保留“开始匹配”的彩色高亮，将其他六个入口统一为低干扰的深色次要按钮。

**Architecture:** 在 `HomeView.vue` 中为所有非匹配操作添加统一的 `secondary-action-btn` 类，通过一组共享 CSS 管理背景、描边、悬停和禁用状态；现有业务类名继续保留，不改事件处理。

**Tech Stack:** Vue 3、TypeScript、Vitest、CSS

---

### Task 1: 锁定按钮视觉层级

**Files:**
- Create: `rob_english_word_front/src/views/HomeView.visual-hierarchy.test.ts`

- [ ] **Step 1: 写失败测试**

读取 `HomeView.vue`，断言模板中六个次要按钮包含 `secondary-action-btn`，两个“开始匹配”按钮不包含该类；断言 CSS 定义统一次要样式。

- [ ] **Step 2: 验证测试失败**

Run: `npx vitest run src/views/HomeView.visual-hierarchy.test.ts`
Expected: FAIL，模板尚未使用统一次要按钮类。

### Task 2: 实现统一次要样式

**Files:**
- Modify: `rob_english_word_front/src/views/HomeView.vue`

- [ ] **Step 1: 修改模板类名**

为“难度训练”“错题集”“难度选择”“单人训练”“查看答题结果”“已掌握单词”添加 `secondary-action-btn`。

- [ ] **Step 2: 合并 CSS**

保留 `.match-btn` 紫色渐变；删除各次要业务类的独立渐变和阴影，改用 `.secondary-action-btn` 的深色半透明背景、细描边和轻量悬停。

- [ ] **Step 3: 验证聚焦测试通过**

Run: `npx vitest run src/views/HomeView.visual-hierarchy.test.ts`
Expected: PASS。

### Task 3: 全量验证

**Files:**
- Verify: `rob_english_word_front`

- [ ] **Step 1: 运行完整测试**

Run: `npm test`
Expected: 全部测试通过。

- [ ] **Step 2: 运行生产构建**

Run: `npm run build`
Expected: TypeScript 与 Vite 构建成功。

- [ ] **Step 3: 页面检查**

打开 `http://localhost:7002/home`，确认首页和难度训练页面仅“开始匹配”保留彩色高亮。
