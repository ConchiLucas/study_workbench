# 按难度精确匹配实施计划

> **执行要求：** 按任务顺序实施；每个行为先写失败测试，确认失败原因正确，再写最小实现并让测试通过。

**目标：** 在“单人训练”入口增加正式匹配功能，使双方只有选择完全相同的难度时才能匹配，并使正式对局只从该难度对应的词库取词。

**架构：** 后端建立唯一的难度目录，WebSocket 层只接受目录中合法的 `difficultyGroup + difficultyLevel`。匹配服务把用户放入 `match:queue:{difficultyLevel}`，同时保存用户匹配偏好；调度器只在同一个队列内寻找段位差合适的对手。匹配成功后把不可变难度上下文交给游戏服务，游戏服务按该难度对应词库抽取 10 个单词，词数不足则整场失败且不跨难度补词。

**技术栈：** Java 17、Spring Boot、Netty WebSocket、Spring Data Redis、MyBatis-Plus、JUnit 5、Mockito、Vue 3、Pinia、Vitest、TypeScript。

---

## 任务 1：建立后端统一难度目录

**文件：**

- 新建：`rob_english_word_back/src/main/java/com/robword/service/TrainingDifficultyCatalog.java`
- 新建：`rob_english_word_back/src/test/java/com/robword/service/TrainingDifficultyCatalogTest.java`
- 修改：`rob_english_word_back/src/main/java/com/robword/service/GameService.java`

### 1.1 先写失败测试

测试 `rank/rank_current`、父级难度、具体子级、跨分组组合和未知值：

```java
@Test
void resolvesExactJuniorChildLibraries() {
    var difficulty = catalog.resolve("junior", "junior_7_1").orElseThrow();
    assertThat(difficulty.libraryNames()).containsExactly("PEPChuZhong7_1");
}

@Test
void resolvesJuniorGroupToAllJuniorLibraries() {
    var difficulty = catalog.resolve("junior", "junior").orElseThrow();
    assertThat(difficulty.libraryNames()).containsExactly(
        "PEPChuZhong7_1", "PEPChuZhong7_2", "PEPChuZhong8_1",
        "PEPChuZhong8_2", "PEPChuZhong9_1"
    );
}

@Test
void rejectsGroupAndLevelFromDifferentBranches() {
    assertThat(catalog.resolve("junior", "senior_1")).isEmpty();
}
```

运行 `mvn -Dtest=TrainingDifficultyCatalogTest test`，预期先因类不存在而失败。

### 1.2 实现目录和值对象

```java
@Component
public class TrainingDifficultyCatalog {
    public record Difficulty(
        String group,
        String level,
        String label,
        List<String> libraryNames,
        boolean rankBased
    ) {}

    public Optional<Difficulty> resolve(String group, String level) {
        if (group == null || level == null) return Optional.empty();
        Difficulty value = difficulties.get(level);
        return value != null && value.group().equals(group)
            ? Optional.of(value)
            : Optional.empty();
    }
}
```

将 `GameService.trainingLibraryNames` 中现有全部映射迁入目录，包括小学、初中、高中、四六级、考研、商务/留学、专四专八、GRE/SAT 和父级汇总项。`rank_current` 标记为 `rankBased=true`。然后让原单人训练复用此目录并删除重复映射。

### 1.3 验证

```bash
cd rob_english_word_back
mvn -Dtest=TrainingDifficultyCatalogTest,GameServiceTest test
```

---

## 任务 2：定义并校验 WebSocket 匹配协议

**文件：**

- 修改：`rob_english_word_back/src/main/java/com/robword/netty/GameChannelHandler.java`
- 修改：`rob_english_word_back/src/test/java/com/robword/netty/GameChannelHandlerTest.java`
- 修改：`rob_english_word_back/src/test/java/com/robword/netty/GameChannelHandlerReconnectTest.java`

### 2.1 先写失败测试

覆盖合法 payload、缺字段、跨分组、未知值、同难度幂等、等待中改难度被拒绝，以及重连快照带规范难度：

```java
@Test
void matchStartPassesCanonicalDifficultyToMatchService() {
    JsonNode data = objectMapper.readTree("""
        {"difficultyGroup":"junior","difficultyLevel":"junior_7_1"}
        """);
    handler.handleMatchStart(USER_ID, data);
    verify(matchService).startMatching(USER_ID, "junior", "junior_7_1");
}

@Test
void invalidMatchDifficultyIsRejectedBeforeStateTransition() {
    JsonNode data = objectMapper.readTree("""
        {"difficultyGroup":"junior","difficultyLevel":"senior_1"}
        """);
    handler.handleMatchStart(USER_ID, data);
    verify(userStateManager, never()).compareAndSetState(anyLong(), any(), any());
}
```

