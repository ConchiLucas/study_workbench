# 儿童学习工作台

两个界面共用一套后端数据：

- **家长看板**（`parent-dashboard/frontend`，电脑）：一眼看出孩子哪些学会了、哪些没学会、哪些需巩固、哪些该复习。
- **孩子答题端**（同级目录 `kid-app/`，iPad 横屏）：每天一份 10 道题的练习，一题一屏、语音读题、零打字。

孩子作答直接写进 `attempts`，掌握度、当日统计、家长看板全部自动更新——两端不是两套数据。

## 数据库

默认连接本机 Postgres（Docker Compose 映射到 `15432`）：

| 项 | 默认值 |
|----|--------|
| Host | `127.0.0.1` |
| Port | `15432` |
| User | `conchi` |
| Password | `conchi123456` |
| Database | `study_workbench` |

可用环境变量覆盖：`APP_DB_HOST` / `APP_DB_PORT` / `APP_DB_USER` / `APP_DB_PASSWORD` / `APP_DB_NAME`，或直接设完整 `APP_DSN`。

## 快速启动（Docker）

在仓库根目录：

```bash
make up
```

| 服务 | 地址 |
|------|------|
| 家长看板 + API | http://localhost:19081 |
| 孩子答题端 | http://localhost:19082 |

首次可选演示数据：`make seed-demo`

## 本地开发（可选）

```bash
# 1. 后端：灌数据并启动 API
cd backend
go run ./cmd/seed -mode=catalog
go run ./cmd/seed -mode=questions
go run ./cmd/seed -mode=demo -days=60   # 可选
make run          # http://localhost:19081

# 2. 家长看板前端（另开终端）
cd frontend
npm install
npm run dev       # http://localhost:19083 （代理 /api → 19081）

# 3. 孩子答题端（另开终端，独立前端工程）
cd ../kid-app
npm install
npm run dev       # http://localhost:19082
```

## 功能一览

### 家长看板

- **总览**：四状态卡片 + 今日答题卡 + 学科进度条 + 掌握矩阵 + 30 天趋势
- **学科详情**：模块折叠的知识点方块矩阵（颜色=掌握状态）
- **需要关注**：易错点与到期复习列表
- **任务列表**（`/tasks`）：每天的答题任务；点进去看 10 道题、对错、孩子选了什么
- **奖励商店**：小红花兑换

### 孩子答题端（`kid-app`）

独立前端工程，详见 [`../kid-app/README.md`](../kid-app/README.md)。用 iPad Safari 打开 `http://localhost:19082`，「添加到主屏幕」后全屏使用。

### 相关接口

`GET /plans/today` · `POST /plans/today` · `GET /plans/:pid/review` ·
`POST /plans/:pid/start` · `POST /plans/:pid/items/:itemId/answer` ·
`POST /plans/:pid/finish` · `POST /plans/extra` · `GET /plans`

## 题库

`-mode=questions` 从知识点自动生成四选一题目，共 1274 道，覆盖 654 个知识点。
只有 `subjects.quiz_enabled` 为真的学科参与出题：

| 学科 | 题目数 | 题型 |
|------|--------|------|
| 识字 | 600 | 听读音选字（干扰项为形近字，已排除同音字） |
| 拼音 | 89 | 听例字找音 / 听单读音选字母（干扰项为易混字母） |
| 算术 | 136 | 算式四选一 / 看图数数（干扰项为邻近数） |
| 英语 | 367 | 听单词选拼写 / 听单词选图 |
| 科普 | 198 | 听常识选答案 |
| 古诗 | 100 | 听首句选诗名 / 选下一句 |
| 逻辑 | 124 | 找规律 / 分类 / 排序 |
| 成语 | 64 | 听成语选释义 / 点出成语 |
| 英语短句 | 64 | 听短句选中文 / 点出英语 |

游戏科知识点仍是占位，暂不参与出题（家长看板照常显示）。

朗读走浏览器自带的 Web Speech API，不依赖后端 TTS。拼音的读音用汉字承载
（声母 `b` 读「波」、韵母 `ang` 读「昂」），否则 TTS 会把字母念成英文字母名。

英语的时间词（`morning`/`noon`/`evening`…）只有「听音选词」没有「看图选词」——
🌅🌞🌇🌆🌃 在 iPad 上肉眼分不出来，分不清的图不如不给图。

## 掌握状态

| 状态 | 颜色含义 |
|------|----------|
| 已掌握 mastered | 绿 |
| 待复习 review_due | 蓝（曾掌握但到期） |
| 学习中 learning | 黄 |
| 需巩固 shaky | 红 |
| 未开始 not_started | 灰 |

## 技术栈

- 后端：Go · Gin · GORM · **PostgreSQL**（Docker `postgres16`，单测仍用 SQLite 内存库）
- 前端：React 18 · TypeScript · Vite · Tailwind · ECharts · TanStack Query

家长端五个页面按需加载，把 ECharts（压缩后 1.1 MB）挡在 `/kid` 之外——
孩子端首屏只有 294 kB，iPad 打开快很多。
