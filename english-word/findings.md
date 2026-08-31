# 调查结论

- “去重单词表”页面总数由 `SELECT COUNT(*) FROM word_clean` 得到。
- 实时数据库中 `word_clean` 与 `word_clean_best_sentence` 均为 22,415 条。
- `word_clean_best_sentence.word_clean_id` 无重复、无孤儿、无缺失。
- `word_clean_best_sentence` 自带 `word` 字段，可直接按标准化单词关联。
- 原始 `word` 表为 153,009 条，标准化单词为 23,882 个；跨词库重复是合法数据。
- `word_clean` 已有 `lower(word)` 唯一索引；本次需要为 `word_clean_best_sentence` 增加 `lower(word)` 唯一索引。
- `word_clean` 与最佳例句表的 22,415 个单词逐字完全一致。
- 原始 `word` 表使用等号关联可匹配 150,927 行，使用 `lower(word)` 可匹配 151,222 行；295 行只存在大小写差异。
- `btrim(word)` 对当前数据覆盖率没有任何提升，因此关联表达式不使用 `btrim()`。
- 295 条大小写差异中仅 45 条原始含义与 `word_clean.meaning` 完全一致；250 条不同，涉及 129 个标准化键。
- 大小写可区分不同词义，例如 `May/may`、`March/march`、`China/china`、`Turkey/turkey`、`US/us`；使用 `lower(word)` 会产生错误例句关联。
- `word_clean_best_sentence.meaning` 当前为空，语义核对必须通过 `word_clean_id` 使用 `word_clean.meaning`。
- `word.id=78494` 是唯一的小写 `april`，含义与最佳例句侧均为“四月”，且无同词库冲突，已安全修正为 `April`。
- 额外修正了 44 条释义完全一致的数据，以及 120 条人工核验同义且无同词库冲突的数据。
- 剩余 130 条大小写差异因语义不同或同词库冲突而保留。
- `Metro`、`right`、`thanksgiving` 的最佳例句与 `word_clean.meaning` 不符，已删除 3 条最佳例句及级联的 3 条旧 TTS 任务；每个词的 5 条候选句和 `word_clean` 均保留。
- 用户随后要求彻底移除这三个精确拼写：已从 `word` 删除 23 行、从 `word_clean` 删除 3 行，并级联删除 15 条候选句和 15 条旧造句任务。
- 大小写不同且含义不同的 `metro` 1行、`Right` 2行、`Thanksgiving` 5行保留。
- 最终 `word_clean` 与 `word_clean_best_sentence` 均为 22,412 条，无缺失、无孤儿。
- 最佳例句词形初扫：20,863 条精确使用目标词；486 条是正常句首大写；122 条是非句首大小写变化；877 条是常规词形变化；64 条需要进一步核验。
- 64 条中发现多类明确拼写错误（如 `ajustment/datebase/eletricity/reservior`）、占位搭配写法、不规则变化和未独立使用目标词等不同原因，不能采用统一替换。
- 词库列表接口位于 `word_select_dashboard/server/api/v1/system/sys_word_library.go` 的 `Words`。
- 已删除 23 个明确错误/重复的基础词条，另删除 14 个低分异常词条；删除均使用精确拼写和数量守卫。
- 已跨 `word`、`word_clean`、候选句、任务表和最佳例句表同步修正 `agro-scientifc/Montgommery/bullrush` 以及 `loder/cando/Lowa/pleasureable` 七组拼写。
- `loader`、`condo`、`Iowa`、`pleasurable` 的词条与例句已同时修正，原 TTS 任务已删除，最佳例句 TTS 状态重置为 `pending`。
- 低分语义复核发现 `secureness` 的基础释义为“停止工作”，最佳例句却表达“安全感”；已按 `word_clean` 为准删除该最佳例句及 TTS 任务，保留 5 条候选句。
- 用户明确要求最佳例句表也保留记录后，已从各词现有最高分候选句恢复 7 条最佳例句记录，评分说明加上“待重新造句”标记，TTS 状态为 `pending`。
- 最终数据为 `word_clean=22,375`、`word_clean_best_sentence=22,375`；单词、释义、来源词条、来源句子和翻译差异均为 0，重复和孤儿均为 0。
- 已保留但待任务中心重新生成的 7 个词是 `approaching`、`lumiere`、`macdonald`、`maker`、`secureness`、`socks`、`yard`。
- 最终全量词法扫描的 25 条特殊命中均为合法的连字号复合词、所有格、副词或过去式等，无新的错误记录。
- 已人工产生并校验 `approaching`、`lumiere`、`maker`、`secureness`、`socks`、`yard` 及修正后的 `McDonald's` 七条最佳例句，同步更新来源候选句。
- 上游数据中 `macdonald` 释义指向麦当劳品牌，已按品牌标准拼写修正为 `McDonald's`；`secureness` 原释义“停止工作”与英文词义冲突，已修正为“安全性；稳固；安全感”。
- 七条例句已调用小米 MiMo `mimo-v2.5-tts` / `Chloe` 生成 WAV，时长 3.68–6.24 秒；最佳例句与 TTS 任务均为 `success`。
- 音频文件由 word-agent 保存，可播放 URL 使用 `http://127.0.0.1:8010/v1/tts/files/<file>`；七条均已验证返回 `HTTP 200 audio/wav` 且 WAV 文件头、大小、时长与数据库一致。
- 已在 `word_clean(word)` 和 `word_clean_best_sentence(word)` 创建区分大小写的唯一索引，两个索引均为 unique/valid/ready。
- “词库单词管理” `Words` 接口已改为 `LEFT JOIN word_clean_best_sentence wcbs ON wcbs.word = w.word`，例句和翻译字段均来自 `wcbs`，关键词例句搜索也同步切换。
- 原始 `word` 表 152,933 行，精确左连接后仍为 152,933 行，没有放大；151,016 行命中最佳例句，其余未命中行返回空例句。
- 已在词库单词列表例句列增加播放/停止按钮；有成功 TTS 和 object URL 时启用，无音频时禁用。
- 实时数据中 22,363 条成功 TTS 均有 URL，其中 43 条使用 MinIO `/ai-file-navigation/...` 相对路径。
- 真实页面已分别验证普通 TTS 地址和 MinIO 地址：按钮能进入播放状态，音频结束自动复位，控制台无错误。
- 已将全部 22,363 条成功 TTS 统一到 MinIO `ai-file-navigation/word_clean_tts/`；其中 43 条原已在 MinIO，本次新上传并切换数据库 22,320 条。
- MinIO 与数据库的对象集均为 22,363 个，总大小均为 5,831,816,612 字节；缺失、多余、大小不符和本地 URL 均为 0。
- `word_clean_best_sentence` 及现存成功 `word_clean_sentence_tts_job` 记录均已更新 bucket、object key 和 `/{bucket}/{object_key}` URL。
- 迁移后从不同 ID 区间抽样 7 个对象，均经 8009 Go MinIO 代理返回 `HTTP 200 audio/wav`；再次 dry-run 为 0 条待迁移。
- 当前任务中心新生成 TTS 的实现仍会先写本地文件并回填 `19186` URL；本次完成的是现有数据全量迁移，未修改仓库外的任务中心生成逻辑。
## TTS 本地文件删除结论（2026-07-18）

- 现有成功 TTS 数据已全部指向 MinIO；删除本地 WAV 后播放抽样仍正常。
- 本地共删除 22,462 个 WAV 文件，仅保留 MinIO 中的对象副本。
- 两个本地 TTS 目录仍存在但为空，未清理目录内的非 WAV 内容。

## 2026-07-18 人名与地名扫描

- 当前 `word_clean` 为 22,375 条唯一词。
- 大写开头候选 576 条；其中混有国家、国籍、月份、机构、宗教、缩写和普通专名。
- 中文释义命中人名/地名关键词的记录约 255 条，但关键词宽松匹配会把大量普通词尾部的罕见姓氏释义误纳入。
- 小写候选中可信的人名词主要是 `elliot/hamlin/harry/lesley/roger/smith/tom/weldon` 等，可信地名主要是 `brea/chicopee/delft/lomas/mecca/ogallala/quincy/scandinavia/shanghai/welland/zagros` 等。
- 最终清单采用保守人工复核口径，不把 `and/big/given/rich` 这类普通词仅因附带生僻人名义项而列入。
- 最终 CSV 279 行：地名 142、人名 137；其中 27 行为中置信度歧义项。
- `Atlanta/Napier/Paris` 同时具有人名和地名义，按类别分别保留两行。
- 国家关键词复核为 0，CSV 与 `word_clean`、`word_clean_best_sentence` 的 ID、单词、释义、例句和翻译逐行差异均为 0。

## 2026-07-18 全库删除人名与地名

