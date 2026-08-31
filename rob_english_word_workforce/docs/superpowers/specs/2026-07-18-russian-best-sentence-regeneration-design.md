# Russian 最佳例句与 TTS 重生成设计

## 目标

为 `word_clean.id = 17241`、`word = 'Russian'` 重新生成一条含义正确的最佳例句，并生成对应的小米 MiMo TTS 音频上传到 MinIO。更新后，最佳例句、中文翻译、挖空句、答案和 TTS 元数据必须彼此一致。

## 现状

- 错误的 `2ussian` 重复主记录及其错误最佳例句已经合并清理。
- 正确主记录为 `word_clean.id = 17241`。
- 当前已经存在一条正确的 `Russian` 最佳例句和成功 TTS；本次按用户要求重新生成并替换。
- `word_clean_best_sentence` 对 `word_clean_id` 和 `word` 都有唯一索引，因此更新现有记录，不新增第二条记录。

## 生成链路

1. 调用 word-agent 的 `POST /v1/sentences/generate`，参数为 `{"words":["Russian"]}`。
2. 该接口使用现有 LLM 生成英文例句、中文翻译和解释。
3. 接口调用小米 MiMo TTS 生成完整英文句子的 WAV 音频。
4. 接口将音频上传到现有 MinIO bucket，并返回 bucket、object key、object URL、文件大小和 Content-Type。
5. 生成结果必须满足：
   - 英文句子包含独立单词 `Russian`，大小写一致。
   - 中文翻译表达“俄罗斯的”或“俄语”的正确含义。
   - 音频 URL、bucket 和 object key 均非空。

## 数据更新

在单个 PostgreSQL 事务中更新 `word_clean_best_sentence` 的现有 `Russian` 记录：

- `sentence`：新英文例句。
- `sentence_translation`：新中文翻译。
- `cloze_sentence`：只把句子中的 `Russian` 替换为 `____`。
- `cloze_answer`：`Russian`。
- `tts_status`：`success`。
- `tts_provider`：`xiaomi-mimo`。
- `tts_model`、`tts_voice`、`tts_audio_format`：使用接口返回值。
- `tts_bucket`、`tts_object_key`、`tts_object_url`：使用 MinIO 返回值。
- `tts_content_type`、`tts_file_size`、`tts_generated_at`：使用生成结果和当前时间。
- `tts_error_message`：清空。
- `updated_at`：当前时间。

保留原记录的 `id`、`word_clean_id`、`word`、评分和来源字段，避免破坏现有外键关系。

## 失败处理

- 造句、TTS 或 MinIO 任一步失败时，不更新数据库，保留当前可用例句和音频。
- 生成内容未包含精确单词 `Russian` 时，不更新数据库。
- 数据库更新前再次校验目标仍为 `word_clean_id = 17241`、`word = 'Russian'`；校验失败则整体回滚。
- 新音频写入成功但数据库事务失败时，报告新 MinIO 对象为孤儿，不删除原音频。

## 验证

- 查询目标记录，核对英文句子、中文翻译、挖空句和答案。
- 对新 `tts_object_url` 发起读取请求，确认返回成功且内容非空。
- 全库复查 `2ussian` 残留仍为 0。
- 确认 `word_clean_best_sentence` 中 `Russian` 仍只有一条记录。
