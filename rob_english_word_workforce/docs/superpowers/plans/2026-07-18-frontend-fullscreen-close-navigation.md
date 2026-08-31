# 7002 前端全屏关闭导航 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 7002 前端所有二级、三级界面的左上角返回入口统一为右上角 `×`，并让点击与 Escape 都回到明确的上一级。

**Architecture:** 新增无路由依赖的共享关闭按钮和 Escape 组合函数，页面自己提供关闭回调。Home 内部全屏层使用响应式状态逐层关闭，路由页面使用显式 `/home` 或携带 `openTrainingSetup` 的 `/home`，详情弹窗只关闭自身。

**Tech Stack:** Vue 3、TypeScript、Vue Router 4、Vite 5、Vitest、Vue Test Utils、jsdom

---

## 文件结构

- Create: `rob_english_word_front/src/components/FullscreenCloseButton.vue` — 统一右上角关闭按钮、无障碍属性与样式。
- Create: `rob_english_word_front/src/composables/useEscapeClose.ts` — 注册和清理 Escape 监听。
- Create: `rob_english_word_front/src/components/FullscreenCloseButton.test.ts` — 关闭按钮行为测试。
- Create: `rob_english_word_front/src/composables/useEscapeClose.test.ts` — Escape 注册、调用和清理测试。
- Modify: `rob_english_word_front/package.json` — 增加测试脚本与测试依赖。
- Modify: `rob_english_word_front/vite.config.ts` — 配置 jsdom 测试环境。
- Modify: `rob_english_word_front/src/views/HomeView.vue` — 单人训练和难度全屏层逐层关闭。
- Modify: `rob_english_word_front/src/views/{TrainingAnswerResultsView,MasteredWordsView,WrongWordsView,RecordView}.vue` — 路由页面显式关闭。
- Modify: `rob_english_word_front/src/components/AnswerDetailModal.vue` — 详情弹窗统一 `×` 与 Escape。
- Modify: `rob_english_word_front/src/views/GameView.vue` — 结果层改为右上角关闭。

### Task 1: 建立共享关闭按钮的测试基线

**Files:**
- Modify: `rob_english_word_front/package.json`
- Modify: `rob_english_word_front/vite.config.ts`
- Create: `rob_english_word_front/src/components/FullscreenCloseButton.test.ts`
- Create: `rob_english_word_front/src/components/FullscreenCloseButton.vue`

- [ ] **Step 1: 配置 Vitest 测试依赖和命令**

在 `package.json` 增加 `"test": "vitest run"`，并在 devDependencies 增加 `vitest`、`@vue/test-utils`、`jsdom`。将 `vite.config.ts` 的 `defineConfig` 改为从 `vitest/config` 导入，并增加：

```ts
test: {
  environment: 'jsdom'
}
```

- [ ] **Step 2: 写关闭按钮失败测试**

```ts
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import FullscreenCloseButton from './FullscreenCloseButton.vue'

describe('FullscreenCloseButton', () => {
  it('renders an accessible close button and emits close', async () => {
    const wrapper = mount(FullscreenCloseButton)
    const button = wrapper.get('button')
    expect(button.attributes('type')).toBe('button')
    expect(button.attributes('aria-label')).toBe('关闭')
    expect(button.text()).toBe('×')
    await button.trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('does not emit close while disabled', async () => {
    const wrapper = mount(FullscreenCloseButton, { props: { disabled: true } })
    await wrapper.get('button').trigger('click')
    expect(wrapper.emitted('close')).toBeUndefined()
  })
})
```

- [ ] **Step 3: 安装依赖并运行测试，确认因组件不存在而失败**

Run: `cd rob_english_word_front && npm install && npm test -- src/components/FullscreenCloseButton.test.ts`

Expected: FAIL，提示无法解析 `FullscreenCloseButton.vue`。

- [ ] **Step 4: 实现最小关闭按钮组件**

组件 props 为 `disabled?: boolean`、`label?: string`，点击发出 `close`；模板固定输出 `×`，使用 `position: fixed; top/right`、圆形弱边框、hover、focus-visible、disabled 和窄屏样式，z-index 高于全屏内容。

- [ ] **Step 5: 运行按钮测试并确认通过**

Run: `cd rob_english_word_front && npm test -- src/components/FullscreenCloseButton.test.ts`

Expected: 2 tests PASS。

### Task 2: 建立 Escape 关闭组合函数

**Files:**
- Create: `rob_english_word_front/src/composables/useEscapeClose.test.ts`
- Create: `rob_english_word_front/src/composables/useEscapeClose.ts`

- [ ] **Step 1: 写 Escape 行为失败测试**

测试挂载一个调用 `useEscapeClose(close)` 的最小组件，派发 `KeyboardEvent('keydown', { key: 'Escape' })` 后断言 close 被调用一次；派发 Enter 不调用；卸载组件后再次派发 Escape 不调用。

- [ ] **Step 2: 运行测试并确认因组合函数不存在而失败**

Run: `cd rob_english_word_front && npm test -- src/composables/useEscapeClose.test.ts`

Expected: FAIL，提示无法解析 `useEscapeClose`。

- [ ] **Step 3: 实现 Escape 生命周期**

```ts
import { onBeforeUnmount, onMounted } from 'vue'

export function useEscapeClose(close: () => void) {
  const handleKeydown = (event: KeyboardEvent) => {
    if (event.key === 'Escape') close()
  }
  onMounted(() => window.addEventListener('keydown', handleKeydown))
  onBeforeUnmount(() => window.removeEventListener('keydown', handleKeydown))
}
```

- [ ] **Step 4: 运行组合函数与按钮测试**