- 按 276 个区分大小写的唯一目标词完成全库删除；不对其他词条的句子正文进行模糊文本删除。
- 显式删除 `word` 486 行、`word_clean` 276 行、`bnc_words` 4 行、`game_answer_detail` 4 行、挖空题 2 行、挖空答案 1 行、复习计划 1 行。
- `word_clean` 外键级联删除候选句 1,380 行、造句任务 1,380 行、TTS 任务 274 行、最佳例句 276 行；评分任务为 0 行。
- 最终在基础词、去重词、BNC、候选句、各任务表、最佳例句、练习表的文本字段与 ID 字段中均无目标残留。
- 删除后 `word_clean` 与 `word_clean_best_sentence` 均为 22,099 行，候选句与造句任务均为 110,495 行；所有相关孤儿计数为 0。
- 本次受影响的 53 个词库缓存计数全部与实际行数一致。另有 4 个未受影响词库的历史计数差异，未擅自扩大本次修改范围。
- 压缩备份为 `outputs/person_place_deletion_backup_20260718.json.gz`，逐词审计为 `outputs/person_place_deletion_audit_20260718.csv`（276 行数据）。

## 2026-07-18 三词挖空题 TTS 与 MinIO

- 三词错题链路为 `WrongWordStrategyService -> Java SentenceClozeService -> word-agent /v1/sentences/generate`；原实现没有 TTS 或 MinIO 编排。
- `sentence_cloze_item` 和练习前端已经具备 `sentence_audio_url`/`sentenceAudioUrl` 支持，因此无需新增数据库列或修改前端。
- 新增 `MinIOStorage` 负责 bucket 校验、唯一 object key、上传及远端大小校验；URL 格式为 `/{bucket}/{object_key}`，可复用现有代理播放。
- Java 在插入前强制校验音频 URL，保证新生成题目不会出现只有句子、没有语音的半成品。
- 当前 word-agent `.env` 的 MinIO 配置有效，`ai-file-navigation` bucket 存在，运行时 object key 前缀为 `sentence_cloze_tts`。
- 全目录 Ruff 仍会报告 42 个本次之前已存在的长行问题，集中在 `llm_client.py`、`word_clean_sentence_score.py` 和 `wrong_word_strategy.py`；为避免覆盖用户现有改动，本次仅校验涉及文件且结果通过。

## 2026-07-19 按 AGENTS.md 启动项目

- 根 `AGENTS.md` 要求通过 ctx 链路读取“子项目总览”和“项目链路流转”，再依据源码和启动脚本启动服务。
- ctx 规则接口 `127.0.0.1:49173` 当前拒绝连接；环境中也没有 `ctx` 命令、`CTX` 或 `SESSION_ID`。因此暂时无法按 ctx 读取链路文档，先从不属于文档链路的启动脚本和运行态信息恢复项目。
- 全项目规范入口是 `./restart_all_services.sh restart`，目标端口为 7001/7002/7003/8009/8010/8019/9091。
- 首次启动在依赖检查阶段因 `mvn` 不在 PATH 中止；项目没有 Maven Wrapper，Homebrew Maven 未安装，但 IntelliJ 内置 Maven 3.9.16 可正常运行，Java 为 GraalVM 21.0.2。
- Docker 中 PostgreSQL 5432、Redis 6379 和 MinIO 19100 均已运行，无需重建数据依赖。
- 以 IntelliJ Maven 临时补入 PATH 后，统一脚本成功启动全部 6 个应用进程，7 个目标端口全部就绪。
- 三个前端首页 7001/7002/7003 均返回 HTTP 200；word-agent `/health` 返回 200；Go API 根路径返回预期 404，但业务轮询 `/executions/runs` 已在日志中持续返回 200；Java actuator 健康端点受安全策略保护返回 403，但业务接口、数据库查询和 WebSocket 鉴权连接均正常。
- 当前日志中可见的旧 403/502、Babel 和 WebSocket 拒绝记录来自本次重启前；本次启动段均显示应用就绪，没有新的启动失败。

## 2026-07-19 难度选择关闭层级修复

- `HomeView.vue` 使用 `showTrainingSetup` 控制难度训练全屏页，使用 `showDifficultyPicker` 在其上再覆盖难度选择全屏页。
- 难度选择叉号调用 `closeDifficultyPicker()`，当前只执行 `showDifficultyPicker.value = false`；底层 `showTrainingSetup` 仍为 true，因此第一次点击后显示训练设置页及第二个叉号。
- 第二次叉号调用 `closeTrainingSetup()` 才把 `showTrainingSetup` 设为 false，回到用户截图中的首页。
- 现有 `fullscreenNavigation.contract.test.ts` 只检查两个关闭函数和两个按钮存在，没有验证一次点击后的最终状态，因此未捕获该回归。
- 旧设计文档明确把难度选择、今日答题结果和已掌握单词的关闭目标设为训练设置页；用户本次反馈覆盖该交互约定，要求显式关闭时一次退出训练工作流。
- `TrainingAnswerResultsView.closePage()` 与 `MasteredWordsView.closePage()` 均通过 `state: { openTrainingSetup: true }` 返回主页，随后 `HomeView.onMounted()` 会重新显示训练设置页，形成同样的第二个叉号。
- 错题详情、记录卡片详情和记录答题详情关闭后返回各自列表，是有意义的详情层级；列表页本身关闭到主页，不属于本次缺陷。
- 7002 自动化浏览器只能访问独立的应用内浏览器会话，该会话没有用户在 Chrome 中的登录状态；访问 `/home` 显示登录表单，无法在不请求账号凭据的情况下继续真实点击验收。

## 2026-07-19 单人训练按钮文案

- 目标按钮位于 `HomeView.vue` 难度训练设置操作区，默认态原为“开始训练”，等待态为“训练准备中...”。
- 本次仅把默认态改为“单人训练”；旧连接提示语中的“重新点击开始训练”不是按钮文案，按用户限定范围保留。

## 2026-07-19 关闭层级校正

- 用户最新截图确认预期层级是“主页 → 难度训练 → 子页面”，前一版“一次退出整个训练流程”的判断错误。
- 越级返回的直接原因是难度训练和难度选择两个叉号共用 `closeTrainingFlow`，它同时清除了两个显示状态。
- 答题结果和已掌握单词直达 `/home` 也会越级；它们应携带 `openTrainingSetup: true`，由 `HomeView` 恢复父级页面。
- 关闭语义现统一为显式直接父级，不依赖浏览器历史栈。
- 用户最终确认难度选择页的叉号返回直接父级“难度训练”；完成难度选择的正常点击也返回难度训练。
- 登录后的真实浏览器复现出源码状态测试未覆盖的问题：两个固定定位叉号处于同一坐标，底层叉号的全局 `z-index: 3000` 高于顶层覆盖层的 `z-index: 90`，所以第一次点击实际落到底层控件。
- 顶层叉号自身的 3000 无法越过父级覆盖层建立的 90 层叠上下文；修复点必须提高整个 `.difficulty-overlay`，而不是只改叉号处理函数。
- 提高覆盖层到 4000 后，顶层叉号可以可靠命中；此时将其绑定到只关闭选择层的 `closeDifficultyPicker()`，即可一次回到难度训练而不误回主页。

## 2026-07-19 三条答题链路进入 7003 错题挖空分析

- 前端主页和难度训练页的两个“开始匹配”都发送 `match_start`，差别仅在所带难度参数；“单人训练”发送 `solo_training_start`。
- Java 后端同时处理 `match_start` 与 `solo_training_start`，错题通知入口为 `WrongWordAgentNotificationService`，外部生成策略由 word-agent `WrongWordStrategyService` 执行。
- 7003 是独立的 `rob_english_word_cloze_web`，最终是否能看到题目还需要继续核对其 API 与 `sentence_cloze_item` 的读取条件，不能仅凭首页错题集按钮判断。
- 无论匹配还是单人训练，玩家提交答案都进入 `AnswerService.submitAnswer()`，随后调用相同的 `saveAnswerDetail(...)`；模式只影响机器人、结算和训练统计，不改变玩家答题提交入口。
- `GameSettlementService` 在插入 `GameAnswerDetail` 后统一调用 `notifyWrongAnswer(detail)`；通知服务只判断 `is_correct=0`、真实用户 ID、明细 ID 和单词非空，没有按 `game_record.mode` 排除 `match` 或 `solo_training`。
- word-agent 按“同一用户累计 3 条 `pending` 错词事件”组批，来源和游戏模式不参与分组；因此匹配与单人训练产生的错词可以混合凑成同一道三词挖空题。生成失败时 3 条事件恢复为 `pending`，成功后写入 `cloze_item_id` 并标记 `processed`。
- 7003 的主入口“错题挖空复习”请求 `/api/cloze-practice/tasks/review-due`；Java 服务查询的是到期复习计划。新生成的 `sentence_cloze_item` 与“已经在 7003 答错后创建的到期复习计划”是两个阶段，需要继续核对 mapper SQL，确认新题首次出现的位置。
- mapper 已确认：新生成且尚未作答的 `source='word-agent'` 题目会出现在 7003 的“待答句子”/通用 next-task 查询；它不会立刻出现在首页“错题挖空复习”。
- 首页“错题挖空复习”只返回已有 `sentence_cloze_review_schedule`、最近一次挖空答案仍错误、且 `next_review_time <= NOW()` 的题目。该计划是在用户实际做 7003 挖空题答错后创建，首次安排为次日，因此比赛/单人训练的原始错词和到期复习题之间至少隔着“凑满 3 词 → 生成成功 → 用户首次作答并答错 → 到期”四步。
- 当前账号数据库实况：`game_answer_detail` 有匹配答错 113 条（34 局）和单人训练答错 42 条（13 局）；带明确难度字段的单人训练错题中，`college/college_cet4` 为 8 条。
- `wrong_word_events` 当前共 39 条：`rob_english_word_back` 来源 27 条（26 processed、1 pending），`sentence_cloze_practice` 来源 12 条（7 processed、5 pending）。27 条游戏来源事件全部来自 `solo_training`。
- 现有 113 条匹配错题发生在 2026-03-14 至 2026-04-02，而错词事件最早从 2026-06-02 才开始记录，说明旧匹配错题没有历史回填；代码可保证新产生的匹配错题进入通知链，但存量旧错题不会自动进入。
- 当前已生成 11 道 `source='word-agent'` 挖空题；首批和多数后续批次可明确追溯到单人训练错词，部分批次混合了单人训练错词与用户在 7003 再次答错产生的词。
- 当前还有 6 条 `pending`：1 条来自单人训练、5 条来自 7003 挖空答错。最老的 3 条（`my/adolescence/outpost`）记录了生成接口 HTTP 400；由于策略始终优先取最老 3 条，这个失败批次会在后续事件到达时反复被选中，后面的 3 条暂未被处理。
- 当前 7003 共有 11 道 `word-agent` 题，其中只有题目 9 已产生错题复习计划且已经到期；其余新生成题不会直接计入首页到期错题数。

