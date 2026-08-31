# 三词挖空题 TTS 与 MinIO 设计

## 目标

当 word-agent 每积累 3 个错词并生成一条挖空练习句子时，同步为完整英文句子生成小米 MiMo TTS 音频、上传 MinIO，并把可播放地址保存到 `sentence_cloze_item.sentence_audio_url`。只有句子、音频和 MinIO 对象全部成功后才保存题目。

## 当前链路

1. `word-agent` 的 `WrongWordStrategyService` 每次按用户取 3 个 `pending` 错词。
2. 它调用 Java 后端 `/api/external/sentence-cloze/generate`。
3. Java 后端调用 word-agent `/v1/sentences/generate` 生成句子，负责挖空并写入 `sentence_cloze_item`。
4. 当前造句响应不含音频；`sentence_audio_url` 没有写入。
5. 当前独立 TTS 接口只保存本地文件，不上传 MinIO。

## 方案

由 word-agent 的 `/v1/sentences/generate` 完成一次原子式媒体生成：

1. 调用现有 LLM 生成完整英文句子、中文翻译和解释。
2. 使用现有 `MiMoTTSService` 为完整英文句子生成 WAV 临时文件。
3. 通过独立的 MinIO 存储组件上传到配置的 bucket，object key 使用 `sentence_cloze_tts/` 前缀和唯一文件名。
4. 上传并校验成功后删除本地临时文件。
5. 响应增加音频 URL、bucket、object key、文件大小、内容类型、TTS provider/model/voice/format。
6. Java 后端确认响应包含有效音频 URL 后才构建挖空句并插入 `sentence_cloze_item`。

选择该方案是因为句子和其音频由同一服务生成，错误边界清晰；Java 只负责业务校验和持久化，不需要编排第二次 TTS 请求。

## 数据流与失败语义

```text
3 个 pending 错词
  -> Java 挖空题生成接口
  -> word-agent 生成完整句子
  -> MiMo TTS 生成临时 WAV
  -> 上传并校验 MinIO
  -> word-agent 返回句子与音频元数据
  -> Java 挖空并写入 sentence_cloze_item
  -> 错词事件标记 processed
```

- LLM、TTS、MinIO 或响应校验任一步失败，Java 不插入题目。
- `WrongWordStrategyService` 捕获失败后把本批 3 条事件恢复为 `pending`，保留错误原因，后续重试。
- MinIO 上传失败时删除本地临时文件。
- MinIO 上传成功、但 Java 持久化失败时，可能留下无数据库引用的对象；本次不引入跨服务分布式事务或对象回删接口。对象 key 唯一，重试不会覆盖旧对象。后续可通过无引用对象巡检清理。
- 不使用浏览器 SpeechSynthesis 回退；新生成题目必须有 MinIO 音频。

## 配置与存储

- 复用 word-agent `.env` 中现有 MinIO endpoint、access key、secret key、SSL 和 bucket 配置。
- bucket：`ai-file-navigation`（以实际配置为准）。
- object key 前缀：`sentence_cloze_tts/`。
- 数据库保存经现有前端/代理可访问的相对 URL：`/{bucket}/{object_key}`。
- 本地 WAV 仅作为上传临时文件，成功或失败后都应清理。

## 代码范围

### `word_select_dashboard/word-agent`

- 为 Settings 补齐 MinIO 配置字段。
- 新增边界清晰的 MinIO 上传组件。
- 扩展造句响应 schema，返回音频元数据。
- 在造句路由中编排 LLM、TTS、MinIO 和临时文件清理。
- 保留独立 `/v1/tts/generate` 接口的现有行为，避免扩大兼容性改动。

### `rob_english_word_back`

- 扩展 word-agent 响应映射。
- 校验 `sentenceAudioUrl` 非空。
- 写入 `SentenceClozeItem.sentenceAudioUrl` 并在生成响应中返回。

### 无需修改

- `sentence_cloze_item` 已有 `sentence_audio_url`，不新增数据库列。
- `rob_english_word_cloze_web` 已支持优先播放 `sentenceAudioUrl` 指向的 MinIO 音频。
- Go server 和后台管理页面不在本次范围内。

## 测试

Python 测试覆盖：

- 成功生成句子、TTS 并上传 MinIO，响应字段正确。
- TTS 失败时不调用 MinIO。
- MinIO 失败时接口失败且临时文件被删除。
- 成功上传后临时文件被删除。
- MinIO object key 使用指定前缀且唯一。

Java 测试覆盖：

- 有有效音频 URL 时保存到 `sentence_cloze_item` 并返回。
- 音频 URL 缺失时拒绝插入题目。
- word-agent 请求失败时不插入题目。

最后运行 word-agent 测试、Java 目标测试与相关构建检查。
