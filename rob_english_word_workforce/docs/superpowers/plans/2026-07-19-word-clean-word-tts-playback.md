# 去重单词表基础单词 TTS 播放 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 7001“去重单词表”的单词旁增加基础单词 TTS 播放按钮，通过现有 MinIO 代理播放，并与例句 TTS 共用单实例播放器。

**Architecture:** Go 列表查询通过 `word_clean.id = word_clean_tts.word_clean_id` 左连接，一次返回当前页基础单词 TTS 元数据。React 页面将现有例句播放器改为按 `word:<id>` / `sentence:<id>` 区分目标的共享播放器，基础单词和例句按钮调用同一播放入口。

**Tech Stack:** Go、Gin、GORM、PostgreSQL、React 19、TypeScript、Ant Design、Node.js 内置测试运行器、Vite、MinIO HTTP 代理。

---

### Task 1: 列表接口返回基础单词 TTS

**Files:**
- Modify: `word_select_dashboard/server/api/v1/system/sys_word_library.go:55-95,275-335`
- Modify: `word_select_dashboard/server/api/v1/system/sys_word_library_test.go`

- [ ] **Step 1: 写失败的 Go 查询契约测试**

在 `sys_word_library_test.go` 增加：

```go
func TestWordCleanListIncludesBaseWordTTS(t *testing.T) {
	joinSQL := wordCleanTTSJoinSQL()
	if !strings.Contains(joinSQL, "LEFT JOIN word_clean_tts wct ON wct.word_clean_id = wc.id") {
		t.Fatalf("expected word_clean_id TTS join, got %q", joinSQL)
	}
	for _, field := range []string{
		"COALESCE(wct.status, '') AS word_tts_status",
		"COALESCE(wct.tts_bucket, '') AS word_tts_bucket",
		"COALESCE(wct.tts_object_key, '') AS word_tts_object_key",
		"COALESCE(wct.tts_object_url, '') AS word_tts_object_url",
	} {
		if !strings.Contains(wordCleanTTSSelectSQL(), field) {
			t.Fatalf("missing base word TTS field %q", field)
		}
	}
}
```

- [ ] **Step 2: 运行测试并确认因缺少查询函数而失败**

Run: `go test ./api/v1/system -run TestWordCleanListIncludesBaseWordTTS -count=1`

Expected: FAIL，提示 `wordCleanTTSJoinSQL` 或 `wordCleanTTSSelectSQL` 未定义。

- [ ] **Step 3: 扩展响应结构与 SQL 片段**

在 `WordCleanItem` 增加：

```go
WordTTSStatus    string `json:"wordTtsStatus"`
WordTTSBucket    string `json:"wordTtsBucket"`
WordTTSObjectKey string `json:"wordTtsObjectKey"`
WordTTSObjectURL string `json:"wordTtsObjectUrl"`
```

新增查询片段函数：

```go
func wordCleanTTSSelectSQL() string {
	return `,
	       COALESCE(wct.status, '') AS word_tts_status,
	       COALESCE(wct.tts_bucket, '') AS word_tts_bucket,
	       COALESCE(wct.tts_object_key, '') AS word_tts_object_key,
	       COALESCE(wct.tts_object_url, '') AS word_tts_object_url`
}

func wordCleanTTSJoinSQL() string {
	return " LEFT JOIN word_clean_tts wct ON wct.word_clean_id = wc.id"
}
```

将 `CleanWords` 的查询拼接为最佳例句字段后追加 `wordCleanTTSSelectSQL()`，并在 `word_clean_best_sentence` 连接后追加 `wordCleanTTSJoinSQL()`。

- [ ] **Step 4: 运行 Go 测试**

Run: `go test ./api/v1/system -count=1`

Expected: PASS。

- [ ] **Step 5: 提交后端变更**

```bash
git add word_select_dashboard/server/api/v1/system/sys_word_library.go word_select_dashboard/server/api/v1/system/sys_word_library_test.go
git commit -m "feat: expose base word tts in clean word list"
```

### Task 2: 统一可播放 URL 规则

**Files:**
- Modify: `word_select_dashboard/web-react/src/utils/wordAudio.ts`
- Modify: `word_select_dashboard/web-react/test/wordAudio.test.ts`

- [ ] **Step 1: 先将测试改为通用 TTS URL 解析函数**

测试导入 `playableTTSAudioURL`，覆盖 `success + MinIO URL`、失败状态和空 URL：

```ts
import { playableTTSAudioURL } from "../src/utils/wordAudio.ts";

test("returns the trimmed MinIO URL for a successful TTS record", () => {
  assert.equal(
    playableTTSAudioURL("success", "  /ai-file-navigation/word_clean_tts/abase.wav  "),
    "/ai-file-navigation/word_clean_tts/abase.wav",
  );
});
```

- [ ] **Step 2: 运行测试并确认导出缺失**

Run: `node --experimental-strip-types --test test/wordAudio.test.ts`

Expected: FAIL，提示模块没有导出 `playableTTSAudioURL`。