## 2026-07-20 挖空练习待练口径与旧题兼容

- `conchi` 用户共有 141 条 `sentence_cloze_item`，仅 1 条至少答过一次，因此现有统计显示待练 140。
- 141 条记录的 `sentence` 与 `cloze_sentence` 均完整，不是等待生成的单词。
- 来源分布为 `best-sentence-practice=49`、`difficulty-practice=92`；未答分别为 48 和 92。
- 当前用户单独训练偏好为 `junior/junior`；该难度现有新格式未答题 48 条、旧格式未答题 10 条。
- 首页复习入口只查询答错、存在复习计划且 `next_review_time <= NOW()` 的题目；当前到期数为 0。
- `getStats()` 按全部来源计算总题数减已完成题数；通用待答列表却限定 `source='word-agent'`，口径不一致。
- 单独训练回退查询限定 `source='best-sentence-practice'`，因此 92 条 `difficulty-practice` 旧题不会被复用。
- Word Agent 模型请求另有 `403 Forbidden`，但用户明确要求本次不修复凭证。
- 用户澄清后重新核验：`select_english_word.public.wrong_word_events` 中 `conchi(user_id=2)` 只有 17 条错词事件，15 条 `failed`、2 条 `pending`，并非 140 条。
- 15 条失败事件均已重试 3 次，`cloze_item_id` 为空；错误链为模型接口 403 → Word Agent `/v1/sentences/generate` 502 → Java 外部生成接口 400。
- 当前活动模型为 `aliyun-deepseek/deepseek-v3.2`，请求端点为 `token-plan.cn-beijing.maas.aliyuncs.com`；API key 字段非空，但服务端拒绝授权。
- 剩余 2 条 pending 尚未造句是因为批大小为 3，未凑满一批。
- 页面 140 精确对应 `rob_english_word.sentence_cloze_item` 中 141 道已生成完整句子的题目减去 1 道已答题；141 条 `sentence` 和 `cloze_sentence` 均完整。

## 2026-07-20 Word Agent 本地 CLI 配置

- 参考项目的“本地 CLI 配置”不是纯 UI：Java 保存多条 CLI 配置和默认项，Python Worker 根据 `command/defaultArgs/model/reasoningEffort/workingDirectory/timeoutSeconds` 启动本地进程，并把 CLI 作为 `TEXT_GENERATION` 执行通道。
- 参考项目对 Codex CLI 追加 `--model`、`model_reasoning_effort` 与 `--ephemeral`，提示词通过标准输入传入；执行时隔离 `CODEX_*` 环境变量、设置进程组并在超时后终止整个进程组。
- 目标项目当前 Go 后端只管理 `ai_provider_configs`，前端“模型配置”只支持 OpenAI/Anthropic Compatible；Word Agent 的 `LLMClient` 只解析可用 API Provider 并调用 `/chat/completions`。
- 目标项目的错词造句最终由 Word Agent `/v1/sentences/generate` 执行 LLM → MiMo TTS → MinIO，因此如果 CLI 要解决当前模型 403，运行时执行应落在 Python Word Agent，而不是浏览器或 Go 进程。
- CLI 生成结果仍需满足现有 JSON 契约 `sentence/translation_zh/explanation_zh`，后续 TTS、MinIO 和 Java 落库流程可以保持不变。
- 当前 `word-agent` 确实运行在 Docker：容器为 Linux arm64，只挂载日志、TTS 临时目录和 Go 配置；容器内没有 Node、npm 或 Codex CLI。
- 宿主机 Codex 位于 ChatGPT App 内，是 macOS 可执行文件，不能直接 bind mount 到 Linux 容器执行；截图中的 `/opt/homebrew/bin/codex` 同样属于宿主机路径。
- 若保持全 Docker，需要在 Word Agent 镜像内安装 Linux 版 Codex CLI，并为容器提供专用 Codex 认证配置；另一条路线是在宿主机增加受控 CLI 执行桥接，让容器通过 `host.docker.internal` 调用。
- 用户选择容器内 CLI 方案，并接受只接管错词造句；例句批量生成和句子评分继续使用现有 API Provider。
- 官方 Codex 手册确认：CLI 支持非交互 `codex exec`、独立 `CODEX_HOME`、无界面 `codex login --device-auth`、`--sandbox read-only`、`--ephemeral`、`--output-schema` 和 `--output-last-message`，可用于容器内受控结构化生成。
- Word Agent 当前真实上线的 AI 能力有两类文本任务和一类语音任务：多词合句、已生成例句评分、MiMo TTS；其中多词合句完成后会自动继续 TTS 和 MinIO 上传。
- `generate_word_clean_sentences()` 已实现批量单词例句生成及 JSON 解析，但全项目没有路由或业务调用，当前属于未接入能力。
- `generate_sentence_guidance()` 明确抛出 `NotImplementedError`；`POST /v1/runs/execute` 的 `llm_call` 与 `result_packaging` 步骤也标为 SKIPPED，属于框架占位，不是可用 AI 功能。
- 错词处理器本身不直接调用模型：它按用户凑满 3 个错词后请求 Java，Java 再回调 Word Agent `/v1/sentences/generate`，由该入口执行文本生成与 TTS。
- 已删除未接入的 `generate_word_clean_sentences()`、专用输入/输出类型、模型请求与解析函数，以及未实现的 `generate_sentence_guidance()`；全项目源码和 README 不再存在这些符号。

## 2026-07-20 MiMo TTS 统一模型配置

- 运行时真正生成 MiMo 音频的集中实现只有 `MiMoTTSService`；它被 `/v1/sentences/generate` 的造句后自动 TTS 和独立 `/v1/tts/generate` 两个入口调用。
- 根目录 `scripts/mimo_tts_word.py` 是唯一绕过服务、直接读取 `MIMO_API_KEY` 并调用 Xiaomi API 的生成脚本；要满足“所有地方统一读取模型配置”，应改为调用 Word Agent `/v1/tts/generate`。
- 两个 MinIO 迁移脚本只迁移已存在音频，不调用 MiMo 生成，不需要读取模型配置。
- 当前数据库 5 条配置为 `aliyun-deepseek`、`glm-5`、`kimi-k2-5`、`minimax-m2-5`、`qwen3-6-flash`，均为 `openai-compatible`；用户明确要求全部删除。
- 当前 Word Agent 容器的 MiMo API Key 已配置，Base URL 为 `https://api.xiaomimimo.com/v1`，模型为 `mimo-v2.5-tts`，音色为 `Chloe`，具备无明文输出的一次性数据库迁移条件。
- 现有 `ai_provider_configs` 缺少 `voice` 字段，Go/React 类型也只允许 OpenAI/Anthropic；需要增加 `mimo-tts` 类型和音色字段，并让 MiMo 服务按类型从数据库加载。
- `ai-task-center/python-worker` 也在为英语项目批量生成基础单词和最佳例句的 MiMo TTS；它当前优先读取 AI Task Center 自己的 Provider 配置，并仍保留环境变量回退。
- 因此“所有 MiMo TTS 都读取这个模型配置”若按全项目理解，还必须让 AI Task Center 的 TTS Worker 改读 `select_english_word.ai_provider_configs` 中的 `xiaomi-mimo-tts`，否则会存在第二套配置源。
- 深入核对后，AI Task Center 不仅读取自己的 MiMo 配置，还依赖该 Provider 作为 `AUDIO_TTS` 任务执行目标；简单改成跨库查询会同时引入第二套数据库连接和重复的 Xiaomi 调用实现。
- 更稳妥的统一方式是让 Word Agent 成为唯一 MiMo 网关：只有 `MiMoTTSService` 读取 `select_english_word.ai_provider_configs` 并调用 Xiaomi；AI Task Center 与独立脚本通过 Word Agent 内部 TTS 接口获取音频，彻底删除各自的 API Key、Base URL、模型和音色读取逻辑。
- AI Task Center 可保留 `xiaomi-mimo-tts` 作为无凭证的内置 `AUDIO_TTS` 执行目标，以兼容现有任务快照，但不再把它作为可编辑 Provider 或保存任何 MiMo 密钥。
- 用户明确校正：`ai-task-center` 与当前英语项目是独立项目，不能合并配置或调用链。上述跨项目网关方案不实施；对 AI Task Center 的只读核对仅用于排除范围。
- 本次“所有 MiMo TTS 生成位置”限定为 `rob_english_word_workforce` 内：Word Agent 的造句后自动 TTS、独立 `/v1/tts/generate`，以及根目录示例脚本。
- 用户最终修订数据边界：不能删除现有 5 条文本模型，也不应把 TTS 混入 `ai_provider_configs`；应新增独立 `tts_provider_configs`（或等价独立表）和“TTS 模型配置”页面。