### 2.2 实现协议入口

将分发改为 `case "match_start" -> handleMatchStart(userId, data)`。先通过难度目录校验，再切换状态。等待响应统一返回：

```json
{
  "type": "match_waiting",
  "data": {
    "difficultyGroup": "junior",
    "difficultyLevel": "junior_7_1",
    "difficultyLabel": "初中英语七年级上册"
  }
}
```

运行 `mvn -Dtest=GameChannelHandlerTest,GameChannelHandlerReconnectTest test`。

---

## 任务 3：把 Redis 匹配队列按难度拆分

**文件：**

- 修改：`rob_english_word_back/src/main/java/com/robword/service/MatchService.java`
- 修改：`rob_english_word_back/src/test/java/com/robword/service/MatchServiceTest.java`

### 3.1 先写失败测试

```java
@Test
void usersWithSameDifficultyCanMatch() {
    addMatchingUser(1L, 10, "junior", "junior_7_1");
    addMatchingUser(2L, 15, "junior", "junior_7_1");
    matchService.checkMatchQueue();
    verify(gameService).startGame(eq(1L), eq(2L), any(),
        argThat(d -> d.difficultyLevel().equals("junior_7_1")));
}

@Test
void usersWithDifferentDifficultyNeverMatch() {
    addMatchingUser(1L, 10, "junior", "junior_7_1");
    addMatchingUser(2L, 10, "junior", "junior_7_2");
    matchService.checkMatchQueue();
    verifyNoInteractions(gameService);
}
```

另测：重连回原队列；MATCHING 但无 preference 时重置 IDLE；取消/超时只清理准确队列。

### 3.2 实现键和值对象

```java
private static final String QUEUE_PREFIX = "match:queue:";
private static final String PREFERENCE_PREFIX = "match:preference:";
private static final String ACTIVE_QUEUES_KEY = "match:active_queues";

public record MatchPreference(
    String difficultyGroup,
    String difficultyLevel,
    String difficultyLabel,
    List<String> libraryNames,
    boolean rankBased
) {}
```

`startMatching` 保存规范 preference、加入 `match:queue:{level}`，并把 level 加入 active set。偏好只持久化 group/level，读取时由目录恢复其他字段。

### 3.3 修改调度和清理

调度器继续用全局锁，遍历 active set，每次只在单一难度队列内使用现有段位差 `0..50` 搜索。匹配成功后：

1. 两人 CAS `MATCHING -> MATCHED`。
2. 复制不可变难度上下文。
3. 立即移除双方队列成员。
4. 调用游戏服务。
5. 成功删除 preference；失败清理房间、状态、队列、preference 并通知双方。

断线 15 秒内按 preference 回原队列；超时准确清理，不使用 `KEYS`。

### 3.4 验证

```bash
cd rob_english_word_back
mvn -Dtest=MatchServiceTest,GameChannelHandlerReconnectTest test
```

---

## 任务 4：正式对局按所选难度抽词

**文件：**

- 修改：`rob_english_word_back/src/main/java/com/robword/service/GameService.java`
- 修改：`rob_english_word_back/src/main/java/com/robword/service/WordService.java`
- 修改：`rob_english_word_back/src/test/java/com/robword/service/GameServiceTest.java`

### 4.1 先写失败测试

```java
@Test
void formalMatchUsesOnlySelectedDifficultyLibraries() {
    when(wordService.getRandomWordsForTrainingLibraries(
        List.of("PEPChuZhong7_1"), 10)).thenReturn(tenWords());

    GameStartResult result = gameService.startGame(1L, 2L, room, juniorSevenFirstSemester());

    assertThat(result.success()).isTrue();
    verify(wordService).getRandomWordsForTrainingLibraries(
        List.of("PEPChuZhong7_1"), 10);
    verify(wordService, never()).getRandomWordsForMatch(anyInt(), anyInt(), anyInt());
}

@Test
void insufficientSelectedDifficultyWordsFailsWithoutFallback() {
    when(wordService.getRandomWordsForTrainingLibraries(anyList(), eq(10)))
        .thenReturn(nineWords());
    GameStartResult result = gameService.startGame(1L, 2L, room, juniorSevenFirstSemester());
    assertThat(result.success()).isFalse();
    verify(wordService, never()).getRandomWordsForMatch(anyInt(), anyInt(), anyInt());
}
```

同时测试 `rank_current` 保持原段位算法。

