# 7003 挖空练习全屏关闭导航 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 移除 7003 单独训练及其子页面的左上角返回按钮，统一使用右上角叉号并按页面层级关闭。

**Architecture:** 使用一个共享 React 关闭按钮组件承载样式和可访问性；各页面仍通过现有 React 状态关闭自身，避免引入路由历史依赖。Escape 按照最上层优先级调用同一组关闭行为。

**Tech Stack:** React 19、TypeScript、Vitest、Testing Library、CSS

---

### Task 1: 定义共享关闭按钮行为

**Files:**
- Create: `rob_english_word_cloze_web/src/components/FullscreenCloseButton.tsx`
- Create: `rob_english_word_cloze_web/test/fullscreenCloseButton.test.tsx`
- Modify: `rob_english_word_cloze_web/src/styles/app.css`

- [ ] **Step 1: 写失败测试**

测试渲染 `aria-label="关闭单独训练"` 的叉号按钮，点击后断言 `onClose` 只调用一次。

- [ ] **Step 2: 验证测试因组件不存在而失败**

Run: `npm test -- --run test/fullscreenCloseButton.test.tsx`
Expected: FAIL，提示无法导入 `FullscreenCloseButton`。

- [ ] **Step 3: 实现最小共享组件与样式**

组件接收 `label`、`onClose` 和可选 `disabled`，渲染 `.fullscreen-close-button`；样式固定在全屏容器右上角。

- [ ] **Step 4: 验证组件测试通过**

Run: `npx vitest run test/fullscreenCloseButton.test.tsx`
Expected: PASS。

### Task 2: 替换单独训练及子页面返回按钮

**Files:**
- Modify: `rob_english_word_cloze_web/src/components/PracticeLaunchers.tsx`
- Modify: `rob_english_word_cloze_web/src/App.tsx`
- Modify: `rob_english_word_cloze_web/test/practiceLaunchers.test.tsx`
- Create: `rob_english_word_cloze_web/test/fullscreenCloseNavigation.test.ts`

- [ ] **Step 1: 写失败测试**

启动器测试要求出现“关闭单独训练”且不存在“返回”；导航契约测试要求难度、句子、结果和答题页面使用共享关闭按钮。

- [ ] **Step 2: 验证测试因旧返回按钮而失败**

Run: `npx vitest run test/practiceLaunchers.test.tsx test/fullscreenCloseNavigation.test.ts`
Expected: FAIL，仍能找到“返回”或缺少关闭标签。

- [ ] **Step 3: 实现页面关闭层级**

将 `onBack` 改为 `onClose`；三个子页面分别关闭自身；答题页面退出到单独训练；Escape 按“答题 → 结果 → 句子 → 难度 → 单独训练”顺序关闭最上层。

- [ ] **Step 4: 调整头部布局**

删除返回按钮占位，保留居中标题和刷新操作，并为右上角叉号留出空间。

- [ ] **Step 5: 验证导航测试通过**

Run: `npx vitest run test/practiceLaunchers.test.tsx test/fullscreenCloseNavigation.test.ts`
Expected: PASS。

### Task 3: 全量验证

**Files:**
- Verify: `rob_english_word_cloze_web`

- [ ] **Step 1: 运行全部前端测试**

Run: `npm test`
Expected: 全部测试通过，无失败。

- [ ] **Step 2: 运行生产构建**

Run: `npm run build`
Expected: TypeScript 与 Vite 构建成功，退出码为 0。

- [ ] **Step 3: 检查变更范围**

Run: `git diff --check && git diff -- rob_english_word_cloze_web`
Expected: 无空白错误，改动仅覆盖关闭导航、样式与测试。