## 2026-07-26 非 Docker 启动全部前后端项目

- 既有记录显示仓库曾通过 `restart_all_services.sh` 在宿主机启动 6 个应用服务，目标端口为 7001、7002、7003、8009、8010、8019、9091；本次仍需以当前文档、脚本和运行态重新核验。
- 当前源码目录确认包含 3 个前端：`rob_english_word_front`、`rob_english_word_cloze_web`、`word_select_dashboard/web-react`；3 个后端：`rob_english_word_back`、`word_select_dashboard/server`、`word_select_dashboard/word-agent`。
- 根目录存在统一宿主机入口 `restart_all_services.sh`，每个应用也有独立 `restart_*.sh`；`deploy/**/start.sh` 属于部署目录，暂不采用。
- `git status --short` 在本轮记录之前无业务代码改动；当前仅规划文件因本轮操作变更。
- 统一脚本默认 `START_DOCKER_DEPS=0` 且 `STOP_DOCKER_STACKS=0`，因此执行 `restart` 不会启动、停止或重建 Docker；会在宿主机通过 `uv`、`go run`、`mvn spring-boot:run` 和三个 `npm run dev` 启动应用。
- 当前 Node/npm、Go、uv、Python、Maven、Java、nc、lsof 和 launchctl 均可用，三个前端依赖目录与 Word Agent 虚拟环境已就绪。
- 当前 6 个应用服务全为停止状态，7001/7002/7003/8009/8010/8019/9091 均未监听；现有 PostgreSQL 5432、Redis 6379、MinIO 19100 已监听，可直接复用。
- 第一次统一启动生成的 `word_agent.plist` 通过 `plutil -lint`，路径、工作目录和执行参数均有效；目标 launchd label 当前并未加载，排除重复 label 冲突。
- 当前 GUI launchd 域 `gui/501` 可正常读取。结合 `bootstrap` 的 I/O error 5，故障边界位于受限执行环境向 launchd 提交持久后台任务，而不是 plist 语法或项目启动命令。
- 旧日志中的 uv cache `Operation not permitted` 同样指向受限环境无法读取用户缓存；下一步以完全相同的 plist 在授权的宿主机环境做最小 bootstrap 验证。
- 授权环境中提交相同 plist 后，launchd 显示 Word Agent 进程正常运行，uv 也能读取缓存；受限环境假设得到确认。
- 新鲜启动日志显示 Uvicorn 监听 `0.0.0.0:6017`，而统一脚本和 Java 默认依赖均预期 `8010`；必须先确认是否由当前 `.env` 或代码默认值改变。
- 当前 HEAD `61d4f72` 明确把 Word Agent 从 8010 改到 6017、Java API 从 8019 改到 6012、WebSocket 从 9091 改到 6013，并把 Java 的 Word Agent 地址改为 6017。
- 根统一脚本和独立重启脚本仍使用旧端口，属于端口迁移后的启动脚本滞后；不能据其旧 700x/800x/9091 端口继续验收。
- Word Agent 的 `.env`、代码默认值和 Java `application.yml` 对 6017 一致，6017 是当前权威端口；下一步盘点全部 601x 配置并按当前配置启动，不修改业务代码。
- 当前权威端口映射为：主前端 6011、Java API 6012、WebSocket 6013、挖空前端 6014、Go API 6015、管理前端 6016、Word Agent 6017。
- Go `config.yaml` 的系统端口为 6015、Word Agent 地址为 6017；三个 Vite 配置分别声明 6011、6014、6016。
- Word Agent 的 launchd 父进程与 Uvicorn 子进程当前保持运行，6017 已实际监听；6018 只属于 CLI runner 代码，未注册为 `pyproject` 项目脚本，也不在统一应用清单中，本次不作为独立项目启动。
- 继续使用现有启动脚本时必须关闭其旧端口等待，并显式把 Java 的 `WORD_AGENT_BASE_URL` 覆盖为 6017；实际应用监听仍由当前源码配置决定。
- 统一脚本在授权宿主机环境完成 6 个 launchd 服务提交；`status` 均显示 RUNNING。
- 实际监听已覆盖全部当前端口 6011–6017：三个 Node 前端、Java API/WebSocket、Go API、Python Word Agent 均有对应监听进程。
- 新鲜日志显示 Word Agent application startup complete、Go 完成数据库/AI 配置/MinIO初始化并注册路由、Java 在 6012/6013 启动完成、三个 Vite 前端均在各自 601x 端口 ready。
- 日志文件采用追加模式，包含本轮之前的旧错误；本轮判断以 2026-07-26 15:28 之后的启动段和当前监听为准。
- 浏览器验收确认三个前端分别渲染“英语抢词大战”登录页、“挖空练习”和“Word Agent 数据追踪后台”，均无控制台错误。
- 最终新鲜验收为 0 失败：6 个 launchd 服务 RUNNING，6011–6017 七个端口全部可连接；三个前端、Go `/health`、Word Agent `/health` 返回 200；Java `/actuator/health` 返回安全策略下的预期 403。
- 所有应用均通过宿主机 Node/Java/Go/Python 进程启动；本轮统一启动显式设置 `START_DOCKER_DEPS=0`、`STOP_DOCKER_STACKS=0`，未启动、停止或重建 Docker 服务。

## 2026-07-26 四个答题入口的错词造句队列核对

