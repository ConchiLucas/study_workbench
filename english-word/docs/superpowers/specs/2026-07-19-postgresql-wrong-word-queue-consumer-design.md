# PostgreSQL 错词队列事件驱动处理器设计

## 目标

保留 PostgreSQL 作为错词事件的唯一事实来源，把当前“收到新错词时同步处理最老 3 条”的机制改成事件驱动处理器。新错词成功入库后立即唤醒处理器；旧失败批次只在后续新错词到达或服务启动恢复时再尝试，不设置按分钟执行的重试计时器。处理器必须在旧批次再次失败后继续处理后续错词，避免毒数据长期阻塞整个用户队列。

## 当前问题

当前 `WrongWordStrategyService.handle_event()` 在请求线程中同时完成事件插入、3 条事件领取和挖空题生成。领取条件固定为同一用户最早的 3 条 `pending` 记录。生成失败后，这 3 条又恢复为 `pending`，且没有重试次数、重试触发标记或最终失败状态。

因此，一批永久失败的数据会反复回到队首。当前 `my / adolescence / outpost` 批次返回 HTTP 400 后，后续 `cheaply / zipper / clearly` 即使已经凑满 3 条也无法被处理。

## 方案选择

采用 PostgreSQL 队列加 word-agent 事件驱动处理器，不引入 Redis、Celery、RabbitMQ 或新的独立任务中心依赖。

选择原因：

- `wrong_word_events` 已经保存了完整事件和状态，继续作为唯一事实来源可避免双写。
- `FOR UPDATE SKIP LOCKED` 已能支持多处理器实例并发领取。
- 当前吞吐量较低，不需要专用消息中间件。
- PostgreSQL 保存完整状态，服务重启后仍可恢复未完成任务。
- 新错词唤醒和启动恢复能处理新任务与旧失败任务，空闲时不周期查询数据库。

## 事件状态

保留现有 `pending`、`processing`、`processed`，新增：

- `retry_wait`：本次生成失败，等待下一条新错词或服务启动恢复触发重试。
- `failed`：连续失败达到上限，已隔离，不再自动领取。

状态流转：

```text
pending
   -> processing
      -> processed
      -> retry_wait -> processing
      -> failed
```

任何一个批次失败后，处理器都必须继续领取下一批可处理事件，不能结束本次唤醒处理。

## 表结构变更

在 `select_english_word.public.wrong_word_events` 增加：

- `retry_count int NOT NULL DEFAULT 0`
- `retry_after_event_id bigint NULL`
- `locked_at timestamp NULL`
- `last_attempt_at timestamp NULL`

复用现有字段：

- `error` 保存最近一次完整错误，包含 HTTP 状态码和受限长度的响应正文。
- `batch_key` 保存稳定的批次幂等键。
- `cloze_item_id` 保存成功生成的挖空题 ID。

索引增加：

```sql
CREATE INDEX ... ON wrong_word_events(user_id, status, retry_after_event_id, id);
CREATE INDEX ... ON wrong_word_events(status, locked_at);
```

现有 `(source, source_answer_detail_id)` 唯一约束继续保证同一答错明细不会重复入队。

## 事件接收

`POST /v1/wrong-words/events` 改为轻量入队：

1. 校验并插入事件。
2. 发生唯一键冲突时返回已有事件 ID。
3. 提交事务后通知应用内处理器有新事件可处理。
4. 返回当前用户待处理数量。
5. 不在 HTTP 请求内调用 Java 造句接口。

Java 当前不依赖响应里的同步生成结果，因此接口可以保持现有响应结构，并将 `generated` 固定为 `false`。唤醒动作只设置进程内事件，不等待 LLM、TTS 或 MinIO，错词通知请求可以立即返回。

## 事件驱动处理器

word-agent 启动时创建一个应用级异步处理器。处理器平时阻塞等待，不周期查询数据库；以下两类事件会唤醒它：

- 新错词事务提交后设置应用内唤醒事件。
- 应用启动完成后主动唤醒一次，用于恢复服务停止期间积累的任务。

配置项包括：

- 每批事件数：3
- 每次唤醒最大处理批次数：10
- 最大重试次数：3
- 单批外部调用硬超时与 `processing` 超时：10 分钟

每次唤醒先恢复超时的 `processing`，再按用户处理本次可重试的旧失败批次，最后处理已经凑满 3 条的全新 `pending` 事件。任何批次都不能混入其他用户的错词。旧批次再次失败后不能在同一次唤醒中反复领取，必须等待该用户出现更晚的新事件，或下一次服务启动恢复。

