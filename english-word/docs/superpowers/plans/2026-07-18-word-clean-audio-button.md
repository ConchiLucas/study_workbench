# 去重单词表播放按钮 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在“去重单词表”的最佳例句右侧增加播放/暂停按钮，通过现有 MinIO URL 播放 TTS。

**Architecture:** 抽取一个纯函数统一判断成功状态与有效音频 URL；`WordCleanPage` 保持自己的单实例 `Audio` 播放状态，避免与其他页面耦合。按钮复用现有样式和 Ant Design 图标。

**Tech Stack:** React 19、TypeScript、Ant Design、Node.js 内置测试运行器、Vite。

---

### Task 1: 可播放 URL 规则

**Files:**
- Create: `word_select_dashboard/web-react/src/utils/wordAudio.ts`
- Create: `word_select_dashboard/web-react/test/wordAudio.test.ts`

- [x] **Step 1: 写失败测试**

测试成功状态返回去空格后的 URL，失败状态或空 URL 返回 `null`。

- [x] **Step 2: 验证测试失败**

Run: `node --experimental-strip-types --test test/wordAudio.test.ts`
Expected: FAIL，提示找不到 `src/utils/wordAudio.ts`。

- [x] **Step 3: 实现最小纯函数**

新增 `playableBestSentenceAudioURL(status, objectURL)`，只接受 `success` 和非空 URL。

- [x] **Step 4: 验证测试通过**

Run: `node --experimental-strip-types --test test/wordAudio.test.ts`
Expected: 3 tests PASS。

### Task 2: 去重单词表播放交互

**Files:**
- Modify: `word_select_dashboard/web-react/src/App.tsx`

- [x] **Step 1: 在 `WordCleanPage` 增加独立播放状态**

维护当前播放/加载的 `word_clean.id` 与唯一 `Audio` 引用；组件卸载时释放音频。

- [x] **Step 2: 实现播放、暂停和错误提示**

播放前调用纯函数取得 URL；切换单词先停止旧音频；结束、报错或重复点击时重置状态。

- [x] **Step 3: 在最佳例句右侧渲染按钮**

使用 `PlayCircleOutlined` / `PauseCircleOutlined`，无可用 TTS 时禁用并显示提示。

- [x] **Step 4: 翻页、筛选和刷新时停止当前播放**

这些操作改变列表内容前统一停止音频，避免不可见行继续播放。

### Task 3: 验收

**Files:**
- Verify: `word_select_dashboard/web-react`

- [x] **Step 1: 运行目标测试**

Run: `node --experimental-strip-types --test test/wordAudio.test.ts`
Expected: PASS。

- [x] **Step 2: 运行前端构建**

Run: `npm run build`
Expected: TypeScript 与 Vite 构建成功。

- [x] **Step 3: 浏览器抽样播放**

真实列表接口返回可播放状态与 MinIO 地址；抽样 WAV 经前端代理返回 HTTP 200 和 `audio/wav`。自动化浏览器标签页连接异常，按钮交互由 TypeScript 构建和状态逻辑审查覆盖。