- 截图 1 是主前端 6011 首页的普通“开始匹配”。
- 截图 2 是主前端 6011 的“难度训练”层，本轮按用户四入口口径核对其中“开始匹配”；同页“单人训练”是另一个游戏模式，但不是截图 4 的挖空“单独训练”。
- 截图 3 是挖空前端的“错题挖空复习”，入口文案为“开始答题”，目标是到期复习题。
- 截图 4 是挖空前端的独立“单独训练”，按所选词库难度练习已存在的句子挖空题。
- 主前端截图 1 的按钮调用 `handleMatch()`；截图 2 的“开始匹配”调用同一个 `handleMatch(selectedDifficulty)`，差别只是是否携带选定难度。
- 挖空前端截图 3、4 最终共用 `App.tsx` 的同一答题表单和 `handleSubmitAnswer()`；两种 launcher 只负责用不同查询装载题目。
- 两个匹配入口都发送 WebSocket `match_start`，数据结构相同：`difficultyGroup`、`difficultyLevel`；首页使用段位难度，难度页使用用户选定难度。
- 挖空复习通过 `getDueReviewTasks` 装载到期复习题并把 `practiceSource` 设为 `review`；挖空单独训练通过 `createDifficultyBatch` 装载所选难度题并把 `practiceSource` 设为 `solo`。
- 挖空复习和挖空单独训练提交答案时调用完全相同的 `POST /api/cloze-practice/answers`，请求只含 `clozeItemId`、答案、答案文本和耗时，不携带 `review/solo` 模式；因此后端通知逻辑不会因这两个页面入口而分叉。
- Java `WrongWordAgentNotificationService` 同时存在游戏错词来源 `rob_english_word_back` 和挖空错词来源 `sentence_cloze_practice`；多空答案会按错误空位逐词发事件，事件失败后的重试由 Word Agent 状态机负责。
- Java `ClozePracticeService.submitAnswer()` 在任何挖空题答错时先写 `sentence_cloze_answer_record`、更新次日复习计划，再调用 `notifyClozeWrongAnswer()`；它不检查题目是由 `review` 还是 `solo` 页面装载，因此截图 3、4 都会触发同一通知。
- 挖空题若有多个空，通知服务逐个比较标准答案；每个答错的空各发一条 `sentence_cloze_practice` 事件。答案数量不等时，所有标准词都按错误处理。答对则不发事件。
- 游戏匹配结算保存每条 `GameAnswerDetail` 后统一调用 `notifyWrongAnswer()`；通知服务只接受 `is_correct=0`、真实用户、非空明细 ID 和非空目标词，不检查 `game_record.mode`。因此普通匹配和带难度匹配都进入同一 `rob_english_word_back` 来源。
- 游戏错词和挖空错词通知都是异步 HTTP POST 到 Word Agent `/v1/wrong-words/events`；Java 本身不先落本地队列表，HTTP 失败只记警告。因此“代码会尝试入队”与“网络失败时必然落队列”需要区分。
- Word Agent 接口收到事件后只做幂等入库：唯一键是 `(source, source_answer_detail_id)`，初始状态为 `pending`，响应中的 `generated` 固定为 `false`；随后唤醒独立后台处理器。
- 后台处理器按“同一用户至少凑够 `cloze_batch_size` 条 pending”领取最早一批，不按来源或入口分组；因此四个入口产生的错词可以互相混合凑批。
- 领取后状态变为 `processing`，以三词批次调用 Java 外部生成接口；成功后整批变为 `processed` 并写 `cloze_item_id`。
- 生成失败不会等同于未入队：前两次变为 `retry_wait`，等待该用户出现更新事件后重试；达到最大重试次数后变为 `failed`。所以“入队成功”不能推出“句子已经生成成功”。
- 当前代码默认组批大小为 3、每次唤醒最多处理 10 批、最大重试 3 次；`.env` 未覆盖这些值，运行时使用默认配置。
- Java 生成服务用批次 key 做幂等，成功生成 `sentence_cloze_item` 时 `source='word-agent'`，并保存源事件 ID；它会要求文本造句、TTS 和音频地址全部成功后才落题目。
- 当前 `select_english_word.wrong_word_events` 实况为 25 条：游戏来源 24 条 `failed`、1 条 `pending`；当前没有 `sentence_cloze_practice` 来源记录，也没有 `processed` 记录。
- 最新 pending 是用户 2 的 `prematurely`，尚未凑满 3 条；24 条 failed 均已重试 3 次，错误来自旧运行环境的生成链路不可达或上游造句失败。
- 当前库“没有挖空来源事件”只能说明现存数据里没有，不能推翻当前源码的入队逻辑；下一步需要核对挖空答错记录时间和应有事件键，判断是否是功能上线前历史数据或当前尚未实际答错测试。
- 跨库回查 25 条游戏事件：23 条来自 `game_record.mode='match'`（用户 1 为 6、用户 2 为 17），2 条来自 `solo_training`（用户 2）。这证明当前运行数据中匹配答错和游戏单人训练答错都实际入过队。
- 当前 `sentence_cloze_answer_record` 只有 1 条答对记录、0 条答错记录，所以队列表没有 `sentence_cloze_practice` 来源是“当前没有挖空答错样本”，不是通知缺失；不存在应有但缺失的挖空事件。
- 23 条 match 事件的实际难度样本覆盖高中、大学六级、商务出国、高阶考试，全部带 `match_difficulty_*` 字段，证明截图 2 的“指定难度开始匹配”已在运行数据中真实入队。
- 当前没有 `rank_current`（截图 1 首页默认段位匹配）的数据库样本，但截图 1、2 前端共用 `handleMatch()` 和 `match_start`，后端均保存为 `mode='match'`，错词通知不读取难度字段，因此源码链路等价。
- 补充：截图 2 同页的游戏“单人训练”也会入队；当前库已有 2 条 `solo_training` 事件。它与截图 4 的挖空“单独训练”不是同一业务入口。
- 最终新鲜只读查询仍为 25 条：`rob_english_word_back/failed=24`、`rob_english_word_back/pending=1`；映射到主库后为 `match=23`、`solo_training=2`。当前没有 `processed`，因此不能把“已入队”表述成“已成功生成句子”。
- Word Agent 定向测试 8/8 通过，覆盖事件接口只入库并唤醒处理器、三词 pending 领取、重试等待和第三次失败隔离；Java `ClozePracticeServiceTest`、`GameSettlementServiceTest`、`SentenceClozeServiceTest` 合计 15/15 通过。

## 2026-07-26 failed 与 retry_wait 根因核对

- `conchi` 的 18 条 `failed` 全部是 6 个三词批次，`retry_count=3`；不是 18 次彼此独立的入队失败。
- 前 5 批共 15 词的最终外层错误均为 Word Agent 调 Java 6012 返回 400，Java 再报告其回调 Word Agent `/v1/sentences/generate` 得到 502；此前同一批历史核验已定位到底层活动 API 模型返回 403。
- 第 6 批 3 词 `redintegrate/sacrilegious/fracture` 的第三次错误为 `[Errno 101] Network is unreachable`，发生在 2026-07-25 21:28:59。
- 当前 3 条 `retry_wait` 是批次 `79-80-81`：`prematurely/momentum/frustration`，已失败 2 次，`retry_after_event_id=82`。
- 当前造句执行器于 2026-07-25 20:41:13 切换为 `cli/codex`；所有 API provider 当前均为 inactive。
- 最新两次失败的最底层异常是 `LLMClient._cli_completion()` 对 `settings.select_db_dsn=None` 直接调用 `.strip()`，抛出 `AttributeError`；因此 CLI 尚未真正启动，更没有到模型、TTS 或 MinIO。
- `llm_client.py:301` 的直接 `.strip()` 来自提交 `61d4f72`。代码后面已有环境变量和 `_resolve_select_db_dsn()` 回退，但异常在进入回退前发生；现有测试都显式传入非空 `select_db_dsn`，没有覆盖默认 `None`。
- `retry_wait` 的两次来源：事件 81 凑满三词后第一次处理失败；随后事件 82 到达满足“同用户出现更新事件”条件，批次重试并第二次失败。现在需等待 id 82 之后的新事件才会进行第三次尝试。
- Codex 可执行文件 `/Applications/ChatGPT.app/Contents/Resources/codex` 存在且可执行，版本为 `codex-cli 0.146.0-alpha.3.1`，登录状态为 `Logged in using ChatGPT`。
- 使用当前数据库中的 `codex` 配置、项目 `CLIRunner` 和无敏感合成词 `apple/book/sun` 真实执行成功；模型 `gpt-5.3-codex-spark` 在 10.779 秒内返回满足契约的 `sentence/translation_zh/explanation_zh` JSON。这证明 CLI、认证、模型和结构化输出均可用。
- 对当前 Word Agent 6017 使用相同合成词请求 `/v1/sentences/generate`，稳定返回 HTTP 500；新鲜日志再次落在 `llm_client.py:301` 的 `None.strip()`，证明 CLI 根本未被调用。
- 6018 CLI Runner 当前没有监听进程，launchd job 也不存在；但当前 HEAD 已把 Word Agent 从 HTTP 6018 Runner 改成进程内直接 `CLIRunner`，所以它不是这次 500 的直接原因。
- 当前 `test_generate_sentence_uses_cli_runner_for_cli_target` 仍断言 HTTP 6018 调用，与生产实现不一致；定向执行结果为 1 failed，并精确复现同一个 `None.strip()`。这说明提交 `61d4f72` 修改实现时没有同步该测试。

## 2026-07-26 本地无鉴权 CLI Runner 双模式兼容

- 原 `2026-07-22-unified-api-cli-sentence-executor-design.md` 已经确定宿主机 6018
  Runner 是 Docker 访问 macOS CLI 的边界；当前回归来自后续提交把 Word Agent
  改成进程内直接执行。
- 要用一套代码兼容 Docker 和非 Docker，应恢复 HTTP Runner 架构，只通过
  `WORD_AGENT_CLI_RUNNER_URL` 区分 `127.0.0.1` 与
  `host.docker.internal`。
- 用户确认当前只在个人本机使用，暂不需要 Runner Token 鉴权；设计明确接受
  6018 对本机进程、容器和同一网络可达者开放的风险，并禁止公网部署。
- Word Agent 不再读取 CLI 数据库 DSN；Runner 独占 DSN 和 CLI 子进程执行，
  可直接消除当前 `None.strip()` 并保持容器不依赖 macOS 二进制。
- 历史异常队列不纳入代码验证；合成词完成非 Docker/Docker 全链路后，再单独
  确认恢复范围和外部模型调用授权。
- 实施后 Word Agent `_cli_completion()` 只使用
  `cli_runner_url + /v1/text/generate`，不再导入或构造
  `CLIProviderConfigClient/CLIRunner`，也不读取 CLI 数据库 DSN。
- Runner `/health` 仅以数据库 DSN 判断就绪；无 Authorization Header 的
  `/v1/text/generate` 已由 API 测试和真实 6017 调用共同验证。
- Docker Compose 展开结果保留
  `WORD_AGENT_CLI_RUNNER_URL=http://host.docker.internal:6018` 和
  `host-gateway`，且不存在 Token 环境变量；按用户“不用 Docker 启动”的边界，
  本轮没有启动容器做运行态测试。