每个失败批次写入 `retry_after_event_id`，记录本次失败时该用户最大的事件 ID。正常运行时，只有存在 `id > retry_after_event_id` 的同用户新事件时，该批次才可再次领取。重试失败后把水位更新为当时最新事件 ID，因此一次新错词唤醒最多推动同一个旧批次重试一次，不会形成忙循环。

每次最多处理 10 个批次。达到上限时，如果仍有全新 `pending` 批次，处理器主动再次设置唤醒事件并让出执行权；仅剩刚失败且尚无更新事件的 `retry_wait` 时则回到等待状态。

唤醒事件按“等待信号、清除信号、排空当前可处理数据、检查信号、重新等待”的顺序处理。新事件如果在排空期间到达，会把信号重新设为已触发；处理器返回等待时会立即再次运行，因此不会丢失唤醒。

每次领取使用事务和行锁：

```sql
SELECT ...
FROM wrong_word_events
WHERE user_id = :user_id
  AND status = 'pending'
ORDER BY id
LIMIT 3
FOR UPDATE SKIP LOCKED;
```

旧失败批次按稳定 `batch_key` 整批领取，并额外要求：

```sql
status = 'retry_wait'
AND retry_count < 3
AND EXISTS (
  SELECT 1
  FROM wrong_word_events newer
  WHERE newer.user_id = wrong_word_events.user_id
    AND newer.id > wrong_word_events.retry_after_event_id
)
```

应用启动恢复是明确例外：每个 `retry_wait` 批次允许尝试一次，不要求存在更晚事件；如果仍失败，同样更新事件 ID 水位并回到等待状态。启动时先记录需要恢复的稳定 `batch_key` 集合，超过单次 10 批上限时主动继续下一次处理，直到这个启动集合全部尝试一次；刚失败重新进入 `retry_wait` 的批次不会再次加入本次启动集合。

领取满 3 条后立即更新为 `processing`，写入 `locked_at`、`last_attempt_at` 和稳定的 `batch_key`，然后提交领取事务。外部 LLM、TTS、MinIO 调用不占用数据库事务和行锁。

处理器处理完一个批次后继续领取下一批。失败批次进入 `retry_wait` 或 `failed` 后不再参与当前可领取集合，因此不会挡住后面的事件。

## 新事件触发重试与失败隔离

批次失败时，三条事件作为一个不可拆分批次处理：

- 第一次失败：`retry_count=1`，状态为 `retry_wait`，记录当前用户最大事件 ID。
- 后续出现新错词或服务重新启动时再尝试一次；第二次失败后 `retry_count=2`，继续等待下一次触发。
- 第三次失败：`retry_count=3`，状态为 `failed`，停止自动重试。

三条事件保留原始数据、稳定 `batch_key` 和完整错误，不删除。没有新错词且服务不重启时，`retry_wait` 会一直等待，不产生查询和外部调用。后续可以人工检查并重新入队，但本次不新增管理页面。

HTTP 错误记录必须同时包含：

- 请求 URL
- HTTP 状态码
- 响应正文摘要
- 批次事件 ID 和单词

响应正文应限制长度并过滤凭据，避免敏感信息进入数据库。

## 卡死恢复

每个批次的 Java 造句、TTS 和 MinIO 链路整体受 10 分钟硬超时保护；超时后由当前处理器立即按失败重试规则释放该批次。处理器还会在应用启动时以及每次被真实事件唤醒后，将 `processing` 且 `locked_at` 早于 10 分钟前的事件恢复为 `retry_wait`：

- `retry_count` 增加 1。
- 未达到 3 次时设置为 `retry_wait`，记录当前用户最大事件 ID。
- 达到 3 次时转为 `failed`。
- `error` 记录“处理超时或处理器异常退出”。

这样 word-agent 在造句、TTS、上传或状态回写过程中崩溃后，任务不会永久停留在 `processing`。运行期间如果处理任务自身异常退出，生命周期管理器必须记录错误并立即重建处理器；这次重建也执行恢复检查。因此不需要为卡死恢复增加固定周期扫描。

## 幂等保护

批次幂等键根据排序后的事件 ID 确定，例如：

```text
wrong-word-events:47-48-49
```

Java 的外部挖空生成请求增加 `generationKey`，`sentence_cloze_item` 增加可空的 `generation_key` 和唯一索引。处理逻辑为：

