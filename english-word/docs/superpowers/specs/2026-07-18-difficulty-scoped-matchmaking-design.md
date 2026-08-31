# 7002 按所选难度匹配设计

## 目标

在 `rob_english_word_front` 的“单人训练”设置界面增加“开始匹配”入口。用户从现有难度选择器选定难度后，只能匹配选择了完全相同 `difficultyLevel` 的真人对手；对局单词也必须来自该难度对应的词库。

没有相同难度的对手时持续等待，直到匹配成功或用户主动取消。不自动放宽难度，不自动切换机器人，也不从其他难度补充单词。

## 交互设计

- “单人训练”面板的按钮顺序为：难度选择、开始匹配、开始训练、查看答题结果、已掌握单词。
- 新“开始匹配”复用主页紫色匹配按钮的视觉风格。
- 主页原“开始匹配”保留，等价于选择 `rank / rank_current` 后开始匹配。
- 新按钮发送当前已选择的 `difficultyGroup` 和 `difficultyLevel`。
- 等待弹窗显示后端返回的权威难度名称，例如“正在匹配：初中英语 · 7年级上册”。
- 匹配期间禁用难度选择、开始匹配、开始训练、结果和掌握单词入口。
- 用户可以继续使用现有“取消匹配”按钮。取消后停留在单人训练设置界面，并保留已选择的难度。

## 难度定义

“难度相同”指双方经过后端校验和规范化后的 `difficultyLevel` 完全相同：

- `rank_current` 只与 `rank_current` 匹配。
- 大类 `junior` 只与大类 `junior` 匹配。
- 具体教材 `junior_7_1` 只与 `junior_7_1` 匹配，不与 `junior` 或其他初中教材匹配。

后端新增统一难度目录，集中维护合法的 group/level 组合、中文名称和对应 `word_library` 名称。前端传入的名称不作为权威数据，避免伪造和前后端映射漂移。现有 `GameService.trainingLibraryNames` 中的映射迁移到该目录，单人训练和正式匹配共同使用。

## WebSocket 协议

### 开始匹配

```json
{
  "type": "match_start",
  "data": {
    "difficultyGroup": "junior",
    "difficultyLevel": "junior_7_1"
  }
}
```

主页入口固定发送：

```json
{
  "difficultyGroup": "rank",
  "difficultyLevel": "rank_current"
}
```

### 等待确认

```json
{
  "type": "match_waiting",
  "data": {
    "difficultyGroup": "junior",
    "difficultyLevel": "junior_7_1",
    "difficultyLabel": "初中英语 · 7年级上册"
  }
}
```

### 游戏开始

`game_start` 增加同样的三个难度字段，供前端展示和断线恢复使用。难度字段由后端当前匹配偏好生成，不回显未经校验的客户端文本。

## Redis 数据结构

### 分难度队列

每个难度使用独立 ZSet：

```text
match:queue:{difficultyLevel}
```

- member：用户 ID。
- score：用户正式段位，继续使用现有按段位差逐步扩展到 ±50 的候选规则。
- 由于队列键已按 `difficultyLevel` 隔离，候选搜索不会读取其他难度队列。

### 用户匹配偏好

```text
match:preference:{userId}
```

保存后端规范化后的 `difficultyGroup`、`difficultyLevel` 和 `difficultyLabel`。该数据用于幂等请求、取消匹配、返回主页、断线重连和过期清理。

### 活跃队列集合

```text
match:active_queues
```

用户加入难度队列时把 `difficultyLevel` 加入集合。调度任务只遍历活跃队列，队列为空后移除对应成员，避免每 500ms 扫描所有支持的难度，也禁止使用 Redis `KEYS` 扫描。

现有全局匹配锁继续保留，确保多实例下同一候选人不会被重复配对。

## 后端流程

### 加入队列

1. `GameChannelHandler` 从 `match_start.data` 读取 group/level。
2. 难度目录验证组合并返回规范化选择；非法值在状态转换前拒绝。
3. 状态从 `IDLE` CAS 到 `MATCHING`。
4. `MatchService` 写入用户匹配偏好、对应难度 ZSet 和活跃队列集合。
5. 后端推送包含权威难度信息的状态快照和 `match_waiting`。

同一用户在 `MATCHING` 状态再次发送相同难度时按幂等请求处理。若发送不同难度，则保留原队列并返回“请先取消当前匹配”，不允许用户同时存在于两个队列。