- 非 Docker 最终运行态为 7 个 launchd 应用进程加 1 个宿主机 Runner，
  6011–6018 全部监听，6017/6018 健康接口均返回 200。
- 真实合成词调用的 Runner 日志为 `executor_id=codex`、
  `driver=codex`、`model=gpt-5.3-codex-spark`、`exit_code=0`，耗时约 6.35 秒。
- 自测前后 `wrong_word_events` 均为 `failed=24, pending=1, processed=3`，
  整表摘要均为 `bfc3884a89f7a4bbe9c5150f87282ac8`；三个合成词匹配事件数为 0。

## 2026-07-26 可入队错题记录列表

- 用户已确认错题集需要统计普通游戏答错和挖空答错产生的全部可入队单词事件；一条多词挖空记录中每个实际错误的目标词单独占一行。
- 用户端列表字段确定为：来源单词、答错时间、入口/模式、词库/难度、词难度、耗时、正确答案。
- 所有用户端和管理后台错题、训练结果、造句结果相关表格都删除“你的答案”展示列，但数据库字段、接口兼容字段和队列负载继续保留。
- 当前用户端 `/api/wrong-words` 只聚合 `game_answer_detail` 中 `is_correct=0` 且 `word_id` 非空的记录，不能覆盖无 `word_id` 的队列候选，也不能覆盖挖空答错。
- 当前 Java 通知资格对普通答题要求答错、有效用户、已持久化明细和非空单词；对挖空答题按错误下标逐词发送，答案数组长度不一致时所有标准词都视为错误。
- 为避免破坏现有聚合接口，设计新增 `/api/wrong-words/events` 扁平事件接口，现有列表和详情接口保留。
- 工作区已有上一项“答错立即复习”的两个 Java 未提交改动和一份未跟踪实施计划，本任务必须保留，不能覆盖或混入错误删除。
- 第二张参考截图实际来自管理后台 `ClozeResultTable` 的造句来源单词展开表；用户最终要求所有同类位置都删除“你的答案”，不只两张截图对应页面。
- 管理后台存在六类具体渲染：独立用户错题、独立用户造句错题、用户详情错题弹窗、用户详情造句错题弹窗、训练结果明细、造句来源单词展开表；各处 CSS 网格需同步减少一列。
- 用户前端训练答题结果仍渲染“你的答案”，需要删除表头、行单元格及未再使用的 `selectedAnswerText`。
- 普通答题的准确答错时间应使用 `game_answer_detail.create_time`；入口和难度信息来自关联的 `game_record`，正式匹配使用 match 难度字段，单人训练使用 training 难度字段。
- 挖空答题后端请求不携带 `review/solo` 模式，题目 `source` 与 `attempt_no` 只能给出可解释的近似映射：最佳例句题为单独训练，外部错词生成题首次为待练句子，后续尝试为错题复习。
- 现有 Java 测试没有 WrongWord mapper 合同测试；可新增反射读取 `@Select` SQL 的合同测试，先锁定 `UNION ALL`、JSON 按下标拆分、空词过滤和排序条件。
- 新增 SQL 使用的 `sentence_cloze_item.provider_label`、`source` 与答题记录 JSON 字段均已由现有实体/Mapper 证实存在；当前静态结构检查未发现字段名偏差。
- 完整验证脚本确认：用户端为 `vitest` + `vue-tsc/vite build`，后台为 Node 合同测试 + `tsc --noEmit/vite build`；生产源码已无“你的答案”文本或 `word.selectedAnswer` 渲染。
- 本机未提供 `psql/pg_isready` CLI；Word Agent 已有项目内 `.venv`，真实 PostgreSQL 只读验证改用其已安装数据库驱动，避免新增依赖。
- 真实库可成功规划并执行统一 CTE；三个有错题用户分别返回 15、6、22 条普通答题事件。库内目前没有错误挖空答题行，所以无法用现存数据展示 cloze 样本，但 JSON/UNION 分支已由 SQL 合同测试覆盖。
- 代码审查指出 `source_word_ids_json` 与目标词按数组序号对齐；对 `word-agent` 来源可安全优先使用该 ID，`best-sentence-practice` 的 ID 属于 `word_clean`，因此仍按英文内容回退，避免误撞 `word.id`。
# 2026-07-26 错题集持久复习状态口径修正

- 产品口径纠正：`/wrong-words` 不是“进入队列的错误事件流水”，而是“仍未完成复习的错词集合”。同一用户同一单词应聚合显示，从入队开始持续存在，直到复习流程真正结束。
- `sentence_cloze_review_schedule` 是当前挖空复习的进行中状态表，主键语义为 `(user_id, cloze_item_id)`；字段包含 `correct_streak`、`review_stage`、`next_review_time`。
- 当前正确推进逻辑：第一次复习答对后安排 7 天，第二次答对后安排 15 天，第三次答对时删除 `sentence_cloze_review_schedule`。因此“计划仍在表内”可以表示该句子尚未完成整套复习，“记录被删除”表示该句子的三阶段复习已结束。
- `user_word_state` 目前仅有用户、单词和创建时间，不携带队列/复习阶段，不能单独作为错题集是否完成的可信状态。
- `wrong_word_events.status` 只表达造句队列处理状态（`pending/processing/retry_wait/failed/processed`），`processed` 仅表示已生成 `cloze_item_id`，不表示用户完成复习。
- 造句成功时，Python 会把一批错误事件 ID 写入 `sentence_cloze_item.source_event_ids_json`，并把生成的 `cloze_item_id` 回写每条 `wrong_word_events`；这是队列错误事件关联复习句子的现有链路。
- 挖空题第一次答错时才创建 `sentence_cloze_review_schedule`；尚未第一次作答的生成句子没有计划记录。因此仅用“计划表存在”会漏掉已入队、待生成或已生成但未首答的错词。
- 当前完整未完成集合至少包含三段状态：造句前（队列事件尚未 processed）、已生成但未首答（cloze item 无答题记录）、已进入三阶段复习（review schedule 仍存在）。
- 现有三阶段计划并不是在“原始单词答错入队”时创建，而是用户在生成后的挖空题里再次答错时才创建。生成句子首次作答若直接答对，`advanceReviewScheduleOnCorrect` 因找不到计划而直接返回，页面会把该句子视为已掌握。
- 现有到期错题查询还要求最近一次答案为错误，这会导致第 1 次答对后虽已安排 7 天复习，到了 7 天仍无法被查询出来；旧设计文档已明确记录这个独立缺陷。
- 因此若产品定义为“每个入队错词都必须完整走完立即、7 天、15 天”，不能只改错题集查询；还需要把复习计划的起点前移到入队生成句子，并修复 7/15 天到期查询条件。
- 用户已确认三词句子中的复习进度按单词独立：某个词答错只重置该词，另外两个答对词继续各自阶段。
- Java 当前向 Word Agent 发送错词事件采用异步 fire-and-forget；Word Agent 返回 `eventId`，但 Java 使用 `toBodilessEntity()` 丢弃响应。因此把 Java 页面状态严格建立在“已成功写入 Python 队列表”上会引入跨服务同步依赖。
- 推荐在 rob 数据库新增单词级复习进度作为用户复习域的事实来源：原始错误提交时激活/重置，造句保存时关联题目，逐空判定时独立推进。Python 队列表继续只负责造句传输，不承担复习完成状态。
- 普通答题的三个持久化路径最终都集中到 `GameSettlementService`，每条 `GameAnswerDetail` 插入后依次调用掌握进度和 `notifyWrongAnswer`；适合在通知前统一调用单词进度服务，覆盖正式匹配、难度训练和单人训练而不散落改动。
- 用户端 `/api/wrong-words/events` 已在 Task 5 改为以 `wrong_word_review_progress.status <> 'completed'` 为主集合，七列事件数据只作为最近来源元数据；Task 6 前端只需切换持续复习语义和稳定 `progressKey`。
- 管理后台主“用户错题集”仍从 `game_answer_detail` 聚合，因此会漏掉仅经挖空入口激活的待复习词，也会在进度完成后继续显示；应改为进度表主查询，普通答题明细仅补充最近模式、难度和展开历史。
- Go 管理后台通过现有 PostgreSQL 配置连接同一个 `rob_english_word` 数据库，因此进度表迁移应用后，主错题列表无需跨服务同步或复制数据。
- 历史数据幂等回填后共有 28 个未完成错词：21 个已关联现有 Word Agent 句子并立即到期，7 个尚未关联句子；这 28 个都会持续出现在错题集，直至各自完成三个正确复习阶段。
- PostgreSQL JDBC 对 `#{keyword} IS NULL` 的空参数无法推断类型；用户端无搜索词访问会报 `$3` 类型错误。事件列表与 count 查询都必须显式 `CAST(#{keyword} AS text)`，且每次参数出现都要带类型。
- `linkActiveSentence` 不能对相同题目重放时重置阶段；错题 upsert 也必须区分精确重复、较旧乱序事件与真正的新错误，避免重复计数和最近时间倒退。
- 数据库采用 `rob_english_word_back/db/*.sql` 独立幂等脚本管理；新增进度表应沿用该方式，并由现有根启动/初始化流程执行，而不是引入新的迁移框架。
- `ClozePracticeServiceTest` 已有 Mockito 构造式依赖和立即复习测试，可扩展单词级进度 mock；`SentenceClozeServiceTest` 可锁定生成时三词关联；`GameSettlementServiceTest` 可锁定普通答错激活。
- 当前用户端 `WrongWordsView.vue` 已完成七字段扁平表和窄屏布局，可以复用 UI，只需把 DTO/key/文案和后端查询改成未完成唯一单词；无需重做视觉结构。
- Java 现有 `/api/wrong-words/events` 及 Provider 按错误事件分页；可保留路由兼容但把 Provider 改为先用 `wrong_word_review_progress` 限定未完成单词，再从统一事件 CTE 选每词最近一条元数据并按 `wrong_count` 排序。
- Go 管理后台 `WrongWords` 当前只从普通 `game_answer_detail` 聚合且包含历史明细。为了与用户端一致，主集合应改为以 `wrong_word_review_progress` 为过滤/计数事实表，明细仍可读取历史答题记录；独立“造句错题集”页面属于挖空错误事件历史，可保持历史诊断用途，不替代待复习主集合。
- Java 编译工具链根因已确认：项目声明 Java 21，系统 `java` 也是 GraalVM 21，但当前 Maven 进程被环境指向 Homebrew JDK 26；Spring Boot 3.2 管理的 Lombok 在 JDK 26 全量编译下未执行注解处理。测试和构建必须显式使用 `/Users/conchi/Library/Java/JavaVirtualMachines/graalvm-ce-21.0.2/Contents/Home`。

