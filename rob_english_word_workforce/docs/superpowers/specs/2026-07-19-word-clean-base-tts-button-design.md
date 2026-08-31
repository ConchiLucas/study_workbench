# 去重单词表基础单词 TTS 播放按钮设计

## 目标

在 `word_select_dashboard` 的“去重单词表”中，为“单词”列增加基础单词发音按钮。按钮读取 `public.word_clean_tts`，不复用当前“例句”列的 `word_clean_best_sentence` 音频。

## 数据现状

- `public.word_clean` 共 22,098 条。
- `public.word_clean_tts` 共 22,098 条，与 `word_clean_id` 一一对应。
- 21,888 条为 `success` 且有 `tts_object_url`，210 条为 `pending`。
- 当前成功记录的 `tts_bucket` 为空，`tts_object_key` 是 `word_clean_*_tts_*.wav` 文件名，URL 指向 `http://127.0.0.1:19186/api/tts/files/...`。
- MinIO `ai-file-navigation/word_clean_tts/` 现有对象名为 `word_clean_*_best_*.wav`，属于最佳例句语音；与 21,888 条基础单词 TTS 文件名匹配数为 0。因此当前基础单词 TTS 不能声明为已迁移到 MinIO。

本功能只消费数据库已保存的可播放 URL，不负责任务中心生成、重试或 MinIO 迁移。任务中心以后更新同一行的状态与 URL 后，列表刷新即可使用新地址。

## 接口设计

Go 列表接口在现有查询中增加：

```sql
LEFT JOIN word_clean_tts wct ON wct.word_clean_id = wc.id
```

每个 `WordCleanItem` 增加以下字段：

- `wordTtsStatus`
- `wordTtsBucket`
- `wordTtsObjectKey`
- `wordTtsObjectUrl`

关联严格使用 `word_clean_id`，不通过单词文本匹配。

## 前端交互

- 在“单词”文本右侧、现有“查看大模型造句结果”按钮旁增加发音按钮。
- 仅当 `wordTtsStatus === "success"` 且 `wordTtsObjectUrl` 非空时允许播放。
- 210 条 `pending` 不增加额外状态文案或业务处理，只保持按钮不可播放。
- “单词”按钮播放 `word_clean_tts` 的基础单词音频；“例句”列按钮继续播放最佳例句音频。
- 页面使用一个共享音频实例，并以 `word:<id>`、`sentence:<id>` 区分当前播放目标，保证开始新音频时停止上一段音频，且两个按钮不会同时显示播放状态。
- 列表筛选、分页和刷新时停止当前音频。

## 错误处理

- URL 缺失或状态非成功时不创建 `Audio`。
- 浏览器加载或播放失败时停止播放器并提示“无法播放 `<word>` 的单词语音”。
- 不使用浏览器 SpeechSynthesis 回退。

## 测试与验证

- Go 查询契约测试：保护 `word_clean_tts` 的 `word_clean_id` 左连接和四个返回别名。
- TypeScript 音频 URL 测试：成功且 URL 非空才返回可播放地址。
- 前端构建验证类型字段和播放逻辑。
- 真实列表接口验证成功记录返回基础单词 URL。
- 浏览器验证单词按钮播放/停止、切换到例句音频时停止前一段，以及 `pending` 行不触发播放。

