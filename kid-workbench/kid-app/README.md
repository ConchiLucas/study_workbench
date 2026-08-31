# 练一练 · 孩子答题端

iPad 横屏答题应用，与 [parent-dashboard](../parent-dashboard) **共用同一套 Go 后端**，前端完全独立。

## 快速启动（Docker）

在仓库根目录：

```bash
make up
```

孩子端：http://localhost:19082  
iPad 局域网访问：`http://<电脑局域网IP>:19082`

## 本地开发（可选）

```bash
# 1. 先启动后端（在 parent-dashboard 目录）
cd ../parent-dashboard/backend
make run          # http://localhost:19081

# 2. 本前端（另开终端）
cd kid-app
npm install
npm run dev       # http://localhost:19082
```

## 功能

- 任务列表：今天的练习 + 最多 2 张补做卡
- 一题一屏、语音读题、零打字
- 做完结算：星星 + 小红花
- PWA：可「添加到主屏幕」全屏使用

## 环境变量

| 变量 | 默认 | 说明 |
|------|------|------|
| `VITE_API_PROXY` | `http://localhost:19081` | 本地 dev 时 Vite 代理目标 |

## 目录结构

```
kid-app/
  src/
    api/          # 仅孩子端需要的接口
    pages/        # 首页、答题、结算
    components/   # 题目 UI、选项按钮等
    hooks/        # 音效、语音
```

后端 API 文档见 parent-dashboard 的 README。
