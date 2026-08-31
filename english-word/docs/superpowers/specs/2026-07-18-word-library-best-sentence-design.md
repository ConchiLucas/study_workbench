# 词库列表最佳例句关联设计

## 目标

“词库单词管理”列表不再展示 `word.sentence`，改为展示 `word_clean_best_sentence.sentence`。关联不经过外键，直接使用区分大小写的英文单词匹配。

## 数据约束

- `word_clean` 是单词及释义的正确来源，`word_clean_best_sentence` 必须与它保持一致。
- 关联键使用原始 `word` 值，保留大小写语义差异。
- 在 `word_clean.word` 和 `word_clean_best_sentence.word` 上新增区分大小写的唯一索引，保证两张去重表的精确关联键唯一。
- 保留现有 `word_clean_best_sentence(word_clean_id)` 唯一索引和外键。
- 不在原始 `word` 表上建立全局单词唯一索引。同一单词可以合法存在于多个词库，实时数据也存在大量此类重复。

索引定义：

```sql
CREATE UNIQUE INDEX IF NOT EXISTS uk_word_clean_best_sentence_word
    ON public.word_clean_best_sentence (word);

CREATE UNIQUE INDEX IF NOT EXISTS uk_word_clean_word
    ON public.word_clean (word);
```

## 查询变更

词库单词列表从 `word` 出发，直接左连接最佳例句表：

```sql
FROM word w
LEFT JOIN word_clean_best_sentence wcbs
  ON wcbs.word = w.word
```

接口响应字段名保持 `sentence` 和 `sentence_translation` 不变，因此前端无需修改：

```sql
COALESCE(wcbs.sentence, '') AS sentence,
COALESCE(wcbs.sentence_translation, '') AS sentence_translation
```

未命中最佳例句的记录返回空字符串，不回退到 `word.sentence`。这主要影响被 `word_clean` 排除的词组。

## 例句语音播放

- 列表接口同时返回最佳例句的 TTS 状态、bucket、object key 和 object URL。
- “词库单词管理”和“去重单词表”都在例句文字右侧显示播放/暂停图标，交互保持一致。
- 前端仅在 `tts_status=success` 且 object URL 非空时启用播放按钮。
- 同一时间只播放一条音频；切换单词、切换词库或分页时停止当前音频。
- 支持 MinIO 相对路径 `/ai-file-navigation/...` 和现有 TTS 文件服务绝对 URL。

关键词搜索中的“例句”口径同步切换为 `wcbs.sentence`。由于搜索条件需要最佳例句字段，总数查询和列表查询使用相同的左连接，确保分页总数准确。唯一索引保证连接不会放大行数。

## 代码与数据库变更

- 修改 `word_select_dashboard/server/api/v1/system/sys_word_library.go`：抽取/复用直接关联 SQL，更新总数、列表字段和关键词搜索。
- 修改 `rob_english_word_back/db/word_clean.sql` 和 `word_clean_best_sentence.sql`：加入区分大小写的单词唯一索引。
- 修改运行时建表逻辑中的索引初始化，保证已部署环境启动接口时也具备该约束。
- 在实时 `rob_english_word` 数据库执行同一个幂等索引语句，并验证索引存在。

## 测试与验收

- 先增加 Go 单元测试，验证关联使用区分大小写的直接等号、例句来自最佳例句表、关键词搜索使用最佳例句字段。
- 运行目标测试，确认修改前按预期失败，修改后通过。
- 运行 Go 相关测试与 React 构建，确认响应结构兼容现有页面。
- 数据库验收：索引存在、最佳例句单词原值无重复、连接后词库列表总数不增加。
- 抽样对比列表响应中的例句与 `word_clean_best_sentence.sentence` 完全一致。

## 关联键与数据核验结果

实时数据库核验表明：

- 修改前，`word_clean` 与 `word_clean_best_sentence` 的 22,415 个单词逐字完全一致。
- 原始 `word` 表存在 295 行大小写差异。语义核验发现 `May/may`、`March/march`、`China/china`、`Turkey/turkey`、`US/us` 等大小写不同的词具有不同含义，使用 `lower(word)` 会关联到错误例句。
- 已将 1 条 `april` 修正为 `April`，并修正 44 条释义完全一致、120 条人工核验同义且无同词库冲突的数据。
- 仍保留 130 条语义不同或存在同词库冲突的数据，直接等号关联会让这些记录保持未匹配，不会误用例句。
- 已从原始 `word` 和 `word_clean` 中删除释义口径不适合共用最佳例句的 `Metro`、`right`、`thanksgiving`；对应候选句和旧任务已级联清理，不再重新造句。
- 使用 `btrim(word)` 对当前数据覆盖率没有提升。

最终 `word_clean` 与 `word_clean_best_sentence` 均为 22,375 条且一一对应。因此使用区分大小写的直接等号关联。数据问题通过受控修数解决，不在查询层模糊匹配。
