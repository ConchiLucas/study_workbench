# 英语工作空间统一运行配置设计

## 目标

为英语单词工作空间提供一个可迁移的 Docker 全量启动入口。每台电脑只维护仓库根目录的一份私有配置，六个项目、宿主机 CLI Runner、PostgreSQL、Redis 和 MinIO 的差异化连接信息都从该配置派生。

配置正确时，用户只需执行：

```bash
./deploy-compose-full.sh
```

脚本应在构建前一次性暴露配置问题，并在启动后验证真实运行状态，不能把“执行了 `docker compose up -d`”等同于服务可用。

## 已确认需求

- 只保留一个配置入口，不按电脑、环境或依赖位置维护多套脚本或 profile。
- 根目录 `.env.local` 是每台电脑唯一需要编辑的私有配置。
- Git 提交完整但不含真实凭据的根 `.env.example`，不提交 `.env.local`。
- 数据库、Redis、MinIO、CLI 命令路径和可调整端口等机器差异统一进入根配置。
- 同一地址只配置一次；脚本自动处理宿主机与 Docker 容器的寻址差异。
- 六个项目继续使用各自 Compose 构建和运行，但由根脚本统一传递配置。
- 启动前失败应汇总缺失项和冲突项；启动后失败应指出具体服务并保留日志定位命令。
- 不要求用户手工复制或维护 `word-agent/.env` 与 `server/config.yaml`。

## 非目标

- 不把数据库自身的连接信息保存到业务数据库中。
- 不引入配置中心、远程密钥服务或按环境拆分的多套配置。
- 不自动创建、删除或覆盖用户已有数据库、Redis、MinIO 数据。
- 不修改六个项目的业务功能、接口契约或数据模型。
- 不承诺绕过缺失的第三方服务、无效凭据或未安装的 Codex/Gemini CLI；这些问题应在检查阶段明确报告。

## 方案选择

### 方案一：根 `.env.local`，运行时自动派生

根目录维护一份 shell/Compose 兼容的 `.env.local`。全量脚本加载基础字段，生成当前进程使用的宿主机和容器变量，再传给六个 Compose 与 CLI Runner。

优点是依赖最少、符合现有 Compose 结构、迁移时只复制一个文件。采用此方案。

### 方案二：根 YAML 加渲染器

使用结构化 YAML 保存所有配置，再依赖 Python、`yq` 或专用二进制转换成环境变量。

该方案层级清晰，但在 Docker 启动前增加额外工具依赖，反而削弱新电脑上的可执行性，不采用。

### 方案三：配置保存到数据库

应用启动后读取业务运行配置可以继续使用数据库，但 PostgreSQL 自身地址、账号和密码无法依赖同一个数据库提供，否则会形成“先连接数据库才能获得数据库连接信息”的循环，不采用。

## 单一配置模型

根 `.env.example` 按服务组件组织基础字段，至少覆盖：

- `SELECT_DB_HOST`、`SELECT_DB_PORT`、`SELECT_DB_NAME`、`SELECT_DB_USER`、`SELECT_DB_PASSWORD`、`SELECT_DB_SSLMODE`
- `ROB_WORD_DB_HOST`、`ROB_WORD_DB_PORT`、`ROB_WORD_DB_NAME`、`ROB_WORD_DB_USER`、`ROB_WORD_DB_PASSWORD`、`ROB_WORD_DB_SSLMODE`
- `REDIS_HOST`、`REDIS_PORT`、`REDIS_PASSWORD`
- `MINIO_HOST`、`MINIO_PORT`、`MINIO_ACCESS_KEY`、`MINIO_SECRET_KEY`、`MINIO_BUCKET`、`MINIO_USE_SSL`
- 可选的 `CODEX_COMMAND_PATH`、`GEMINI_COMMAND_PATH`
- 六个服务和 CLI Runner 的宿主机端口；没有调整需求时使用仓库默认值

根配置只保存基础值，不要求用户手写多份 JDBC URL、libpq DSN 或容器地址。脚本或应用适配层根据基础值生成各技术栈需要的格式。

密码和密钥不得出现在命令行参数、日志、生成报告或 Git 跟踪文件中。

## 地址归一化

每个依赖 Host 只配置一次，含义是“宿主机可访问的地址”。

- Host 为 `127.0.0.1` 或 `localhost` 时，宿主机进程继续使用原值，Docker 容器自动使用 `host.docker.internal`。
- Host 为其他 IP 或域名时，宿主机和容器使用同一值。
- Compose 保留 `host.docker.internal:host-gateway`，兼容支持该映射的平台。

因此用户无需分别维护 `WORD_AGENT_SELECT_DB_DSN` 和 `WORD_AGENT_CLI_RUNNER_DB_DSN`。Word Agent 容器与 CLI Runner 从同一组数据库字段得到各自可达的连接配置。

## 配置消费边界

### 根启动脚本

`deploy-compose-full.sh` 是唯一正式 Docker 全量入口，负责：

