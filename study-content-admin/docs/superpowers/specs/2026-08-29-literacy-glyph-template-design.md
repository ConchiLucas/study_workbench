# Study Content Admin · 识字字图模板生成设计

日期：2026-08-29  
状态：已确认（方案 1）

## 目标

为识字条目生成**风格统一的字图**（大字 PNG），存 MinIO，列表可预览。本版不生成义图。

## 模板常量

| 项 | 值 |
|----|-----|
| 字体 | Noto Serif SC Regular |
| 画布 | 1024×1024 |
| 背景 | #FFFFFF + 浅田字格（外框 + 十字，线色 #B8B8B8，线宽 12px，便于列表缩略可见） |
| 字色 | #1A1A1A |
| 布局 | 水平垂直居中；田字格内边距约 12% |
| 格式 | PNG |

渲染：Go `freetype` / `truetype`，**不**走文生图。

## 存储

- MinIO 配置来自 Shared Config Center `objectStorage`
- Object key：`{basePath}/literacy/glyphs/{kpId}.png`（basePath 可空）
- 写回 `literacy_assets.glyph_image_url`（可访问 URL）

## API

| 方法 | 路径 | 作用 |
|------|------|------|
| POST | `/api/v1/literacy/chars/:kpId/glyph` | 生成单字字图（覆盖） |
| POST | `/api/v1/literacy/glyphs/batch` | 仅为 `glyph_image_url` 为空的字生成 |

## UI

- 顶栏：批量生成未生成字图
- 卡片/表格：有图显示预览；无图显示「生成字图」按钮

## 不做

义图生成、改模板 UI、人工改要不要义图。
