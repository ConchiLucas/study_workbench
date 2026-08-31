---
title: 英语单词登录注册与 JWT 鉴权链路
summary: 说明 Vue 与完形前端如何调用 Java 鉴权接口、保存 Token、访问受保护 API 并处理失效状态。
---

# 英语单词登录注册与 JWT 鉴权链路

## 范围与状态

本链路覆盖两个用户前端到 Java 后端的注册、登录和 Bearer Token 校验。源码入口已静态核对；Token 生命周期和生产代理仍需按运行配置验证。

## 主路径

```text
Vue 用户主站或 React 完形前端
  -> POST /api/auth/register | /api/auth/login
  -> AuthController / AuthService
  -> 用户数据与密码校验
  -> 返回 JWT 和用户信息
  -> 前端本地保存 Token
  -> 后续 REST 请求附加 Authorization: Bearer <token>
  -> Spring Security 校验并建立用户上下文
```

## 前端行为

- Vue 通过 Pinia 维护登录状态，Router 阻止未登录用户进入受保护页面。
- Vue Axios 遇到 401 会清理 Token 并跳转 `/login`。
- 完形前端在 `App.tsx` 保存登录结果，并把 Token 传给 `src/lib/api.ts` 的后续请求。
- WebSocket 连接通过查询参数传递 Token，由 Netty 握手完成后的处理逻辑校验。

## 数据与安全边界

- 用户事实保存在 `rob_english_word`；密码摘要、JWT 密钥和真实 Token 不进入文档或日志。
- 浏览器本地 Token 的清理只影响当前客户端，不等同于服务端撤销。
- 重复登录、Token 过期和 WebSocket 断开需要分别处理。

## 证据路径

- `rob_english_word_front/src/router/index.ts`
- `rob_english_word_front/src/stores/auth.ts`
- `rob_english_word_front/src/api/index.ts`
- `rob_english_word_cloze_web/src/lib/api.ts`
- `rob_english_word_back/src/main/java/com/robword/controller/AuthController.java`
- `rob_english_word_back/src/main/java/com/robword/config/`
- `rob_english_word_back/src/main/java/com/robword/netty/GameChannelHandler.java`

## 运行时待核对

- JWT 过期、刷新或撤销策略，以及多个前端共享登录状态的预期。
- 反向代理是否会过滤 Authorization 或 WebSocket 查询参数。