1. 根据脚本自身位置解析仓库根目录，不依赖调用者当前目录。
2. 加载并验证根 `.env.local`。
3. 派生宿主机和容器使用的地址与连接变量。
4. 执行全部 Compose 的静态配置检查。
5. 启动宿主机 CLI Runner。
6. 按依赖顺序构建并启动六个项目。
7. 等待并汇总运行状态。

子项目脚本可以保留，但必须转调统一配置加载逻辑，不能再形成第二套默认值。

### Word Agent

- Compose 不再从 `word-agent/.env` 获取启动必填项。
- 数据库、MinIO 和 CLI Runner 地址全部由根脚本传入。
- Docker 启动不再挂载被 Git 忽略的 `server/config.yaml`；需要的只读默认配置改用仓库跟踪的 Docker 配置，并由显式环境变量覆盖机器差异。

### Go 管理后端

- Dockerfile 删除构建期写死账号、密码和地址的改写。
- Compose 显式传入根配置派生的数据库、Redis 和 Word Agent 变量。
- Go 配置加载层对这些机器差异字段提供环境变量覆盖，仓库 YAML 只保存无敏感信息的默认结构。

### Java 核心后端

- Compose 使用根配置派生 JDBC、Redis 和 Word Agent 地址。
- Word Agent 默认指向实际的 `6017` 服务端口，不再保留 `8010` 旧值。

### CLI Runner

CLI Runner 继续作为宿主机进程运行，因为它需要调用本机 Codex/Gemini CLI 及其登录状态。根脚本从同一配置生成宿主机数据库连接，在六个 Docker 项目之前启动并验证 `6018`。

## 启动前检查

任何镜像构建开始前，脚本一次性检查：

- `.env.local` 是否存在；缺失时提示从 `.env.example` 复制。
- 所有必填字段是否存在且非空。
- Docker 与 Docker Compose 是否可用。
- 六个 Compose 是否能通过 `docker compose config -q`。
- 需要挂载的文件是否存在。
- 目标端口是否已被非目标进程占用。
- PostgreSQL、Redis 和 MinIO 地址是否可达；检查失败只显示字段名和目标主机，不显示密码。
- 配置为 CLI 执行器时，对应命令是否存在且可执行。

检查阶段不得修改数据库或业务数据。所有配置错误汇总后一次性退出，避免用户逐项重跑才能发现下一处问题。

## 启动与错误处理

- 通过预检后才开始构建。
- 服务按 CLI Runner、后端、前端顺序启动。
- 构建或启动失败时停止后续步骤，但不自动删除已经成功构建的镜像、容器或数据。
- 输出失败服务、Compose 项目、容器状态和安全的日志查看命令。
- 不在错误输出中打印展开后的 DSN、密码、Token 或完整环境文件。

## 启动后验证

启动成功必须同时满足：

- 六个目标容器均处于运行状态，且没有重启循环。
- CLI Runner 的 `6018` 健康接口可用。
- Word Agent、Go 后端和 Java 后端的健康或代表性只读接口可用。
- 三个前端端口能够返回 HTTP 响应。
- PostgreSQL、Redis 和 MinIO 的连接检查通过。

Word Agent 当前 `/health` 只证明 HTTP 进程存活，不能单独作为数据库、CLI Runner 和 MinIO 可用性的证据。根脚本必须独立检查这些依赖。

## 安全与兼容性

- `.env.local`、运行态文件和任何派生密钥文件均加入 Git 忽略规则。
- `.env.example` 使用明显的占位值，不包含个人账号、密码或绝对路径。
- 移除版本控制文件中的真实或机器相关凭据；若现有值曾是真实凭据，应在改造后轮换。
- 保留现有容器名和默认宿主机端口，避免破坏前端代理、页面链接和运维习惯。
- 从仓库根目录或任意其他目录调用脚本，行为必须一致。

## 测试策略

### 配置加载测试

- 缺少 `.env.local` 时在构建前失败。
- 多个必填项缺失时一次性列出全部字段名。
- 回环地址为宿主机和容器派生不同地址。
- 远程 IP/域名在两侧保持不变。
- 特殊字符不会让密码进入日志，也不会破坏派生连接配置。

### 静态配置测试

- 六个 Compose 使用统一根配置后全部通过 `docker compose config -q`。
- 版本控制文件不再包含固定数据库、Redis、MinIO 凭据和 Word Agent `8010` 旧地址。
- Word Agent Docker 启动不依赖 `word-agent/.env` 与 `server/config.yaml`。

### 运行测试

- 在当前电脑使用真实 `.env.local` 完成六项目全量构建和启动。
- 验证六个容器、CLI Runner、三个后端/Agent 接口和三个前端端口。
- 使用一份替换主机、端口和账号的测试配置验证所有 Compose 展开结果来自根配置。
- 人为删除一个必填字段，确认预检在构建前失败且不泄露其他配置。

## 验收标准

- 新电脑克隆仓库后，只需创建并填写根 `.env.local`，无需进入任何子目录复制配置。
- 同一数据库、Redis 或 MinIO 地址只填写一次。
- 从任意目录执行根脚本都能定位仓库。
- 配置错误在构建前一次性报告；配置正确时六个项目与 CLI Runner 均能启动并通过验证。
- Git 中不保存真实凭据，启动日志不泄露敏感值。