# 2026-07-30 错题集例句展示

- 用户端 `WrongWordsView.vue` 当前展示七列，接口 DTO 与 SQL Provider 尚未返回例句。
- `word` 表已包含 `sentence` 和 `sentence_translation`；`wrong_word_review_progress.word_id` 可用于优先关联原错词所在词库记录。
- `word_clean_best_sentence` 提供按单词唯一的 AI 最佳例句，可作为原词库例句缺失时的回退。
- 用户确认只展示英文例句，不展示中文翻译；原词库优先，AI 最佳例句回退。
- 推荐在现有 `/api/wrong-words/events` 查询中一次性补充例句，避免前端按行请求。
- 高亮必须使用安全文本片段渲染，匹配忽略大小写、支持词组，并阻止 `raw` 高亮 `draw` 之类的单词内部子串。
- 真实生产查询当前 22 个未完成错词均可解析例句：20 条来自原词库，2 条来自 `word_clean_best_sentence`，0 条缺失。
- 小屏卡片的通用直接子元素 `display: grid` 会覆盖 `-webkit-box` 两行截断；例句需使用“字段容器 + 内层截断文本”两层结构。
- 规范化回退使用 `LOWER(BTRIM(word))`；当前数据量与实测正确，但若规模继续增长，应评估函数索引，并让最佳句回退仅在原词库例句为空时执行。
- 用户后续明确修订来源规则：错题集只允许使用
  `word_clean_best_sentence.sentence`；最佳句缺失时显示 `—`，不再查询或
  回退 `word.sentence`。
- 结果级 PostgreSQL 测试需要 7 张最小临时表覆盖生产 Provider 的完整 CTE；
  `SET LOCAL search_path TO pg_temp` 加事务回滚可以执行真实 SQL，而不接触
  持久表数据。
- 当前账号 22 个未完成错词按最佳句唯一来源查询后，20 个命中
  `word_clean_best_sentence`，2 个返回 `none/null`；这 2 个不再回退原词库
  例句，符合最新业务口径。
# 2026-07-30 全部单词答题切换到 word_clean

- 当前单词四选一题目的主数据来自 `word`，词库筛选通过 `word.library_id ->
  word_library.id` 和 `word_library.library_name` 完成。
- 段位匹配按 `word.difficulty` 的 1–1000 十档概率随机抽题；指定难度匹配按
  `TrainingDifficultyCatalog.libraryNames` 过滤词库。
- 单人训练会先从 `user_word_mastery_progress` 取到期复习词，再从 `word` 随机补充
  未掌握词；指定词库不足时会回退到段位难度词池。
- 每题正确释义来自目标词，三个干扰释义来自同一轮已抽取的其他单词。
- `word_clean_best_sentence` 当前只用于错题列表例句，与单词四选一抽题无关。
- 迁移的核心风险不是随机查询本身，而是 `word_clean` 是否具备等价的词库归属、
  难度字段，以及所有以旧 `word.id` 为外键或逻辑引用的用户进度链路如何保持一致。
- `word_clean` 已包含 `word`、`meaning`、`difficulty`、`frequency`、`sentence`，
  以及去重后的 `pep_difficulty`/`source_difficulty` 与标签，但没有
  `library_id`；它用“同一词只保留最低来源难度”的方式表达教材/考试来源。
- 因此指定词库迁移不能原样照搬 `word_library.library_name IN (...)`；需要把前端
  难度目录映射为 `word_clean.source_difficulty` 或 `pep_difficulty` 的值/范围。
- 该最低来源策略会使跨多个词库出现的单词只属于最早的来源层级，例如进入更高阶
  词库过滤时不会再次出现；这是去重表与当前原始词库一词多行模型的实际语义差异。
- `word_clean` 的来源等级完整映射为：PEP 小学/初中/高中 1–24、CET4 25、
  考研 26、BEC 27、CET6 28、IELTS 29、TEM4 30、TEM8 31、TOEFL 32、
  GMAT 33、SAT 34、GRE 35、其他 36。
- 当前用户掌握、普通答题明细、用户单词状态和错词进度等表均存在名为 `word_id`
  的历史字段；部分链路会用它重新连接 `word`。不能直接把 `word_clean.id` 写进这些
  旧字段，否则相同数字可能指向两张表中的不同单词。
- 安全迁移需要显式表达题目来源，例如新增 `word_clean_id`，同时以标准化单词文本
  作为跨新旧题库的稳定兼容键；旧数据继续保留旧 `word_id` 语义。
- `user_word_mastery_progress` 目前以 `(user_id, word_id)` 唯一，虽然没有数据库外键，
  Java 查询明确把该 ID 连接到 `word.id`；`game_answer_detail`、`user_word_state` 和
  `wrong_word_review_progress` 也都沿用旧 `word_id` 命名和语义。
- 旧错题详情接口、已掌握状态、复习完成回写等多个下游会按 `word_id` 查询或
  `wordMapper.selectById`，所以只替换抽题 Mapper 而不迁移标识链路会产生静默串词。
- `wrong_word_review_progress` 已以 `(user_id, normalized_word)` 做用户内唯一键，
  这条错题挖空链路更适合以单词文本兼容新旧来源；但其 `word_id` 仍可能被复习完成
  后的掌握回写使用，需要同步改造。
- 真实库当前 `word_clean` 有 22,098 条，标准化单词 22,098 条、释义缺失 0、
  难度越界 0；各来源难度最少也有 63 词，全部足够支撑每轮抽题。
- 真实库现有掌握进度 76 条中 68 条可按单词映射到 `word_clean`；普通答题明细
  126 条中 114 条可映射。未映射项很可能来自此前已删除的人名/地名或历史脏词，
  设计中必须保留这些历史记录而不是强制塞入新题库。
- 挖空“选择难度/单独训练”已通过 `SentenceClozeItemMapper` 从 `word_clean` 联接
  `word_clean_best_sentence` 选择候选句，因此该入口的抽题主源本来就是
  `word_clean`。
- 挖空错题复习读取已经生成的 `sentence_cloze_item`，并非每次临场从基础词表抽题；
  其中 `word-agent` 来源是普通答错词生成的句子，其他最佳句练习的来源 ID 已是
  `word_clean.id`。
- 当前挖空答对后的“已掌握”回写仍使用 `WordMapper` 将单词解析回旧 `word.id`，
  即使挖空候选题来自 `word_clean`；若用户的“所有答题”包含挖空链路，这部分也需要
  切换为显式 `word_clean_id`。

# 2026-08-02 完形句子错题集实现方案

- 截图一显示完形练习首页已有“开始答题”和“单独训练”两个入口，页面当前只有今日到期
  数量和两个操作按钮，没有可浏览的错题列表入口。
- 截图二可作为“主行摘要 + 单行展开详情”的交互参考；它当前偏管理审计视角，包含用户、
  模型、来源追溯等字段，不适合直接复制到用户复习页。
- 用户明确判定规则：一道句子只要有一个目标单词拼写错误，就进入句子错题集。
- 用户明确复习目标：句子错题遵循现有单词错题相同的复习规律；具体状态机与完成条件仍需
  从源码确认。
- 初步 UI 方向应优先展示“句子、错词数、错误次数、当前阶段、下次复习、最近作答”，
  展开后再展示每个空的正确答案/本次拼写、中文释义、来源模式与复习轨迹。
