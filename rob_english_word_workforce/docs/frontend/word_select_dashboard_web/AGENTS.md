---
title: 单词选择 React 运营管理端
summary: 提供执行记录、句子生成、词库清洗、用户学习数据、完形结果以及 AI/TTS 配置等运营页面。
---

# 单词选择 React 运营管理端

## 职责与边界

- 为管理员提供单词库、清洗单词、句子、执行任务和完形结果查看入口。
- 展示用户、错词、完形错词、掌握词和训练结果。
- 维护 AI、执行和 TTS 配置，并触发 Go 服务编排的后台任务。
- 不直接调用 Python Agent 或业务数据库；所有业务请求通过 Go 服务 API。

## 项目定位

| 项目 | 值 |
| --- | --- |
| 相对路径 | `word_select_dashboard/web-react` |
| 技术栈 | React 19、TypeScript、Ant Design、TanStack Query、Vite |
| 应用入口 | `src/main.tsx`、`src/App.tsx` |
| API 封装 | `src/lib/*Api.ts` |
| 领域类型 | `src/types/` |

## 页面模块

- 执行记录、句子生成历史、完形结果。
- AI 配置、执行配置、TTS 配置。
- 用户、用户错词、完形错词、已掌握词和训练结果。
- 词库浏览、清洗单词、清洗句子与评分结果。
- 页面导航目前由 `App.tsx` 的 `PageKey` 和内部状态切换，不是独立路由文件。

## 请求边界

- 前端以同源 `/api/**` 调用 Go 服务。
- `src/lib/auth.ts` 管理后台身份信息；各 `*Api.ts` 按业务域封装请求。
- 句子生成走 `/api/sentences/**`，词库和评分走 `/api/word-libraries/**`，执行与配置走各自 API。
- 音频 URL 与播放逻辑集中在 `src/utils/wordAudio.ts`，最终存储来源由服务端配置决定。

## 构建与验证

- 开发：`npm run dev`。
- 构建：`npm run build`，执行 TypeScript 检查与 Vite 构建。
- 当前 `package.json` 没有统一测试脚本；修改核心 API 或页面状态时应补充针对性测试或执行手工合同验证。

## 关键代码锚点

- 页面总装配：`src/App.tsx`
- 执行配置：`src/components/ExecutionConfigPage.tsx`
- 完形结果：`src/components/ClozeResultTable.tsx`
- API：`src/lib/`
- 配置领域：`src/features/aiConfig.ts`、`executionConfig.ts`、`ttsConfig.ts`

## 关联链路

- [AI 句子生成、评分与 TTS](../../chains/ai-sentence-tts-scoring.md)
- [完形练习与复习](../../chains/cloze-practice-review.md)
- [错词与掌握进度](../../chains/wrong-word-mastery.md)

## 复核信息

- 复核日期：2026-08-01；源码基线：`dev` 分支。
- 证据：`package.json`、`App.tsx`、Components、Features、API、Types 和 Vite 配置。
