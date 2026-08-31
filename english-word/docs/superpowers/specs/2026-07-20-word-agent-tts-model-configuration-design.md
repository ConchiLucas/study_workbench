# Word Agent 独立 TTS 模型配置设计

## 背景

Word Agent 后台当前只有“大模型配置”，用于维护文本模型 Provider。Xiaomi MiMo TTS 的 Base URL、API Key、模型和默认音色则由 Word Agent 的环境变量提供，导致后台页面无法查看或切换实际使用的 TTS 配置。

本次将文本模型和 TTS 模型明确分开：保留现有“大模型配置”及其中 5 条文本模型，新增独立“TTS 模型配置”。当前英语项目内所有 MiMo TTS 生成路径统一通过 Word Agent 的 `MiMoTTSService` 读取新的数据库配置。

## 目标

1. 在 Word Agent 后台新增独立“TTS 模型配置”页面。
2. 新增独立的 TTS Provider 数据表和 Go 管理接口。
3. 将现有 Xiaomi MiMo TTS 参数安全迁移到数据库。
4. 让造句后自动 TTS、独立 TTS API 和项目内示例脚本统一使用数据库 TTS 配置。
5. 配置保存后无需重启 Word Agent 即可在下一次生成时生效。
6. 保留现有 5 条文本模型配置及其行为。

## 非目标

- 不修改 `/Users/conchi/workforce/python_workforce/ai-task-center`；两个项目不共享数据库、配置或运行链路。
- 不把 MiMo TTS 写入现有 `ai_provider_configs`。
- 不删除或改写已有音频、MinIO 对象、TTS 任务记录或句子数据。
- 不在本次接入 Codex CLI；CLI 设计作为独立后续需求处理。
- 不支持请求级别覆盖 TTS 模型或默认音色，避免绕过后台配置。

## 当前生成入口

当前项目内存在三类 MiMo TTS 生成入口：

1. `POST /v1/sentences/generate`：文本造句成功后调用 `MiMoTTSService` 生成完整句子语音，再上传 MinIO。
2. `POST /v1/tts/generate`：直接生成单词或句子的 WAV 临时文件。
3. `scripts/mimo_tts_word.py`：当前直接读取 `MIMO_API_KEY` 并请求 Xiaomi API。

两个 MinIO 迁移脚本只迁移已有音频，不生成新音频，不纳入配置读取改造。

## 总体架构

```text
React TTS 模型配置页
  -> Go /api/tts/config
  -> select_english_word.tts_provider_configs
  -> Word Agent MiMoTTSService
       -> /v1/sentences/generate
       -> /v1/tts/generate

scripts/mimo_tts_word.py
  -> Word Agent /v1/tts/generate
  -> MiMoTTSService
  -> tts_provider_configs
```

只有 Word Agent 的 `MiMoTTSService` 直接调用 Xiaomi API。项目内示例脚本改为调用 Word Agent，不再持有 MiMo 凭证或复制请求协议。

## 数据模型

新增表 `tts_provider_configs`：

| 字段 | 类型/约束 | 说明 |
| --- | --- | --- |
| `id` | bigint/主键 | 数据库主键 |
| `provider_id` | varchar(120)/唯一/非空 | 配置 ID，例如 `xiaomi-mimo-tts` |
| `label` | varchar(160)/非空 | 显示名称 |
| `type` | varchar(80)/非空 | 第一版仅支持 `mimo-tts` |
| `base_url` | varchar(500)/非空 | Xiaomi API Base URL |
| `api_key` | text/非空 | Xiaomi API Key |
| `model` | varchar(160)/非空 | TTS 模型名称 |
| `voice` | varchar(120)/非空 | 默认音色 |
| `enabled` | boolean/非空 | 是否允许运行 |
| `active` | boolean/非空 | 是否为默认 TTS 配置 |
| `created_at` | timestamp | 创建时间 |
| `updated_at` | timestamp | 更新时间 |

第一版迁移后只存在一条配置：

- `provider_id`: `xiaomi-mimo-tts`
- `label`: `Xiaomi MiMo TTS`
- `type`: `mimo-tts`
- `base_url`: `https://api.xiaomimimo.com/v1`
- `model`: `mimo-v2.5-tts`
- `voice`: `Chloe`
- `enabled`: `true`
- `active`: `true`

服务层保证配置 ID 唯一、至少保留一条配置，并保证启用配置中恰好有一条默认配置。删除或停用当前默认项前，必须先选择其他默认项。

## Go 后端接口

新增路由组：

- `GET /api/tts/config`：返回默认配置 ID 和配置列表。
- `POST /api/tts/config`：校验并事务保存完整配置列表。

请求和响应结构：

```json
{
  "active": "xiaomi-mimo-tts",
  "providers": [
    {
      "id": "xiaomi-mimo-tts",
      "label": "Xiaomi MiMo TTS",
      "type": "mimo-tts",
      "base_url": "https://api.xiaomimimo.com/v1",
      "api_key": "",
      "api_key_configured": true,
      "model": "mimo-v2.5-tts",
      "voice": "Chloe",
      "enabled": true
    }
  ]
}
```

GET 不返回真实 API Key，只返回 `api_key_configured`。POST 中 `api_key` 为空时保留数据库中的旧值；新增配置则必须提供 API Key。这样页面可以显示凭证已配置，同时避免每次读取页面都把密钥发送到浏览器。