- 工作区已有直接相关的事实文档 `docs/chains/cloze-practice-review.md`、
  `docs/chains/wrong-word-mastery.md`，以及历史设计 `2026-07-18-cloze-review-solo-training-design.md`、
  `2026-07-26-persistent-word-review-progress-design.md`，可用于区分现状与历史设计意图。
- 最近提交集中在 Context Router/部署流程，不代表当前答题业务已经变更；设计判断仍以当前源码为准。
- 工作树已有用户的 `AGENTS.md`、部署目录和既有规划文件修改，本轮只追加规划记录，不覆盖或整理这些改动。
- 完形前端是独立 React 单页应用，状态切换集中在 `App.tsx`，入口组件为
  `PracticeLaunchers.tsx`，API 集中在 `src/lib/api.ts`；没有 Router 页面体系。
- 完形权威写入方是 Java `ClozePracticeController`/`ClozePracticeService`，现有接口已覆盖
  next、pending、review-due、answered、difficulty-batch、answers、stats 与 history。
- 跨项目文档把“并发提交/网络重试的答案幂等性”列为待运行核对项；新错题集设计必须显式
  提供幂等键或数据库唯一约束，不能假定现有 POST 天然幂等。
- 现有错词业务口径是“当前需要复习的状态”，不是完整答错历史；句子错题集也应分离
  当前复习队列与不可变答题历史，避免列表永久膨胀。
- Java 应继续作为判定和复习进度的唯一写入方；Go 管理服务与 Python Agent 不应直接推进
  用户句子复习状态。
- 历史设计显示完形练习本来就有整句级 `sentence_cloze_review_schedule`：首次/再次答错立即到期，
  连续答对后依次安排 7 天、15 天，第三次答对后删除计划；这与用户要求的“跟单词一样”节奏高度一致。
- 现有单词错题集后来新增 `wrong_word_review_progress`，原因是造句队列、整句计划和错误事件无法
  统一表达逐词完成状态；它按 `(user_id, normalized_word)` 聚合，阶段 0/1/2/3 分别表示立即、
  7 天、15 天、完成。
- 句子错题集若按“任一空错即整句入集”，不应照搬单词表的逐词唯一键和逐词推进；更自然的唯一键是
  `(user_id, cloze_item_id)`，并保留每次答案明细作为历史证据。
- 现有首页的“开始答题”就是到期错题复习入口，单独训练是主动练习入口；新增错题集主要缺的是可浏览、
  可筛选的句子级状态列表，不应再造第三套复习执行器。
- 需要在当前源码核实 `sentence_cloze_review_schedule` 是否仍删除完成记录，以及到期查询是否已取消
  “最近一次仍为错误”的旧限制；历史设计不能当作当前事实。
- 当前源码仍保留独立的整句 `sentence_cloze_review_schedule`，唯一索引已是
  `(user_id, cloze_item_id)`；Mapper 同时存在答错 upsert、答对推进和删除计划三类操作。
- React `App.tsx` 已有“复习计划”和“错题句子”的列表配置，文案写着连续答对 3 次后转为已会；
  因此新增“错题集”可能更多是把既有隐藏/次级列表提升为首页入口并重定义查询字段，而非从零开发列表。
- 当前首页启动组件仍只有“开始答题”和“单独训练”；可在同一启动区增加第三个“错题集”次级按钮，
  或在左侧统计卡片增加可点击入口，需结合 App 当前 overlay/list 状态判断最小改动。
- `App.tsx` 有两处 `submitAnswer` 调用，说明错题复习与单独训练可能走同一 API、不同客户端状态；
  后端统一判定可以天然覆盖两个入口，但需要核实两处提交负载是否都携带相同 taskId/answers。
- 已核实两种模式正常提交均使用统一 `POST /api/cloze-practice/answers`，负载核心为
  `clozeItemId + answers[] + answerText + costMs`；强制显示答案也提交 `?`，因此也会被权威判为错误，
  设计应把“查看答案”算作一次错题（可用触发来源区分），避免绕过学习状态。
- 后端整题判定正是“答案数量一致且每个下标规范化后相等”；任何一个拼写不匹配都会使整题
  `correct=false`，已满足用户的入集判定，无需前端二次判断。
- 当前 schedule 完成后调用 DELETE，因此它只能表示“正在复习”，不能表示已完成/曾经错过的持久错题档案；
  如果错题集只展示未完成项可以复用，若还要“已完成”筛选则必须保留状态或新增独立进度事实。
- React 当前已有 pending/review/mastered/wrong 四个标签页，但 `wrong` 查询语义是“最近一次答错”，
  不等同于“仍未完成三轮的错题”；未来错题集应以持久复习进度为主集合，答题记录只做详情。
- DB 脚本中的注释/历史回填仍写“隔天/1 天”，而 Java 历史设计和当前服务可能已改为立即；实施时需
  同步修正文档/回填规则，避免新旧数据的首阶段时间不一致。
- 当前实现存在两套复习所有权：`source=word-agent` 走 `wrong_word_review_progress` 的逐词推进；
  `source=best-sentence-practice` 走 `sentence_cloze_review_schedule` 的整句推进。两者都写统一
  `sentence_cloze_answer_record`，但列表与到期队列没有真正统一。
- 首页 `/tasks/review-due` 只查 word-agent 的逐词到期句子；单独训练答错产生的整句 schedule 不会
  进入首页，只会在同难度没有新候选时由 difficulty-batch 回捞。这正是新错题集需要修复的业务缺口。
- 推荐不是新增第三张并行表，而是把现有 `sentence_cloze_review_schedule` 升级为所有句源共用的
  持久句子进度：增加 `status/wrong_count/first_wrong_time/last_wrong_record_id/completed_time`，
  完成三轮改为标记 completed，不再删除；`wrong_word_review_progress` 仍独立负责单词掌握。
- 统一提交事务中应同时维护两个正交事实：整句进度按“任一空错则整句重置”，单词进度仍按每空
  独立推进。这样既满足新句子错题集，又不破坏单词错题集的逐词复习与掌握逻辑。
- 到期任务应合并“句子进度到期”和“既有 word-agent 单词进度到期”，按 `cloze_item_id` 去重并取最早
  到期时间；这样未曾答错的单词复习句仍可继续服务，答错过的句子也能按整句规律复习。
- 当前提交没有客户端 submission ID，`attemptNo=count+1` 后直接 insert；重复点击、重试和并发可能
  重复计数并重复推进。方案应新增每次题目展示生成的 `submissionId` 与唯一约束，命中时返回原结果。
- 当前 UI 的“复习计划按 1 天、7 天、15 天”已过时；实际 Java 对非 word-agent 答错传入 `now()`，
  新文案应明确“立即、7 天、15 天”。
- 两张截图的推荐组合是：首页主按钮保持“开始答题”，下方两个等宽次按钮为“错题集 N”和
  “单独训练”；错题集打开现有风格的全屏覆盖层，桌面用可展开宽表，移动端改为卡片而非 980px 横滚。
- 主表推荐字段为“句子/翻译、错词数、来源、复习阶段、下次复习、错误次数、最近答错、操作”；
  不复制管理端的用户、模型、内部追溯 ID，也不展示“你的答案”。
- 展开区推荐包含“还原完整句/中文翻译、按空位的正确词与最近正误、固定复习时间轴、首次/最近答错、
  最近耗时、来源难度、最近 5 次结果”；不新增精确事件表，轨迹详情继续读不可变 answer records。
- 架构比较结论：动态 UNION 现有两套状态最快但语义割裂；新建第三张进度表边界最干净但会与现有
  schedule 双轨迁移；推荐扩展现有 `sentence_cloze_review_schedule` 为全来源持久句子进度，
  复用既有唯一键并完成后保留 stage 3。
- 为不破坏现有“开始答题”对尚未在完形中答错的 word-agent 载体句服务，到期取题不能只改查整句进度；
  应合并“整句进度到期”和“单词进度载体到期”，按 `cloze_item_id` 去重。错题集列表本身只查真正
  答错后建立的整句进度。
- 历史迁移不应把所有曾经答错的句子强制复活；推荐保留现有 active schedule，并仅把“最新一次仍错误”
  且缺进度的历史句保守回填为立即到期，其他历史作为答题记录保留但不推断阶段。
- 实施后句子进度和单词进度消费同一份 `ClozeAnswerComparison`，中间空字符串不再被前端或后端过滤，
  Unicode 全角字符与大小写按统一 NFKC 规则比较。
- `review-due` 的句子/单词两集合在限量前按题目 ID 去重；`stats.dueReviewTasks` 使用同一 CTE 计数，
  首页不再把最多 10 条的响应长度当成真实到期数量。
- 用户错题详情 DTO 和 SQL 均不投影 `answer_text/answers_json`、模型或 Provider；最近结果只保留时间、
  整句正误、耗时、入口和 answer/reveal 操作类型。
- 实际数据库迁移后当前没有历史句子 schedule 可回填；字段和唯一索引已存在，因此首次新版本错误提交
  会从 active stage 0 开始，不会混入推断历史阶段。
