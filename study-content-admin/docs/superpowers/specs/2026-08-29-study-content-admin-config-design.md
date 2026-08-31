# Study Content Admin · 配置管理接入设计

日期：2026-08-29  
状态：已评审（对话确认 §1–§4）

## 目标

新建独立项目 `study-content-admin`（路径 `/Users/conchi/workforce/go_workforce/study-content-admin`），作为学习内容后台骨架。

第一版只做「配置管理」只读菜单，完整对齐 Shared Config Center / 英语材料的七项导航。配置单据仍只存在 Shared Config Center；本项目不编辑、不自建数据库。

后续（本 spec 不实现）：识字等内容菜单、按知识点生成「字图 / 实物图」、与 `study_workbench` 题库联动。

## 背景与参考

- `study_workbench` 目前**尚未**接入 Shared Config Center。
- 完整消费范例是 `english_material`：后端 `GET /api/runtime/v1/configuration` 拉运行时快照；前端自绘只读配置页；编辑引导至配置中心。
- 配置中心仓库：`/Users/conchi/workforce/go_workforce/shared-config-center`
- Go 客户端示例：`shared-config-center/examples/go/client.go`

## 决策摘要

| 项 | 选择 |
|----|------|
| 配置菜单形态 | 只读展示（方案 A），不跳转 iframe、不直连浏览器调配置中心 |
| 路径 / 名称 | `go_workforce/study-content-admin` |
| 技术栈 | Go 后端 + React 前端 |
| 子菜单范围 | 全套七项 |
| 自有数据库 | 第一版不要 |
| 接入方式 | 后端 Runtime 代理 + 本项目 catalog API |

## §1 仓库结构与端口

```
study-content-admin/
├── README.md
├── docker-compose.yml
├── Makefile
├── backend/                 # Go（Gin）
│   ├── cmd/server/
│   └── internal/
│       ├── configclient/    # load / refresh / current
│       └── http/            # /healthz + catalog / refresh
└── frontend/                # React + Vite + TypeScript
    └── src/
        ├── layout/          # 侧栏：仅「配置管理」及其子项
        └── features/config/ # 七个只读页
```

| 服务 | 地址 |
|------|------|
| 本项目 Web + API | http://localhost:19091 |
| Shared Config Center（已有） | http://localhost:18427（API 直连常见为 :18783） |

环境变量：`SHARED_CONFIG_CENTER_BASE_URL`（容器内指向 `http://shared-config-center-api:8080`）。

对外只占一个宿主机端口 `19091`：后端 embed 前端 build 产物（与 study_workbench 家长端类似）。

## §2 后端 API 与配置加载

### 配置客户端

- 启动时可选调用一次 `Load()`：`GET {BASE}/api/runtime/v1/configuration`
- `schemaVersion` 必须为 `"1"`，否则视为不可用
- 内存快照；**无轮询**；仅管理员点「刷新」时 `Refresh()`
- 另拉：`GET /api/admin/v1/configuration/image-models`、`GET /api/admin/v1/configuration/video-models`
- 配置中心不可达：**进程仍可启动**；真正读 catalog 时返回：「无法加载共享配置中心」

### 本项目对外 API（只读）

| 方法 | 路径 | 作用 |
|------|------|------|
| GET | `/healthz` | 进程健康（不依赖配置中心） |
| GET | `/api/v1/runtime-config/catalog` | 组装前端 catalog（含明文密钥，内网约定） |
| POST | `/api/v1/runtime-config/refresh` | 强制再拉配置中心 |

### catalog 形状（对齐英语材料前端）

- `databases[]`、`ai`、`localCli`、`objectStorage`
- `imageModels`、`videoModels`（来自筛选接口）
- `runtime`：`schemaVersion` / `generatedAt` / 原始 JSON 字符串（Runtime 页用）

### 明确不做

- 任何 PUT/DELETE 到配置中心
- 本机默认项持久化
- 测连、执行 CLI、调 MinIO

## §3 前端菜单与页面

### 壳子

- 顶栏品牌：Study Content Admin / 学习内容后台
- 左侧导航第一版**只有「配置管理」一组**，子项：
  1. 数据库配置
  2. AI 配置
  3. 本地 CLI 配置
  4. MinIO 配置
  5. 图片模型配置
  6. 视频模型配置
  7. Runtime Contract
- 其他学科 / 出图菜单不出现

### 页面行为

- 进入配置区：`GET /api/v1/runtime-config/catalog`
- 各页只读展示（含明文密钥）；不嵌入配置中心 iframe
- 「刷新配置」→ `POST /api/v1/runtime-config/refresh` 再重载
- 文案提示：修改请到 Shared Config Center
- 失败态：整页 Alert，不伪造默认配置

### 路由

```
/config/databases          （默认入口）
/config/ai
/config/local-cli
/config/object-storage
/config/image-models
/config/video-models
/config/runtime
```

## §4 Docker / 部署与验收

### Compose

- 单服务 `backend`（embed 前端静态资源），端口 `19091:19091`
- 加入外部网络 `vibedeploy-shared`
- `SHARED_CONFIG_CENTER_BASE_URL=http://shared-config-center-api:8080`
- 不依赖本项目 Postgres；前提是 Shared Config Center 已在运行
- 若本机尚无 `vibedeploy-shared`，文档说明先 `docker network create vibedeploy-shared` 并启动配置中心

### 本地开发

```bash
# 配置中心已 up
cd backend && SHARED_CONFIG_CENTER_BASE_URL=http://127.0.0.1:18783 go run ./cmd/server
cd frontend && npm run dev   # 代理 /api → 本后端
```

### 验收清单

1. `make up` 后打开 http://localhost:19091，侧栏仅配置管理七项
2. 配置中心有数据时，七页都能只读展示（含密钥）
3. 「刷新」后能看到配置中心刚改过的值
4. 关掉配置中心后，catalog 返回明确失败文案，`/healthz` 仍 200
5. 本项目无任何写入配置中心的接口

## 职责边界

| 系统 | 职责 |
|------|------|
| Shared Config Center | AI / 数据库 / MinIO / CLI 单据的唯一事实源与编辑入口 |
| study-content-admin v1 | 只读展示 + 内存快照，为后续出图保留客户端能力 |
| study_workbench | 孩子答题 / 家长看板；与本项目 v1 无运行时耦合 |

## 风险与约束

- 密钥明文出现在内网管理页；不得写入日志、文档快照或对外截图
- 无自有库则无法保存「本工作台默认图片模型」；后续出图迭代若需要再加库或轻量文件存储
- 依赖外部网络名 `vibedeploy-shared` 与配置中心容器别名

