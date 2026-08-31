---
title: 英语完形填空 React 前端
summary: 提供登录、难度选择、完形任务、答题反馈、待练与复习列表、历史统计及音频播放体验。
---

# 英语完形填空 React 前端

## 职责与边界

- 面向普通用户提供独立的完形填空练习与复习体验。
- 复用 Java 后端的登录注册和 `/api/cloze-practice/**` 能力。
- 不参与实时抢词 WebSocket；运营数据查看由管理后台负责。

## 项目定位

| 项目 | 值 |
| --- | --- |
| 相对路径 | `rob_english_word_cloze_web` |
| 技术栈 | React 19、TypeScript、Vite、Vitest |
| 应用入口 | `src/main.tsx`、`src/App.tsx` |
| API 封装 | `src/lib/api.ts` |
| 领域类型 | `src/types/cloze.ts` |

## 页面状态与业务入口

- 当前是以 `App.tsx` 状态切换为主的单页应用，没有独立 Router 路由表。
- 支持登录、注册、单人难度选择、待练任务、到期复习、已答列表和分轮结果。
- 支持获取下一题、按难度批量获取、提交答案、查看统计与历史。
- 音频能力由 `sentenceAudio.ts`、`wordAudio.ts` 和 `WordAudioButton.tsx` 承担。

## API 与外部依赖

- 身份：`/api/auth/login`、`/api/auth/register`。
- 任务：`/api/cloze-practice/tasks/next|pending|review-due|answered|difficulty-batch`。
- 偏好与结果：`preferences/**`、`answers`、`stats`、`history`。
- 单词音标存在对公共词典 HTTP API 的直接请求；网络失败时不得阻断核心答题流程。

## 构建与验证

- 开发：`npm run dev`。
- 构建：`npm run build`，执行 TypeScript 检查与 Vite 构建。
- 测试：`npm test`，覆盖难度、练习模式、音频和全屏关闭导航。

## 关键代码锚点

- 主流程：`src/App.tsx`
- 练习入口组件：`src/components/PracticeLaunchers.tsx`
- API：`src/lib/api.ts`
- 难度与模式：`src/lib/soloDifficulty.ts`、`practiceMode.ts`
- 测试：`test/`

## 关联链路

- [登录注册与鉴权](../../chains/login-authentication.md)
- [完形练习与复习](../../chains/cloze-practice-review.md)

## 运行时待核对

- 同源 `/api` 在开发、独立 Nginx 和集中部署下的代理目标。
- 公共词典服务不可用、限流或跨域失败时的降级表现。

## 复核信息

- 复核日期：2026-08-01；源码基线：`dev` 分支。
- 证据：`package.json`、`App.tsx`、API、组件、领域类型、Vite 和测试。
