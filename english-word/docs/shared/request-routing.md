---
title: 英语单词请求与实时通信寻址
summary: 说明三个前端如何通过同源 REST、WebSocket、开发代理或静态托管连接 Java 与 Go 后端。
---

# 英语单词请求与实时通信寻址

## 用户主站

- Vue Axios 的 `baseURL` 为空，浏览器以同源 `/api/**` 请求 Java 后端。
- Bearer Token 由请求拦截器从本地存储读取。
- 实时连接使用与当前页面同源的 `/ws?token=...`，协议按 HTTP/HTTPS 自动选择 WS/WSS。
- Java REST 与 Netty WebSocket 是两个运行入口，部署层必须分别正确转发。

## 完形前端

- React 完形前端以同源 `/api/auth/**` 和 `/api/cloze-practice/**` 请求 Java 后端。
- 开发环境、独立 Nginx 和集中部署的代理目标可能不同，修改时同时核对 Vite、Nginx 与 Compose。
- 公共词典 API 是浏览器直连外部服务，不经过 Java 后端。

## 运营管理端

- React 管理端以同源 `/api/**` 请求 Go 服务。
- Go 服务可以在部分运行模式下托管构建后的静态资源，也可以与独立前端容器组合；以所用部署目录为准。
- Go 再以服务端 HTTP 调用 Python Agent，浏览器不直接访问 Agent。

## 排查顺序

1. 浏览器 Network 中确认最终 URL、方法、状态码和响应体。
2. 检查 Vite 或 Nginx 是否改写 `/api`、`/ws`。
3. 检查对应 Java/Go Controller 或 Router 是否存在。
4. WebSocket 额外检查 Upgrade 头、Token、心跳和连接关闭码。
5. 只有源码无法确认的路由才进入运行时代理或容器网络排查。
