# Study Content Admin · 识字义图生成（MVP）

日期：2026-08-29  
状态：已确认方向，先做第 2 组

## 目标

为 `effectiveNeedsSenseImage=true` 的字生成**风格统一的义图**（儿童扁平闪卡插画），存 MinIO，列表里显示在字图下方。

## 风格锁死（修订 A）

- 纯白底、单个具象物体居中
- Flat vector / 粗黑描边、亮色、圆角
- **图上禁止任何文字**：汉字、字母、数字、标签、书法、印章；也禁止把物体摆成汉字字形
- Prompt / subject **只用英文**，绝不把目标汉字写入 prompt（否则模型会直接画出字形）
- 物体描述按字映射英文 subject（见 `internal/sense/prompt.go`）

## 技术

| 项 | 选择 |
|----|------|
| 模型 | Shared Config Center `image-models`（openai-compatible，`b64_json`） |
| 画幅 | 优先方形 `1024x1024` / Grok `aspect_ratio=1:1`；失败则用 provider `options.size` |
| 存储 | MinIO `literacy/senses/{kpId}.png` |
| 浏览器 URL | 本服务代理 `/api/v1/literacy/chars/:kpId/sense.png` |

## API

| 方法 | 路径 | 作用 |
|------|------|------|
| POST | `/api/v1/literacy/chars/:kpId/sense` | 单字生成（覆盖） |
| GET | `/api/v1/literacy/chars/:kpId/sense.png` | 代理 PNG |
| POST | `/api/v1/literacy/senses/batch?moduleCode=&workers=3&maxRetries=3` | 批量：该组内要义图且 URL 空；**并发工人池** + **失败重试队列** |

并发默认 `workers=3`（可用环境变量 `SENSE_WORKERS` 或 query 覆盖，上限 8）。失败任务进入重试队列，**最多再试 3 次**（`maxRetries` 默认且上限均为 3；query 传更大值也会被压到 3），轮间指数退避。

## UI

- 卡片：字图下方显示义图预览；按钮「生成义图 / 重新生成」
- 组头：可「生成本组未生成义图」

## 不做（本版）

全库一键、人工改 prompt UI、义图审核工作流、参考图一致性。
