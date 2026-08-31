---
title: 英语抢词 WebSocket 匹配、对战与结算链路
summary: 说明 Vue 用户主站通过 Netty WebSocket 完成匹配、房间、答题、重连和持久化结算的主流程。
---

# 英语抢词 WebSocket 匹配、对战与结算链路

## 范围与状态

当前匹配主路径已从 REST 迁移到 WebSocket。`/api/match/start` 和 `/cancel` 只返回兼容提示，不代表实际匹配成功。

## 主路径

```text
HomeView / GameView
  -> Pinia WebSocket Store 建立 /ws?token=...
  -> match_start 消息
  -> Netty GameChannelHandler
  -> 匹配与房间 Service
  -> Redis 临时状态 / ChannelManager 推送
  -> 出题与 answer 消息
  -> AnswerService / GameService 状态推进
  -> GameSettlementService 结算
  -> PostgreSQL 游戏、答题和学习记录
  -> WebSocket 推送最终结果
```

## 连接行为

- Home 与 Game 页面共用一个全局 WebSocket，避免重复连接重置服务端状态。
- 客户端定时发送 `ping`，收到 `pong` 更新心跳期限；超时后关闭并尝试重连。
- `duplicate_login` 会停止自动重连并关闭当前连接。
- 页面通过 handler key 注册消息监听，离开页面时应正确注销。

## 状态边界

- Redis 保存匹配、房间或短期状态；PostgreSQL 保存最终游戏和训练事实。
- WebSocket 已推送但数据库结算失败，或数据库成功但推送中断，都必须作为独立故障排查。
- 机器人答题可走服务端内部流程，不一定经过 WebSocket。

## 证据路径

- `rob_english_word_front/src/stores/websocket.ts`
- `rob_english_word_front/src/views/HomeView.vue`
- `rob_english_word_front/src/views/GameView.vue`
- `rob_english_word_back/src/main/java/com/robword/netty/`
- `rob_english_word_back/src/main/java/com/robword/service/GameService.java`
- `rob_english_word_back/src/main/java/com/robword/service/AnswerService.java`
- `rob_english_word_back/src/main/java/com/robword/service/GameSettlementService.java`

## 运行时待核对

- 匹配并发、重复消息、断线恢复和结算幂等性。
- 代理的 WebSocket Upgrade、空闲超时和粘性会话要求。
