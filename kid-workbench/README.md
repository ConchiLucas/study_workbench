# 儿童学习工作台

家长看板 + 孩子答题端，共用一套 Go 后端与 PostgreSQL。

## Docker 启动（推荐）

```bash
make up
```

| 服务 | 地址 |
|------|------|
| 家长看板 + API | http://localhost:19081 |
| 孩子答题端 | http://localhost:19082 |
| PostgreSQL | localhost:15432 |

首次如需 60 天演示数据：

```bash
make seed-demo
```

其他命令：

```bash
make down    # 停止
make logs    # 查看日志
make seed    # 重新灌 catalog + 题库（幂等）
```

## 本地开发（可选）

需本机 PostgreSQL 监听 `15432`，或使用 Docker 只起数据库：

```bash
docker compose up postgres -d
```

```bash
# 后端
cd parent-dashboard/backend
go run ./cmd/seed -mode=catalog
go run ./cmd/seed -mode=questions
make run                    # http://localhost:19081

# 家长看板前端（另开终端）
cd parent-dashboard/frontend
npm install && npm run dev    # http://localhost:19083

# 孩子端（另开终端）
cd kid-app
npm install && npm run dev    # http://localhost:19082
```

## 目录

```
kid-workbench/
├── docker-compose.yml
├── parent-dashboard/
│   ├── backend/     # Go API（Docker 内嵌家长看板静态资源）
│   └── frontend/    # 家长看板 React 源码
└── kid-app/         # 孩子答题端 React 源码
```

详细功能说明见 [parent-dashboard/README.md](parent-dashboard/README.md)。