Run: `cd rob_english_word_front && npm test`

Expected: 全部测试 PASS。

### Task 3: 改造 Home 内部全屏层

**Files:**
- Modify: `rob_english_word_front/src/views/HomeView.vue`

- [ ] **Step 1: 写源码契约失败测试**

新增 `src/views/fullscreenNavigation.contract.test.ts`，通过 `import homeSource from './HomeView.vue?raw'` 读取源码，断言不含 `training-back-btn`、`difficulty-back-btn` 和子界面的 `training-setup-user`，并包含两处 `FullscreenCloseButton` 及关闭函数名 `closeTrainingSetup`、`closeDifficultyPicker`。

- [ ] **Step 2: 运行契约测试并确认失败**

Run: `cd rob_english_word_front && npm test -- src/views/fullscreenNavigation.contract.test.ts`

Expected: FAIL，当前源码仍包含旧返回按钮。

- [ ] **Step 3: 实现逐层关闭**

导入共享组件和 `useEscapeClose`，增加：

```ts
function closeTrainingSetup() {
  if (isSoloPending.value) return
  showTrainingSetup.value = false
}

function closeDifficultyPicker() {
  showDifficultyPicker.value = false
}

useEscapeClose(() => {
  if (showDifficultyPicker.value) closeDifficultyPicker()
  else if (showTrainingSetup.value) closeTrainingSetup()
})
```

单人训练层右上角放置 disabled 绑定 `isSoloPending` 的关闭按钮，难度层右上角放置关闭按钮；删除子界面用户/退出区与旧返回按钮样式。

- [ ] **Step 4: 运行契约测试和构建**

Run: `cd rob_english_word_front && npm test && npm run build`

Expected: 测试 PASS，TypeScript/Vite build 成功。

### Task 4: 改造路由页和详情弹窗

**Files:**
- Modify: `rob_english_word_front/src/views/TrainingAnswerResultsView.vue`
- Modify: `rob_english_word_front/src/views/MasteredWordsView.vue`
- Modify: `rob_english_word_front/src/views/WrongWordsView.vue`
- Modify: `rob_english_word_front/src/views/RecordView.vue`
- Modify: `rob_english_word_front/src/components/AnswerDetailModal.vue`
- Modify: `rob_english_word_front/src/views/fullscreenNavigation.contract.test.ts`

- [ ] **Step 1: 扩展失败契约测试**

断言以上文件不含 `router.back()` 和页面导航文字 `返回`；四个路由页均使用 `FullscreenCloseButton` 和 `useEscapeClose`；训练结果/掌握页包含 `openTrainingSetup: true`，记录/错题页包含显式 `/home`；详情弹窗仍发出 `close`。

- [ ] **Step 2: 运行测试并确认旧页面失败**

Run: `cd rob_english_word_front && npm test -- src/views/fullscreenNavigation.contract.test.ts`

Expected: FAIL，显示旧返回按钮或 `router.back()` 残留。

- [ ] **Step 3: 实现显式路由关闭**

训练结果和掌握页的 `closePage` 执行：

```ts
router.push({ path: '/home', state: { openTrainingSetup: true } })
```

记录和错题页的 `closePage` 执行 `router.push('/home')`。每页渲染共享关闭按钮并调用 `useEscapeClose(closePage)`，根容器保持至少一屏；移除旧返回按钮 CSS。

- [ ] **Step 4: 统一详情弹窗关闭**

`AnswerDetailModal` 使用共享关闭按钮；记录和错题已有详情层也替换为共享组件或同一视觉类。Escape 优先关闭已打开的详情，未打开详情时才关闭列表页；遮罩点击仍只关闭详情。

- [ ] **Step 5: 运行全部测试与构建**

Run: `cd rob_english_word_front && npm test && npm run build`

Expected: 全部 PASS，且无 TypeScript unused import 错误。

### Task 5: 改造游戏结果并完成浏览器验收

**Files:**
- Modify: `rob_english_word_front/src/views/GameView.vue`
- Modify: `rob_english_word_front/src/views/fullscreenNavigation.contract.test.ts`

- [ ] **Step 1: 增加游戏结果失败测试**

断言 `GameView.vue` 不含 `返回首页` 和 `home-btn`，结果层包含 `FullscreenCloseButton`，并以 `gameOver` 作为显示条件。

- [ ] **Step 2: 运行测试并确认失败**

Run: `cd rob_english_word_front && npm test -- src/views/fullscreenNavigation.contract.test.ts`

Expected: FAIL，当前仍有底部返回首页按钮。

- [ ] **Step 3: 实现结果层关闭**

在 `result-overlay` 内以 `v-if="gameOver"` 渲染共享关闭按钮，点击调用 `goHome`；删除底部按钮和 `.home-btn` 样式。Escape 回调仅在 `gameOver` 为真时调用 `goHome`，对局进行中不响应。

- [ ] **Step 4: 运行完整自动验证**

Run: `cd rob_english_word_front && npm test && npm run build && rg -n "router\\.back|>\\s*返回\\s*<|返回首页|training-back-btn|difficulty-back-btn" src`

Expected: tests/build PASS；静态扫描无页面导航用途命中。

- [ ] **Step 5: 在 7002 做桌面和窄屏验收**

启动或重启前端后，在已登录状态依次打开单人训练、难度选择、训练结果、已掌握、错题、记录、各详情和游戏结果；逐一验证点击 `×` 与 Escape 的父级目标一致。将视口缩窄到移动端宽度，确认按钮固定可见、不遮挡标题/刷新/筛选内容，且每层只有一个有效关闭入口。
