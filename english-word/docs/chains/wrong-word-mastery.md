---
title: 英语错词、掌握词与学习进度链路
summary: 说明游戏和完形答题如何形成错词与复习进度，并在用户端、运营端和 Python Agent 间流转。
---

# 英语错词、掌握词与学习进度链路

## 主路径

```text
游戏或完形答题
  -> Java 答题与结算 Service
  -> 错词事件 / 用户词状态 / 复习进度
  -> rob_english_word
  -> Vue 错词与掌握词页面
  -> Go 管理服务跨库读取
  -> React 管理端用户学习数据页面
  -> 必要时 Python Agent 处理错词事件或句子任务
```

## 业务口径

- 错词表示当前需要复习的用户词状态，不等同于所有历史答错事件。
- 掌握词是业务状态结果，展示口径应与 Java 查询一致。
- 完形错词与普通游戏错词来源不同，但可能汇总到统一复习进度。
- 复习进度应按稳定用户、词和任务关联推进，重复事件不能反复创建无界记录。

## 读取入口

- 用户端：`/api/wrong-words/**`、`/api/mastered-words`。
- 管理端：Go `users/wrong-words`、`users/cloze-wrong-words`、`users/mastered-words` 等入口。
- Agent：`POST /v1/wrong-words/events` 及 `wrong_word_strategy.py`。

## 证据路径

- `rob_english_word_back/src/main/java/com/robword/controller/WrongWordController.java`
- `rob_english_word_back/src/main/java/com/robword/controller/MasteredWordController.java`
- `rob_english_word_back/src/main/java/com/robword/service/GameSettlementService.java`
- `rob_english_word_front/src/views/WrongWordsView.vue`
- `rob_english_word_front/src/views/MasteredWordsView.vue`
- `word_select_dashboard/server/router/system/sys_app_user.go`
- `word_select_dashboard/word-agent/src/word_agent/services/wrong_word_strategy.py`

## 运行时待核对

- 普通答题与完形答题同时影响同一词时的合并规则。
- “掌握”后的再次答错、复习重置和后台展示口径。
