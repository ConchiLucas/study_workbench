# 科普素材页 · 字图 / 义图 / 读音

**日期：** 2026-08-30  
**范围：** study-content-admin 新增「科普」导航与素材页（镜像英语）  
**题库来源：** study_workbench `subjects.code = 'science'`（约 99 KP / 10 组）  
**Mockup：** `docs/superpowers/mockups/2026-08-30-science-assets.html`

## 目标

在内容后台提供与英语页同构的科普素材管理：从题库同步知识点，浏览/筛选，展示并生成**字图、义图、读音**；默认识图全开且可手动覆盖。页面不挂批量按钮，批量走 API（多 worker，优先 Grok）。

## 已确认决策

| 项 | 选择 |
|----|------|
| 素材范围 | 字图 + 义图 + 读音（对齐英语） |
| 字图 | 整段标题渲染为一张字卡 |
| 义图规则 | 同步默认全要，可 PATCH 覆盖为不要 |
| 批量 UI | 不挂按钮；API `.../batch?workers=` |
| 实现路线 | 新建 `science` 包/页，镜像 english，不抽通用框架 |

## UI

- 顶栏主导航：识字 / 拼音 / 算术 / 英语 / **科普** / 题目管理 / 配置管理  
- 路由：`/science` → `SciencePage`  
- 控件：从题库同步；按组 | 表格；义图筛选（全部 / 要 / 不要）；搜索标题  
- 卡片：标题、字图、义图、要/不要义图、试听读音  
- 复用现有 `literacy-page` / 英语页样式类，文案改为科普

## 数据

表 `science_assets`（字段对齐 `english_assets`，文本列用 `title`）：

- `kp_id` PK  
- `title`  
- `module_code` / `module_name` / `module_order` / `kp_order`  
- `needs_sense_image`（同步写入 `true`）  
- `needs_sense_image_override`（nullable）  
- `glyph_image_url` / `sense_image_url` / `speech_audio_url`  
- `synced_at` / `updated_at`  

`EffectiveNeedsSenseImage()`：有 override 用 override，否则用 `needs_sense_image`。

MinIO keys：`science/glyphs/{id}.png`、`science/senses/{id}.png`、`science/speech/{id}.mp3`。

## API（`/api/v1/science`）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/sync` | 从 workbench 同步 `science` |
| GET | `/items` | `view=groups\|table`，`needsSenseImage` 筛选 |
| PATCH | `/items/:kpId` | 更新义图 override |
| POST | `/items/:kpId/glyph` | 生成字图（整段标题） |
| GET | `/items/:kpId/glyph.png` | 取字图 |
| POST | `/glyphs/batch` | 批量字图 |
| POST | `/items/:kpId/sense` | 生成义图 |
| GET | `/items/:kpId/sense.png` | 取义图 |
| POST | `/senses/batch` | 批量义图，`workers` / `maxRetries` / `moduleCode` |
| POST | `/items/:kpId/speech` | 生成读音 |
| GET | `/items/:kpId/speech.mp3` | 取读音 |
| POST | `/speech/batch` | 批量读音 |

公开 URL 形如：`{APP_PUBLIC_BASE}/api/v1/science/items/{kpId}/sense.png`。

## 生成逻辑

- **字图：** 本地/现有 glyph renderer，输入为 `title` 全文（多字换行策略与英语长词一致或略收紧字号）。  
- **义图：** `sense.PromptScience(title)`（儿童友好卡通插画，表现该科普概念；禁止画面出字）。`NeedsSenseImage` 规则函数恒为 `true`（空排除表）。生成时若 `!EffectiveNeedsSenseImage()` 则拒绝。Provider 选择与英语相同（优先 Grok）。  
- **读音：** TTS 朗读 `title`。  
- **批量：** 复制英语 worker 池 + 失败重试退避；只处理 URL 为空的行。

## 文件地图

| 新建/修改 | 作用 |
|-----------|------|
| `backend/internal/science/*` | Service、rules、DTO、Sync |
| `backend/internal/http/handler_science.go` + `router.go` | HTTP |
| `backend/internal/db` / migrate | `science_assets` |
| `backend/internal/storage` | MinIO key helpers |
| `backend/internal/sense` | `PromptScience` |
| `cmd/server/main.go` | 注入依赖 |
| `frontend/src/features/science/SciencePage.tsx` | 页面 |
| `frontend/src/api/science.ts` + `scienceTypes.ts` | API 客户端 |
| `frontend/src/App.tsx` / `layout/AppShell.tsx` | 路由与导航 |

题库种子仍留在 workbench（`catalog_science.go`）；admin 只同步、不改题干。

## 非目标

- 不在本页做题目预览 / 出题任务  
- 不把 payload（q/a/wrong/emoji）做成可编辑 CMS  
- 不抽跨学科通用 Asset 框架  
- 不改 kid-app / parent-dashboard 答题协议（除非另开需求接义图 URL）

## 测试要点

- Sync 后 10 组、条目数与题库一致；`needs_sense_image=true`  
- Override 关闭后单条/批量跳过义图  
- 字图含多字标题可读；义图/读音 URL 可访问  
- `senses/batch?workers=4` 只补缺失且可重试  
- 非科普学科页面不受影响  
