---
title: 英语单词游戏 Vue 用户前端
summary: 提供登录注册、抢词对战、训练记录、错词和掌握词等用户侧页面，并通过 REST 与 WebSocket 连接 Java 后端。
---

# 英语单词游戏 Vue 用户前端

## 职责与边界

- 承载普通用户登录、注册、首页、实时抢词游戏和学习记录页面。
- 展示错词、已掌握单词、训练结果和答题详情。
- REST 请求和 WebSocket 都面向 Java 核心后端；完形专项体验由独立 React 项目承担。

## 项目定位

| 项目 | 值 |
| --- | --- |
| 相对路径 | `rob_english_word_front` |
| 技术栈 | Vue 3、Vue Router、Pinia、Axios、TypeScript、Vite |
| 应用入口 | `src/main.ts` |
| 路由 | `src/router/index.ts` |
| 请求封装 | `src/api/index.ts` |
| WebSocket 状态 | `src/stores/websocket.ts` |

## 页面与路由

- `/login`、`/register`：身份入口。
- `/home`：登录后的用户主页和匹配入口。
- `/game`：实时对战页面。
- `/records`：游戏或训练记录。
- `/wrong-words`、`/mastered-words`：错词和掌握词列表。
- `/training-results`：训练答题结果。
- 除登录注册外的页面由路由守卫检查 Pinia 鉴权状态。

## 请求与实时通信

- Axios 使用同源地址，自动附加 Bearer Token；遇到 401 清理本地 Token 并跳转登录。
- WebSocket 使用同源 `/ws?token=...`，由全局 Pinia Store 保证 Home 与 Game 共用单连接。
- Store 实现心跳、超时关闭、自动重连、重复登录处理和页面级消息订阅。
- 匹配和取消匹配走 WebSocket 消息，不应回退到兼容 REST 接口作为主路径。

## 构建与验证

- 开发：`npm run dev`。
- 构建：`npm run build`，包含 `vue-tsc` 与 Vite。
- 测试：`npm test`，使用 Vitest。
- 容器入口位于项目 `docker-compose.yml`，工作空间还提供集中部署目录。

## 关键代码锚点

- 页面：`src/views/`
- 鉴权状态：`src/stores/auth.ts`
- WebSocket：`src/stores/websocket.ts`
- API：`src/api/index.ts`
- 请求代理：`vite.config.ts` 和容器 Nginx 配置

## 关联链路

- [登录注册与鉴权](../../chains/login-authentication.md)
- [实时抢词对战](../../chains/realtime-match-game.md)
- [错词与掌握进度](../../chains/wrong-word-mastery.md)

## 复核信息

- 复核日期：2026-08-01；源码基线：`dev` 分支。
- 证据：`package.json`、Router、Store、View、API、Vite 和测试文件。