1. 收到 `generationKey` 后先查询已有题目。
2. 已存在则直接返回原 `cloze_item_id`。
3. 不存在才执行造句、TTS、MinIO 和插入。
4. 并发插入发生唯一键冲突时重新查询并返回已有题目。

这可以防止“挖空题已经插入，但 word-agent 在标记 `processed` 前崩溃”造成重复题；正常重试会在调用造句前命中已有题目，因此也不会重复生成音频。极低概率的并发竞态仍由数据库唯一约束保证只保留一条题目记录，队列的 `processing` 状态与行锁负责避免正常处理器走入该竞态。

## 现有数据迁移

迁移不删除任何事件：

- `pending` 且 `error IS NULL`：保持 `pending`、`retry_count=0`。
- `pending` 且已有错误：改为 `retry_wait`、`retry_count=1`，`retry_after_event_id` 设为该用户当前最大事件 ID。
- 超时的历史 `processing`：按卡死恢复规则处理。
- 现有 `processed`：保持不变。

部署启动恢复时，`my / adolescence / outpost` 会尝试一次；仍失败则等待下一条新错词触发后续尝试。处理器随后处理 `cheaply / zipper / clearly`，不再被前一批阻塞。

## 服务生命周期

后台处理器跟随 FastAPI/word-agent 生命周期启动和停止：

- 应用启动完成后创建异步处理器，执行一次恢复扫描并处理已经满足条件的任务。
- 新错词事务提交后，通过进程内 `asyncio.Event` 唤醒处理器。
- 没有新错词时处理器持续等待，不创建重试计时器，也不使用固定间隔轮询。
- 应用关闭时通知处理器停止，并等待当前批次完成到安全边界。
- 多实例同时运行时依靠数据库状态更新和 `FOR UPDATE SKIP LOCKED` 避免重复领取。
- 单个批次异常必须在处理器内部捕获，不能导致整个处理任务退出。

进程内唤醒只负责降低延迟，PostgreSQL 状态才是可靠来源。即使服务在“提交事件”和“发出唤醒”之间退出，重启恢复扫描也会重新发现任务；不会因为内存信号丢失而丢数据。

## 可观测性

每个批次记录结构化日志：

- `batch_key`
- 事件 ID 和单词
- 尝试次数
- 开始和结束时间
- 成功生成的 `cloze_item_id`
- 失败状态、`retry_after_event_id` 和错误摘要

健康检查继续表示 HTTP 服务是否可用；队列状态通过 PostgreSQL 统计确认，不把“存在 failed 任务”误报为 HTTP 服务不健康。

## 测试

word-agent 测试覆盖：

- 事件接口只入队，不同步生成。
- 新事件提交后会立即唤醒处理器，无需等待轮询间隔。
- 空闲期间不会重复执行队列扫描 SQL。
- 满 3 条才领取；不足 3 条保持等待。
- `retry_wait` 在没有更新事件时不会被领取。
- 新事件到达后，旧失败批次最多重试一次。
- 同一次唤醒中再次失败的旧批次不会形成忙循环。
- 第三次失败转为 `failed`。
- 第一批失败后同一次唤醒继续处理第二批。
- 超时 `processing` 能恢复，并遵守最大重试次数。
- 服务启动时能恢复停机期间遗留的 `pending`、`retry_wait` 和超时 `processing`，并且每个旧失败批次最多尝试一次。
- 两个处理器实例并发时同一事件不会被重复领取。
- HTTP 错误保存状态码和响应正文摘要。
- 单批异常不会让事件驱动处理器退出。

Java 测试覆盖：

- 相同 `generationKey` 重复请求只产生一条 `sentence_cloze_item`。
- 并发唯一键冲突后返回已有题目。
- 首次成功仍要求有效句子、TTS 和 MinIO URL。

数据库迁移验证：

- 现有 6 条 `pending` 全部保留。
- 历史错误批次进入 `retry_wait`。
- 后续正常批次能在毒批次隔离前后继续处理。
- 不产生重复 `generation_key` 或重复题目。

## 发布顺序

1. 执行 PostgreSQL 向前兼容字段和索引迁移。
2. 部署支持 `generationKey` 幂等的 Java 后端。
3. 部署轻量入队接口和事件驱动处理器的 word-agent。
4. 检查现有事件迁移数量与状态。
5. 通过新的错词事件验证旧失败批次重试和失败隔离结果。
6. 确认后续正常批次生成成功，且 7003 能读取新题。

不需要修改 7002 或 7003 前端，也不需要引入 Redis 队列、固定周期定时任务或重试计时器。