保存成功后只更新数据库，不写入现有 `ai` YAML 节点，也不改变 `global.GVA_CONFIG.AI`。

## React 页面

在 Word Agent 左侧菜单中，将“TTS 模型配置”放在“模型配置”之后。页面沿用现有模型配置的视觉结构：

- 页头：`TTS 模型配置`
- 操作：新增配置、刷新、保存配置
- 左侧：配置列表、模型摘要、默认标记
- 右侧：配置编辑表单、设为默认、删除配置

字段包括：

- 配置 ID
- 显示名称
- 接口类型（第一版仅 `Xiaomi MiMo TTS`）
- Base URL
- API Key
- 模型名称
- 默认音色
- 启用状态

页面不显示 Max Tokens。API Key 输入框为空表示保持原密钥，用户输入新值才覆盖。

## Word Agent 配置加载

`MiMoTTSService` 不再从以下 Settings 字段读取模型配置：

- `mimo_api_key`
- `mimo_tts_base_url`
- `mimo_tts_default_model`
- `mimo_tts_default_voice`

服务在每次生成前查询 `tts_provider_configs`，条件为：

- `type = 'mimo-tts'`
- `enabled = true`
- `active = true`

必须且只能查询到一条完整配置。缺失、重复或字段为空时抛出 `TTSConfigError`，不得回退到 `.env`。

以下运行参数仍由环境变量控制，因为它们不属于模型凭证：

- TTS HTTP 超时
- SSL 校验
- 临时音频目录
- MinIO 配置

## 请求契约

`TTSGenerationRequest` 不再接受 `model` 和 `voice` 作为有效覆盖项。模型和默认音色始终来自数据库配置；请求仍可提供文本、风格、文件名、格式和覆盖文件开关。

响应继续返回实际使用的 `model` 和 `voice`，便于现有数据表保存审计信息。

`POST /v1/sentences/generate` 和 `POST /v1/tts/generate` 都继续调用同一个 `MiMoTTSService.generate()`，无需在路由层复制配置读取逻辑。

## 示例脚本

`scripts/mimo_tts_word.py` 改为 Word Agent 客户端：

- 默认服务地址为 `http://127.0.0.1:6017`，可通过命令行参数覆盖。
- 调用 `/v1/tts/generate`。
- 根据响应中的下载地址获取 WAV 并写入用户指定路径。
- 删除 `MIMO_API_KEY`、API URL、模型和音色相关参数及直接 Xiaomi 请求代码。

因此脚本生成的音频也必然使用后台 TTS 模型配置。

## 一次性迁移与部署顺序

1. Go AutoMigrate 创建 `tts_provider_configs`。
2. 在当前 Word Agent 容器内部读取已有 MiMo 环境变量，但不打印其值。
3. 使用数据库事务写入 `xiaomi-mimo-tts`，随后读取并校验非敏感字段和密钥非空状态。
4. 保持 `ai_provider_configs` 的 5 条文本模型完全不变。
5. 部署不再读取旧 MiMo Settings 的 Word Agent 版本。
6. 项目 Compose 和 `.env.example` 删除 MiMo 模型凭证示例；本地旧 `.env` 即使仍有旧变量也不会被运行时代码读取。

迁移失败时回滚数据库事务，旧 Word Agent 容器继续运行；只有数据库配置校验成功后才重建 Word Agent。

## 错误处理

- TTS 配置缺失或无默认项：HTTP 500，明确返回配置错误。
- Xiaomi 请求失败、超时或响应格式错误：HTTP 502。
- 造句后的 TTS 失败：保持现有整条造句请求失败语义，不写入缺少音频的挖空题。
- 独立 TTS API 失败：不留下不完整文件。
- 不在错误消息、日志、API 响应或测试快照中包含 API Key。

## 测试策略

### Go

- TTS 配置规范化、重复 ID、默认项、启用状态和 API Key 保留测试。
- 数据库保存、删除和事务回滚测试。
- GET 不暴露 API Key 的接口测试。
- AutoMigrate 包含新表的测试。
- 现有 AI 配置测试证明 5 条文本模型逻辑不受影响。

### React

- 新菜单和页面路由测试。
- TTS 配置加载、编辑、设为默认和保存测试。
- API Key 留空保留旧值的请求契约测试。
- 现有大模型配置页面回归测试。

### Python Word Agent

- 从数据库加载唯一默认 MiMo 配置。
- 配置缺失、重复、停用和字段不完整的失败测试。
- 证明环境变量不再作为回退。
- 造句后自动 TTS 和独立 TTS API 都使用数据库返回的模型与音色。
- 示例脚本只调用 Word Agent，不直接访问 Xiaomi。

### 运行态

- 数据库确认 `tts_provider_configs` 只有一条启用默认 MiMo 配置。
- 数据库确认 `ai_provider_configs` 原有 5 条文本配置仍存在。
- Docker 内 Word Agent 健康检查通过。
- 后台 TTS 配置页读取成功且不返回明文密钥。
- 使用隔离测试覆盖实际生成协议；若需要真实生成一条音频，将在明确控制调用次数的情况下执行。

## 安全与回滚

- API Key 只在数据库、Go 保存请求和 Word Agent 到 Xiaomi 的请求中出现。
- 不把密钥写入设计文档、日志、测试数据快照或版本库。
- 数据迁移为单事务，失败即回滚。
- 在 Word Agent 新版本启动失败时，可继续运行旧容器；数据库新增表不会影响现有文本模型和旧服务。
