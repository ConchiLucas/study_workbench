# 基础单词 TTS 全量迁移 MinIO 设计

## 目标

将 `public.word_clean_tts` 中 21,888 条 `success` 基础单词语音从任务中心本地目录迁移到 MinIO，更新数据库为 MinIO 代理地址；只有在数据库、MinIO 和播放校验全部通过后，才删除这些数据库记录对应的本地 WAV。

本次不处理 210 条 `pending` 记录。任务中心后续负责生成并回填它们。

## 已确认的源数据

- 本地目录：`/Users/conchi/workforce/python_workforce/ai-task-center/data/tts_audio`
- 数据库成功记录：21,888 条
- 本地 WAV：21,888 个
- 数据库缺失文件：0
- 本地多余文件：0
- 文件大小不一致：0
- 非 RIFF/WAVE 文件：0
- 总大小：1,594,209,792 字节

当前数据库成功记录：

- `tts_bucket` 为空；
- `tts_object_key` 为 `word_clean_*_tts_*.wav` 文件名；
- `tts_object_url` 指向 `http://127.0.0.1:19186/api/tts/files/...`。

## MinIO 目标

- endpoint：使用项目现有 MinIO 配置；
- bucket：`ai-file-navigation`；
- object key：`word_clean_tts/<原文件名>`；
- 数据库 URL：`/ai-file-navigation/word_clean_tts/<原文件名>`；
- Content-Type：`audio/wav`。

现有最佳例句对象位于同一前缀，但文件名为 `word_clean_*_best_*.wav`；基础单词文件名为 `word_clean_*_tts_*.wav`，两者不冲突。

## 迁移工具

新增基础单词专用迁移脚本，支持：

- `--dry-run`：仅检查并汇总，不上传、不更新数据库、不删除；
- `--limit`：小批量验证；
- `--workers`：并发上传；
- `--batch-size`：数据库分批提交；
- `--delete-local`：仅在全量校验通过后删除已迁移源文件。

脚本必须可断点续跑：

- 已存在且 MD5、大小一致的 MinIO 对象直接复用；
- 未迁移记录继续上传并更新；
- 已迁移记录不重复更新；
- 同一 object key 内容不一致时重新上传并复核。

## 单条迁移流程

1. 从 `word_clean_tts` 读取 `success` 记录。
2. 根据 `tts_object_key` 的文件名解析本地源文件。
3. 验证文件存在、大小与数据库一致、文件头为 RIFF/WAVE。
4. 计算本地 MD5。
5. 上传或复用 MinIO 对象。
6. 使用 `stat_object` 校验远端大小和 ETag。
7. 使用带旧 URL、状态和大小条件的 `UPDATE` 更新数据库：
   - `tts_bucket`
   - `tts_object_key`
   - `tts_object_url`
   - `updated_at`
8. 任一步失败时保留本地文件，并停止或报告失败，不把该条记录切换到 MinIO。

## 全量校验门槛

删除本地文件前必须同时满足：

- 21,888 条成功记录全部具有目标 bucket、完整 object key 和 MinIO 相对 URL；
- 数据库成功记录的 object key 在 MinIO 中全部存在；
- MinIO 对象大小与数据库 `file_size` 全部一致；
- 本地文件大小与 MinIO 对象大小全部一致；
- 缺失对象数为 0；
- 字段不符数为 0；
- 大小不符数为 0；
- 从不同 ID 区间抽取至少 20 条，经现有 Go/前端代理返回 HTTP 200、`audio/wav` 且文件头为 RIFF/WAVE。

## 本地删除保护

- 只删除本轮全量校验集合中的 21,888 个明确路径；
- 不使用目录级递归删除，不删除目录；
- 不使用未解析的通配符作为删除目标；
- 删除前再次确认每条数据库 URL 已不再指向 `19186`；
- 若任一校验失败，则本轮删除数必须为 0；
- 删除后再次确认本地匹配文件为 0，并抽样至少 5 条通过 MinIO URL 播放。

## 与播放按钮功能的关系

迁移完成后再实现“去重单词表”的基础单词播放按钮。列表接口读取更新后的 `word_clean_tts.tts_object_url`，前端通过现有 `/ai-file-navigation` 代理播放，不依赖任务中心 `19186` 文件服务。

