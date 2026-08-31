# Study Content Admin · 识字素材列表设计

日期：2026-08-29  
状态：已评审（对话确认 §1–§4）

## 目标

在 `study-content-admin` 增加「识字」菜单。第一版只做：

- 从同库 `study_workbench` 同步识字知识点列表
- 系统自动判定是否需要「义图」
- 支持人工覆盖判定
- 按组 / 表格双视图 + 筛选

不做：生成字图/义图按钮、MinIO 上传、改 workbench 题干。

## 决策摘要

| 项 | 选择 |
|----|------|
| 字表来源 | 同步 study_workbench 识字 KP（约 130 字） |
| 义图判定 | 系统规则词表；可人工 override |
| 出图 | 本版不做 |
| 列表 UI | 按组 + 表格（可切换、可筛） |
| 数据库 | 与 study_workbench **同一 PostgreSQL 库** |
| 字图风格（后续） | 程序模板统一渲染；义图另用统一 prompt |

## §1 数据与职责

### 已有（workbench）

`subjects` / `modules` / `knowledge_points`：识字「第 N 组」与单字。

### 新增表 `literacy_assets`（同库）

| 列 | 说明 |
|----|------|
| `kp_id` PK FK → knowledge_points | |
| `char_text` | 汉字 |
| `module_code` / `module_name` / `module_order` / `kp_order` | 分组与排序 |
| `needs_sense_image` | 系统判定 |
| `needs_sense_image_override` | 人工覆盖，NULL=跟系统 |
| `glyph_image_url` / `sense_image_url` | 预留，本版空 |
| `synced_at` / `updated_at` | |

生效值：`coalesce(override, needs_sense_image)`。

同步：upsert KP 元数据 + 重算系统判定；**不覆盖**已有 override。

## §2 义图判定 + API

### 规则

1. 在「不要义图」词表 → false  
2. 否则 → true  

不要义图表（第一版，含当前库内虚词/指代等，并预留扩展）：

的 了 不 在 有 是 我 你 他 她 们 这 那 吗 呢 吧 着 过 和 与 也 很 就 都 把 被 让 给 从 向 对 比 为 以 而 且 或

边界字如「一」「大」「上」默认 **要** 义图。

### API

| 方法 | 路径 | 作用 |
|------|------|------|
| POST | `/api/v1/literacy/sync` | 同步 + 跑规则 |
| GET | `/api/v1/literacy/chars?view=groups\|table&needsSenseImage=` | 列表 |
| PATCH | `/api/v1/literacy/chars/:kpId` | body: `{ "needsSenseImageOverride": true\|false\|null }` |

## §3 前端

- 侧栏增加「识字」→ `/literacy`
- 同步按钮；视图：按组 | 表格；筛选：全部 / 要义图 / 不要义图
- 按组：折叠组卡片，字 + 标签，可改 override
- 表格：字、组别、系统、覆盖、生效；可搜字
- 无生成/上传入口

## §4 DSN 与验收

- `APP_DSN` / `APP_DB_*` 指向 workbench 同一库（本机常见 `127.0.0.1:15432` / `study_workbench`）
- Docker：`host.docker.internal:15432` 或接入 workbench 网络；不另起 Postgres
- 启动时自动 migrate `literacy_assets`

验收：

1. 侧栏有识字；同步后约 130 字、可分组/表格/筛选  
2. 的/了/是/我… 系统不要义图；人/口/火… 要  
3. override 持久；再同步不丢  
4. 无出图入口  

## 后续（不在本 spec）

字图模板渲染、义图生成、写入 URL 字段、与答题端联动。
