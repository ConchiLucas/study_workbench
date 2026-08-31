# Context Router 运行配置

本目录和六个 Project 根目录中的 `deploy/context-router/` 是运行配置唯一事实源。Context Router 数据库只保存这里文件的同步副本。

## 不依赖 Context Router 启动

启动整个 Workspace：

```sh
./deploy/context-router/workspace/start/deploy.sh
```

单 Project 启动时执行对应项目的 `fast/deploy.sh` 或 `full/deploy.sh`。当前两个模式都复用根 `deploy-compose-full.sh --project <key>`；后续需要不同构建策略时，可以直接修改对应入口，再从 Workspace 页面重新同步。

## 文件规范

- 根 `manifest.yaml` 使用 `schema_version: 1`。
- `project_order` 使用 Project 相对 Workspace 的路径，必须覆盖全部登记项目且不能重复。
- Workspace start 目录及每个 Project 的 fast/full 目录都必须包含可执行的 `deploy.sh`。
- 所有文件必须是 UTF-8 普通文本；不允许软链接、绝对路径、`..` 或越出 Workspace。
- 不得放入 `.env.local`、密码、Token、私钥或其他机器秘密；本机私有配置仍只保存在根 `.env.local`。
- 脚本必须能脱离 Context Router 直接运行。`WORKSPACE_HOST_ROOT` 只是 Runtime Runner 提供的可选覆盖值。

同步按钮会先扫描和校验整个 Workspace，展示摘要及增删改统计；确认后才在单个数据库事务中替换 Workspace start、policy 和六个 Project 的 fast/full 配置。任一文件失败时，数据库旧配置保持不变。