### 匹配成功

1. 调度器在同一难度队列中按现有段位范围寻找候选人。
2. 双方分别 CAS 从 `MATCHING` 到 `MATCHED`。
3. 复制双方一致的偏好形成不可变匹配上下文，并立即从 ZSet 删除双方队列成员，防止再次被调度器扫描；此时暂不删除偏好。
4. 创建房间并把规范化难度传给 `GameService.startGame`。
5. `GameService.startGame` 返回明确的成功或失败结果。成功后删除双方匹配偏好；失败后由匹配服务根据失败类型完成状态、房间和偏好清理。

匹配上下文必须在删除任何偏好前创建，并显式传给游戏服务，游戏服务不得重新从 Redis 猜测本局难度。

### 取消与断线

- 主动取消或返回主页：从偏好中解析准确队列键，删除 ZSet member、离线标记和偏好；队列为空时同步清理活跃队列集合。
- 断线 15 秒内：保留队列和偏好；重连后从偏好恢复原难度队列并返回对应 `match_waiting`。
- 断线超过宽限期、状态过期或候选人不合法：调度器清理队列、偏好和状态。
- 偏好缺失但状态仍为 `MATCHING`：视为脏状态，强制回到 `IDLE`，不猜测或回退到默认难度。

## 对局抽词

`GameService.startGame` 增加规范化匹配难度参数：

- `rank_current`：保留现有双方段位概率算法，从难度 Tier 抽取 10 个抢词单词。
- 大类难度（如 `junior`）：从该大类包含的全部教材词库随机抽取 10 个单词。
- 具体教材（如 `junior_7_1`）：只从该教材对应词库随机抽取 10 个单词。

正式匹配保持现有“不过滤用户已掌握单词”的行为。非段位难度抽词不足 10 个时，不从全库或其他难度补齐；回滚游戏启动、清理房间，将双方恢复到 `IDLE`，并推送“该难度词库暂时无法开始匹配”的错误。

游戏状态保存 `matchDifficultyGroup`、`matchDifficultyLevel` 和 `matchDifficultyLabel`，`game_resume` 同步返回这些字段。

## 对局记录

`game_record` 新增：

- `match_difficulty_group`
- `match_difficulty_level`

正式匹配结算时写入这两个字段；单人训练继续使用现有 `training_difficulty_group`、`training_difficulty_level`，两套语义不混用。记录列表为正式匹配记录显示难度标签。

## 错误处理

- 缺少、未知或 group/level 不匹配：拒绝请求，不进入 `MATCHING`。
- 匹配中请求切换难度：拒绝并提示先取消。
- 没有同难度对手：持续等待，不扩大难度范围。
- 词库不足：终止本次启动，不跨难度补词。
- 双方状态 CAS 部分失败：沿用现有回滚逻辑，并保留仍有效用户的原难度偏好以便重新排队。
- 房间或推送创建失败：`GameService.startGame` 返回失败结果；匹配服务回滚双方状态、房间和用户房间映射，并清理本次匹配上下文和偏好。

## 测试与验收

### 前端

- 主页匹配发送 `rank / rank_current`。
- 单人训练面板匹配发送当前选择。
- 等待弹窗展示后端难度标签。
- 匹配期间所有冲突按钮被禁用，取消后保留选中难度。

### WebSocket 与匹配服务

- 合法、缺失、未知和 group/level 不匹配参数。
- 相同难度重复请求幂等，不同难度重复请求被拒绝。
- 同一难度且段位范围合适的用户成功匹配。
- 不同难度用户永不匹配，即使段位完全相同。
- 取消、返回主页、断线宽限期、重连和过期清理均作用于正确队列。
- 活跃队列为空后移除，多个实例下全局锁仍生效。

### 游戏服务与记录

- `rank_current` 继续调用段位 Tier 算法。
- 大类和具体教材只从目录映射的词库抽词。
- 词库不足时启动失败且没有跨难度查询。
- `game_start`、`game_resume`、游戏状态和正式匹配记录包含一致的难度字段。

### 集成验收

- 两个浏览器选择相同教材后能够进入同一局，所有单词均属于该教材词库。
- 两个浏览器选择不同教材时持续等待，任何时长都不会互相匹配。
- 一方断线 15 秒内恢复到原难度等待；超过宽限期后被清理。
- 桌面和窄屏下新按钮、等待难度标签及取消入口均清晰可用。