### 4.2 返回明确启动结果并写入游戏状态

```java
public record GameStartResult(boolean success, String message) {
    public static GameStartResult ok() { return new GameStartResult(true, ""); }
    public static GameStartResult failed(String message) {
        return new GameStartResult(false, message);
    }
}
```

`startGame` 接收 preference。非段位难度调用 `getRandomWordsForTrainingLibraries(libraryNames, 10)`，必须恰好 10 个，不足时失败且不回退。游戏状态及 `game_start/game_resume` 增加 `matchDifficultyGroup/Level/Label`。

### 4.3 验证

运行 `mvn -Dtest=GameServiceTest,MatchServiceTest test`。

---

## 任务 5：保存和展示正式匹配难度

**文件：**

- 修改：`rob_english_word_back/db/game_record.sql`
- 修改：`rob_english_word_back/src/main/java/com/robword/entity/GameRecord.java`
- 修改：`rob_english_word_back/src/main/java/com/robword/service/GameSettlementService.java`
- 修改：`rob_english_word_back/src/test/java/com/robword/service/GameSettlementServiceTest.java`
- 修改：`rob_english_word_front/src/views/RecordView.vue`

### 5.1 先写失败测试

结算测试断言正式对局复制 `matchDifficultyGroup/Level/Label`，训练记录仍只写 training 字段。

### 5.2 增加字段

```sql
ALTER TABLE game_record
    ADD COLUMN IF NOT EXISTS match_difficulty_group VARCHAR(64),
    ADD COLUMN IF NOT EXISTS match_difficulty_level VARCHAR(64),
    ADD COLUMN IF NOT EXISTS match_difficulty_label VARCHAR(128);
```

实体增加三个驼峰字段。记录页正式对局优先显示 `matchDifficultyLabel`，旧记录按原文案降级。

### 5.3 验证

运行 `mvn -Dtest=GameSettlementServiceTest test`。

---

## 任务 6：在单人训练面板增加“开始匹配”

**文件：**

- 修改：`rob_english_word_front/src/views/HomeView.vue`
- 新建：`rob_english_word_front/src/views/HomeView.matchmaking.test.ts`

### 6.1 先写失败测试

```ts
it('sends selected difficulty when matching from training panel', () => {
  expect(source).toContain('difficultyGroup: selectedDifficulty.value.parentKey')
  expect(source).toContain('difficultyLevel: selectedDifficulty.value.key')
  expect(source).toContain("wsStore.send('match_start'")
})

it('renders match button between difficulty and solo training', () => {
  const difficulty = source.indexOf('难度选择')
  const match = source.indexOf('开始匹配')
  const solo = source.indexOf('开始训练')
  expect(difficulty).toBeLessThan(match)
  expect(match).toBeLessThan(solo)
})
```

另测等待弹窗展示后端 `difficultyLabel`，等待期间禁用其他动作。

### 6.2 实现两个入口

主界面发送：

```ts
wsStore.send('match_start', {
  difficultyGroup: 'rank',
  difficultyLevel: 'rank_current',
})
```

单人训练面板发送当前选择：

```ts
wsStore.send('match_start', {
  difficultyGroup: selectedDifficulty.value.parentKey,
  difficultyLevel: selectedDifficulty.value.key,
})
```

按钮顺序为：难度选择、开始匹配、开始训练、查看答题结果、已掌握单词。等待弹窗显示“正在匹配：{difficultyLabel}”。取消不自动降级，也不使用机器人。

### 6.3 验证

```bash
cd rob_english_word_front
npm test -- --run src/views/HomeView.matchmaking.test.ts
npm run build
```

---

## 任务 7：全量回归与运行验证

### 7.1 自动验证

```bash
cd rob_english_word_back
mvn test

cd ../rob_english_word_front
npm test -- --run
npm run build
```

### 7.2 数据库迁移

对当前开发库执行幂等 `ALTER TABLE`，查询 `information_schema.columns` 确认三个字段存在。

### 7.3 双用户验收

1. A、B 都选七年级上册：匹配成功，10 词全部来自 `PEPChuZhong7_1`。
2. A 选七年级上册、B 选七年级下册：持续等待，不能互相匹配。
3. 等待中 15 秒内重连：回到原难度队列。
4. 取消：队列成员和 preference 都删除。
5. 测试词数不足：明确失败，绝不跨难度补词。

### 7.4 范围检查

```bash
git status --short
git diff --check
git diff -- rob_english_word_back rob_english_word_front docs/superpowers
```

只处理本功能文件，不覆盖或暂存工作区内其他已有改动。
