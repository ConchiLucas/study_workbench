# Study Content Admin · 拼音素材（MVP）

## Goal

在内容后台增加「拼音」菜单，对齐识字的同步列表 + TTS 缓存模式。

## Scope (MVP)

| 做 | 不做 |
|---|---|
| 顶栏「拼音」`/pinyin` | 字图 / 义图 |
| 从题库同步 45 个音（声母/韵母） | 在后台编辑 Solo/Word |
| 分组列表展示字母 + Solo/Word 例字 | 改 kid-app 消费方式 |
| Solo / Word 读音 → MinIO，按组批量 | |

## Data

表 `pinyin_assets`：`kp_id`, `letter`, `module_code/name/order`, `kp_order`, `solo_text`, `word_text`, `solo_speech_url`, `word_speech_url`, timestamps。

例字表与 workbench `quiz/pinyin.go` 的 `pinyinTable` 保持一致（MVP 内嵌副本；编辑能力后续再做）。

## Storage

- `pinyin/speech/{kpId}/solo.mp3`
- `pinyin/speech/{kpId}/word.mp3`

## APIs

- `POST /api/v1/pinyin/sync`
- `GET /api/v1/pinyin/items?view=groups|table`
- `GET /api/v1/pinyin/items/:kpId/speech/:kind.mp3`（`kind=solo|word`，懒生成）
- `POST /api/v1/pinyin/items/:kpId/speech/:kind`（强制重生成）
- `POST /api/v1/pinyin/speech/batch?moduleCode=shengmu|yunmu`（只补缺失）

## TTS

朗读例字汉字（非字母本身）。Provider 走配置中心默认 TTS（与识字默认一致）。`eng` 无 Solo，跳过。