- [ ] **Step 3: 实现通用解析函数并保留兼容导出**

```ts
export function playableTTSAudioURL(status: string, objectURL: string): string | null {
  const audioURL = objectURL.trim();
  return status === "success" && audioURL ? audioURL : null;
}

export const playableBestSentenceAudioURL = playableTTSAudioURL;
```

- [ ] **Step 4: 运行纯函数测试**

Run: `node --experimental-strip-types --test test/wordAudio.test.ts`

Expected: 3 tests PASS。

### Task 3: React 页面共用一个播放器

**Files:**
- Modify: `word_select_dashboard/web-react/src/types/wordLibrary.ts:35-65`
- Modify: `word_select_dashboard/web-react/src/App.tsx:3279-3760`
- Create: `word_select_dashboard/web-react/test/wordCleanPlayback.test.ts`

- [ ] **Step 1: 写失败的页面源码契约测试**

创建测试读取 `App.tsx`，要求页面包含两种唯一播放键、基础单词字段和单词按钮：

```ts
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const source = readFileSync(new URL("../src/App.tsx", import.meta.url), "utf8");

test("word and sentence audio use distinct targets in one player", () => {
  assert.match(source, /`word:\$\{item\.id\}`/);
  assert.match(source, /`sentence:\$\{item\.id\}`/);
  assert.match(source, /item\.wordTtsStatus/);
  assert.match(source, /item\.wordTtsObjectUrl/);
  assert.match(source, /播放.*单词发音/);
});
```

- [ ] **Step 2: 运行页面契约测试并确认失败**

Run: `node --experimental-strip-types --test test/wordCleanPlayback.test.ts`

Expected: FAIL，提示缺少 `word:<id>` 播放目标或基础单词 TTS 字段。

- [ ] **Step 3: 扩展前端列表项类型**

在 `WordCleanItem` 增加：

```ts
wordTtsStatus: string;
wordTtsBucket: string;
wordTtsObjectKey: string;
wordTtsObjectUrl: string;
```

- [ ] **Step 4: 将现有播放器状态改成唯一目标键**

使用：

```ts
type CleanWordAudioTarget = `word:${number}` | `sentence:${number}`;
const [playingCleanWordAudioTarget, setPlayingCleanWordAudioTarget] = useState<CleanWordAudioTarget | null>(null);
const [loadingCleanWordAudioTarget, setLoadingCleanWordAudioTarget] = useState<CleanWordAudioTarget | null>(null);
```

统一播放入口接收 `item`、`target`、`audioURL` 和错误文案；重复点击当前加载或播放目标时调用 `stopCleanWordAudio()`，切换目标时先停止旧音频，再创建新的 `Audio`。

- [ ] **Step 5: 渲染基础单词播放按钮**

在单词文字与造句历史按钮之间增加 Ant Design 文本按钮。URL 使用：

```ts
const wordAudioURL = playableTTSAudioURL(item.wordTtsStatus, item.wordTtsObjectUrl);
const wordAudioTarget = `word:${item.id}` as const;
```

无 URL 时禁用；加载时显示加载；播放时显示 `PauseCircleOutlined`；错误信息为 `无法播放 ${item.word} 的单词发音`。例句按钮使用 `sentence:${item.id}` 调用同一入口。

- [ ] **Step 6: 运行页面契约测试和纯函数测试**

Run: `node --experimental-strip-types --test test/wordAudio.test.ts test/wordCleanPlayback.test.ts`

Expected: 全部 PASS。

- [ ] **Step 7: 运行 TypeScript 与生产构建**

Run: `npm run build`

Expected: TypeScript 检查通过，Vite 构建成功。

- [ ] **Step 8: 提交前端变更**

```bash
git add word_select_dashboard/web-react/src/App.tsx word_select_dashboard/web-react/src/types/wordLibrary.ts word_select_dashboard/web-react/src/utils/wordAudio.ts word_select_dashboard/web-react/test/wordAudio.test.ts word_select_dashboard/web-react/test/wordCleanPlayback.test.ts
git commit -m "feat: play base word tts from minio"
```

### Task 4: 运行态验收

**Files:**
- Verify: `word_select_dashboard/server`
- Verify: `word_select_dashboard/web-react`

- [ ] **Step 1: 运行全部相关自动化验证**

Run: `go test ./...`

Run: `node --experimental-strip-types --test test/*.test.ts`

Run: `npm run build`

Expected: 所有命令退出码为 0。

- [ ] **Step 2: 抽样验证 MinIO 音频响应**

从 `clean-words` 接口选取 `wordTtsStatus=success` 的记录，请求其 `wordTtsObjectUrl`。确认 HTTP 200、音频 Content-Type 和非空响应体。

- [ ] **Step 3: 浏览器验证互斥播放**

在 7001“去重单词表”点击基础单词播放按钮，确认进入播放状态；随后点击同一行例句播放按钮，确认基础单词音频停止、例句开始播放；再次点击当前按钮确认停止并复位。
