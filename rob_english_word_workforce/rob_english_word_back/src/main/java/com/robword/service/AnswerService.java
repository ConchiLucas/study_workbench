package com.robword.service;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.robword.entity.Word;
import com.robword.netty.ChannelManager;
import com.robword.state.PlayerState;
import com.robword.state.PlayerStateManager;
import jakarta.annotation.PreDestroy;
import lombok.extern.slf4j.Slf4j;
import org.springframework.context.annotation.Lazy;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.stereotype.Service;

import java.util.*;
import java.util.concurrent.*;

/**
 * 答题服务
 * 后端出题、判题、推题，前端只做渲染
 */
@Service
@Slf4j
public class AnswerService {

    private final RedisTemplate<String, Object> redisTemplate;
    private final ChannelManager channelManager;
    private final PlayerStateManager stateManager;
    private final GameSettlementService settlementService;
    private final ObjectMapper objectMapper;

    private static final String ANSWER_STATE_KEY = "game:answer_state:";
    private static final String GAME_STATE_KEY = "game:state:";
    private static final int QUESTION_TIME_SECONDS = 4; // 每题4秒
    private static final String SOLO_TRAINING_MODE = "solo_training";

    // 每题超时定时器
    private final ScheduledExecutorService scheduler = Executors.newScheduledThreadPool(2);
    // roomId:userId -> 超时任务
    private final Map<String, ScheduledFuture<?>> questionTimers = new ConcurrentHashMap<>();

    public AnswerService(RedisTemplate<String, Object> redisTemplate,
                         ChannelManager channelManager,
                         PlayerStateManager stateManager,
                         @Lazy GameSettlementService settlementService,
                         ObjectMapper objectMapper) {
        this.redisTemplate = redisTemplate;
        this.channelManager = channelManager;
        this.stateManager = stateManager;
        this.settlementService = settlementService;
        this.objectMapper = objectMapper;
    }

    @PreDestroy
    public void shutdown() {
        log.info("Shutting down AnswerService scheduler...");
        scheduler.shutdownNow();
    }

    /**
     * 初始化答题阶段
     * 为每个玩家生成题目列表（单词 + 4选项），并推送第一题
     */
    @SuppressWarnings("unchecked")
    public void initAnswerPhase(Long roomId) {
        String gameKey = GAME_STATE_KEY + roomId;
        Map<String, Object> gameState = (Map<String, Object>) redisTemplate.opsForValue().get(gameKey);
        if (gameState == null) {
            log.error("Game state not found for room {}", roomId);
            return;
        }

        Number p1Num = (Number) gameState.get("player1Id");
        Number p2Num = (Number) gameState.get("player2Id");
        Long player1Id = p1Num.longValue();
        Long player2Id = p2Num.longValue();

        List<Map<String, Object>> allWordMaps = (List<Map<String, Object>>) gameState.get("allWords");
        List<Map<String, Object>> p1WordMaps = (List<Map<String, Object>>) gameState.get("player1Words");
        List<Map<String, Object>> p2WordMaps = (List<Map<String, Object>>) gameState.get("player2Words");

        // 为两个玩家分别生成题目（含4选项）
        List<Map<String, Object>> p1Questions = generateQuestions(p1WordMaps, allWordMaps);
        List<Map<String, Object>> p2Questions = generateQuestions(p2WordMaps, allWordMaps);

        // 构建答题状态
        Map<String, Object> answerState = new HashMap<>();

        Map<String, Object> p1State = new HashMap<>();
        p1State.put("userId", player1Id);
        p1State.put("questions", p1Questions);
        p1State.put("currentIndex", 0);
        p1State.put("score", 0);
        p1State.put("correctCount", 0);
        p1State.put("answeredCount", 0);
        p1State.put("finished", false);

        Map<String, Object> p2State = new HashMap<>();
        p2State.put("userId", player2Id);
        p2State.put("questions", p2Questions);
        p2State.put("currentIndex", 0);
        p2State.put("score", 0);
        p2State.put("correctCount", 0);
        p2State.put("answeredCount", 0);
        p2State.put("finished", false);

        answerState.put("player1", p1State);
        answerState.put("player2", p2State);
        answerState.put("player1Id", player1Id);
        answerState.put("player2Id", player2Id);
        answerState.put("roomId", roomId);
        answerState.put("mode", gameState.get("mode"));
        answerState.put("robotProfile", gameState.get("robotProfile"));

        // 存入 Redis
        String answerKey = ANSWER_STATE_KEY + roomId;
        redisTemplate.opsForValue().set(answerKey, answerState, 5, TimeUnit.MINUTES);

        // 推送第一题给两个玩家
        pushNextQuestion(roomId, player1Id);
        pushNextQuestion(roomId, player2Id);

        log.info("Answer phase initialized for room {}", roomId);
    }

    /**
     * 处理玩家提交答案
     */
    @SuppressWarnings("unchecked")
    public synchronized void submitAnswer(Long userId, Long roomId, int selectedIndex) {
        String answerKey = ANSWER_STATE_KEY + roomId;
        Map<String, Object> answerState = (Map<String, Object>) redisTemplate.opsForValue().get(answerKey);
        if (answerState == null) {
            log.warn("Answer state not found for room {}", roomId);
            return;
        }

        // 确定是 player1 还是 player2
        Map<String, Object> playerState = getPlayerState(answerState, userId);
        if (playerState == null) {
            log.warn("Player {} not found in answer state for room {}", userId, roomId);
            return;
        }

        Boolean finished = (Boolean) playerState.get("finished");
        if (Boolean.TRUE.equals(finished)) {
            log.warn("Player {} already finished in room {}", userId, roomId);
            return;
        }

        int currentIndex = ((Number) playerState.get("currentIndex")).intValue();
        List<Map<String, Object>> questions = (List<Map<String, Object>>) playerState.get("questions");

        if (currentIndex >= questions.size()) {
            return;
        }

        // 取消超时定时器
        cancelQuestionTimer(roomId, userId);

        // 取当前题目
        Map<String, Object> question = questions.get(currentIndex);
        int correctIndex = ((Number) question.get("correctIndex")).intValue();
        int wordDifficulty = ((Number) question.get("difficulty")).intValue();
        int answerTimeMs = calculateAnswerTimeMs(playerState);

        boolean isCorrect = selectedIndex == correctIndex;
        int questionScore = isCorrect ? wordDifficulty : 0;

        // 更新答题状态
        int score = ((Number) playerState.get("score")).intValue() + questionScore;
        int correctCount = ((Number) playerState.get("correctCount")).intValue() + (isCorrect ? 1 : 0);
        int answeredCount = ((Number) playerState.get("answeredCount")).intValue() + 1;

        playerState.put("score", score);
        playerState.put("correctCount", correctCount);
        playerState.put("answeredCount", answeredCount);
        playerState.put("currentIndex", currentIndex + 1);

        // 保存到 Redis
        redisTemplate.opsForValue().set(answerKey, answerState, 5, TimeUnit.MINUTES);

        // 推送答题结果给玩家
        if (!isSoloTrainingRobot(answerState, userId)) {
            channelManager.sendToPlayer(userId, "answer_result", Map.of(
                    "correct", isCorrect,
                    "correctIndex", correctIndex,
                    "score", questionScore,
                    "totalScore", score,
                    "selectedIndex", selectedIndex
            ));
        }

        // 推送对手进度
        Long opponentId = getOpponentId(answerState, userId);
        if (opponentId != null && !isSoloTrainingRobot(answerState, opponentId)) {
            channelManager.sendToPlayer(opponentId, "opponent_progress", Map.of(
                    "answeredCount", answeredCount,
                    "correctCount", correctCount,
                    "score", score
            ));
        }

        saveAnswerDetail(roomId, userId, question, selectedIndex, isCorrect, questionScore, answerTimeMs, answerState);

        // 检查是否还有下一题
        if (currentIndex + 1 < questions.size()) {
            // 延迟 1 秒后推送下一题（让前端显示答案结果）
            scheduler.schedule(() -> pushNextQuestion(roomId, userId), 1, TimeUnit.SECONDS);
        } else {
            // 答完所有题
            finishPlayer(roomId, userId, answerState);
        }
    }

    /**
     * 推送下一题给玩家
     */
    @SuppressWarnings("unchecked")
    private void pushNextQuestion(Long roomId, Long userId) {
        String answerKey = ANSWER_STATE_KEY + roomId;
        Map<String, Object> answerState = (Map<String, Object>) redisTemplate.opsForValue().get(answerKey);
        if (answerState == null) return;

        boolean robotPlayer = isSoloTrainingRobot(answerState, userId);
        if (!robotPlayer) {
            // 检查玩家是否仍在答题状态
            PlayerState state = stateManager.getState(userId);
            if (state != PlayerState.ANSWERING) {
                log.debug("Skipping pushNextQuestion for user {} (state={})", userId, state);
                return;
            }
        }

        Map<String, Object> playerState = getPlayerState(answerState, userId);
        if (playerState == null) return;

        Boolean finished = (Boolean) playerState.get("finished");
        if (Boolean.TRUE.equals(finished)) return;

        int currentIndex = ((Number) playerState.get("currentIndex")).intValue();
        List<Map<String, Object>> questions = (List<Map<String, Object>>) playerState.get("questions");

        if (currentIndex >= questions.size()) {
            finishPlayer(roomId, userId, answerState);
            return;
        }

        Map<String, Object> question = questions.get(currentIndex);
        markQuestionStart(answerKey, answerState, playerState);

        if (robotPlayer) {
            scheduleRobotAnswer(roomId, userId, currentIndex, question, answerState);
            return;
        }

        // 推题给前端（不包含正确答案索引，防作弊）
        channelManager.sendToPlayer(userId, "next_question", Map.of(
                "index", currentIndex,
                "total", questions.size(),
                "word", question.get("word"),
                "options", question.get("options"),
                "timeLeft", QUESTION_TIME_SECONDS
        ));

        // 启动此题超时倒计时
        startQuestionTimer(roomId, userId, currentIndex);

        log.debug("Pushed question {} to user {} in room {}", currentIndex, userId, roomId);
    }

    /**
     * 训练机器人按资质面板自动答题，不走 WebSocket。
     */
    private void scheduleRobotAnswer(Long roomId,
                                     Long robotId,
                                     int questionIndex,
                                     Map<String, Object> question,
                                     Map<String, Object> answerState) {
        String timerKey = roomId + ":" + robotId;
        cancelQuestionTimer(roomId, robotId);

        int selectedIndex = pickRobotAnswerIndex(question, answerState);
        long delayMs = ThreadLocalRandom.current().nextLong(650, 2200);
        ScheduledFuture<?> future = scheduler.schedule(
                () -> submitAnswer(robotId, roomId, selectedIndex),
                delayMs,
                TimeUnit.MILLISECONDS
        );
        questionTimers.put(timerKey, future);
        log.debug("Scheduled robot answer for room {}, question {}, delay={}ms", roomId, questionIndex, delayMs);
    }

    @SuppressWarnings("unchecked")
    private int pickRobotAnswerIndex(Map<String, Object> question, Map<String, Object> answerState) {
        Map<String, Object> profile = (Map<String, Object>) answerState.getOrDefault("robotProfile", Map.of());
        int correctIndex = ((Number) question.get("correctIndex")).intValue();
        int difficulty = ((Number) question.getOrDefault("difficulty", 1)).intValue();

        double baseAccuracy = numberValue(profile.get("baseAccuracy"), 0.62);
        double resistance = numberValue(profile.get("difficultyResistance"), 0.45);
        double volatility = numberValue(profile.get("volatility"), 0.12);
        double carelessRate = numberValue(profile.get("carelessRate"), 0.10);
        double burstRate = numberValue(profile.get("burstRate"), 0.06);

        double difficultyPressure = Math.max(0.0, Math.min(1.0, difficulty / 1000.0));
        double difficultyPenalty = difficultyPressure * (0.34 - resistance * 0.18);
        double roundNoise = ThreadLocalRandom.current().nextDouble(-volatility, volatility);
        double accuracy = baseAccuracy - difficultyPenalty + roundNoise;

        if (ThreadLocalRandom.current().nextDouble() < burstRate) {
            accuracy += ThreadLocalRandom.current().nextDouble(0.08, 0.18);
        }
        if (ThreadLocalRandom.current().nextDouble() < carelessRate) {
            accuracy -= ThreadLocalRandom.current().nextDouble(0.14, 0.30);
        }

        accuracy = Math.max(0.08, Math.min(0.96, accuracy));
        if (ThreadLocalRandom.current().nextDouble() < accuracy) {
            return correctIndex;
        }

        List<Integer> wrongIndexes = new ArrayList<>(List.of(1, 2, 3, 4));
        wrongIndexes.remove(Integer.valueOf(correctIndex));
        return wrongIndexes.get(ThreadLocalRandom.current().nextInt(wrongIndexes.size()));
    }

    /**
     * 启动每题超时定时器
     */
    private void startQuestionTimer(Long roomId, Long userId, int questionIndex) {
        String timerKey = roomId + ":" + userId;
        cancelQuestionTimer(roomId, userId);

        ScheduledFuture<?> future = scheduler.schedule(() -> {
            handleQuestionTimeout(roomId, userId, questionIndex);
        }, QUESTION_TIME_SECONDS, TimeUnit.SECONDS);

        questionTimers.put(timerKey, future);
    }

    /**
     * 取消超时定时器
     */
    private void cancelQuestionTimer(Long roomId, Long userId) {
        String timerKey = roomId + ":" + userId;
        ScheduledFuture<?> future = questionTimers.remove(timerKey);
        if (future != null) {
            future.cancel(false);
        }
    }

    private void markQuestionStart(String answerKey,
                                   Map<String, Object> answerState,
                                   Map<String, Object> playerState) {
        playerState.put("questionStartTimeMs", System.currentTimeMillis());
        redisTemplate.opsForValue().set(answerKey, answerState, 5, TimeUnit.MINUTES);
    }

    private int calculateAnswerTimeMs(Map<String, Object> playerState) {
        Number startedAtNum = (Number) playerState.get("questionStartTimeMs");
        if (startedAtNum == null) {
            return 0;
        }
        long elapsed = System.currentTimeMillis() - startedAtNum.longValue();
        return (int) Math.max(0, Math.min(elapsed, Integer.MAX_VALUE));
    }

    /**
     * 取消房间内所有玩家的超时定时器
     */
    public void cancelAllRoomTimers(Long roomId) {
        String prefix = roomId + ":";
        questionTimers.entrySet().removeIf(entry -> {
            if (entry.getKey().startsWith(prefix)) {
                entry.getValue().cancel(false);
                log.info("Cancelled timer for key: {}", entry.getKey());
                return true;
            }
            return false;
        });
    }

    /**
     * 处理超时未答
     */
    @SuppressWarnings("unchecked")
    private synchronized void handleQuestionTimeout(Long roomId, Long userId, int expectedIndex) {
        // 检查玩家是否仍在答题状态
        PlayerState state = stateManager.getState(userId);
        if (state != PlayerState.ANSWERING) {
            log.debug("Skipping question timeout for user {} (state={})", userId, state);
            return;
        }

        String answerKey = ANSWER_STATE_KEY + roomId;
        Map<String, Object> answerState = (Map<String, Object>) redisTemplate.opsForValue().get(answerKey);
        if (answerState == null) return;

        Map<String, Object> playerState = getPlayerState(answerState, userId);
        if (playerState == null) return;

        // 检查玩家是否已标记为完成（退出/断线）
        if (Boolean.TRUE.equals(playerState.get("finished"))) {
            log.debug("Skipping question timeout for user {} (already finished)", userId);
            return;
        }

        int currentIndex = ((Number) playerState.get("currentIndex")).intValue();
        if (currentIndex != expectedIndex) {
            // 已经回答过了，忽略
            return;
        }

        List<Map<String, Object>> questions = (List<Map<String, Object>>) playerState.get("questions");
        Map<String, Object> question = questions.get(currentIndex);
        int correctIndex = ((Number) question.get("correctIndex")).intValue();
        int answerTimeMs = calculateAnswerTimeMs(playerState);

        // 超时判错
        int answeredCount = ((Number) playerState.get("answeredCount")).intValue() + 1;
        playerState.put("answeredCount", answeredCount);
        playerState.put("currentIndex", currentIndex + 1);

        int score = ((Number) playerState.get("score")).intValue();
        int correctCount = ((Number) playerState.get("correctCount")).intValue();

        redisTemplate.opsForValue().set(answerKey, answerState, 5, TimeUnit.MINUTES);

        // 推送超时结果
        if (!isSoloTrainingRobot(answerState, userId)) {
            channelManager.sendToPlayer(userId, "question_timeout", Map.of(
                    "correctIndex", correctIndex,
                    "totalScore", score
            ));
        }

        // 推送对手进度
        Long opponentId = getOpponentId(answerState, userId);
        if (opponentId != null && !isSoloTrainingRobot(answerState, opponentId)) {
            channelManager.sendToPlayer(opponentId, "opponent_progress", Map.of(
                    "answeredCount", answeredCount,
                    "correctCount", correctCount,
                    "score", score
            ));
        }

        // 保存答题详情（超时未答）
        saveAnswerDetail(roomId, userId, question, null, false, 0, answerTimeMs, answerState);

        // 下一题
        if (currentIndex + 1 < questions.size()) {
            scheduler.schedule(() -> pushNextQuestion(roomId, userId), 1, TimeUnit.SECONDS);
        } else {
            finishPlayer(roomId, userId, answerState);
        }
    }

    /**
     * 玩家完成所有题目
     */
    @SuppressWarnings("unchecked")
    private void finishPlayer(Long roomId, Long userId, Map<String, Object> answerState) {
        Map<String, Object> playerState = getPlayerState(answerState, userId);
        if (playerState == null) return;

        playerState.put("finished", true);

        String answerKey = ANSWER_STATE_KEY + roomId;
        redisTemplate.opsForValue().set(answerKey, answerState, 5, TimeUnit.MINUTES);

        int score = ((Number) playerState.get("score")).intValue();

        // 通知对手此玩家已完成
        Long opponentId = getOpponentId(answerState, userId);
        if (opponentId != null && !isSoloTrainingRobot(answerState, opponentId)) {
            channelManager.sendToPlayer(opponentId, "opponent_finished", Map.of("score", score));
        }

        log.info("Player {} finished all questions in room {} with score {}", userId, roomId, score);

        // 检查双方是否都完成
        Map<String, Object> p1State = (Map<String, Object>) answerState.get("player1");
        Map<String, Object> p2State = (Map<String, Object>) answerState.get("player2");

        boolean p1Finished = Boolean.TRUE.equals(p1State.get("finished"));
        boolean p2Finished = Boolean.TRUE.equals(p2State.get("finished"));

        if (p1Finished && p2Finished) {
            log.info("Both players finished in room {}, triggering settlement", roomId);
            triggerSettlement(roomId, answerState);
        }
    }

    /**
     * 触发结算
     */
    @SuppressWarnings("unchecked")
    private void triggerSettlement(Long roomId, Map<String, Object> answerState) {
        Number p1IdNum = (Number) answerState.get("player1Id");
        Number p2IdNum = (Number) answerState.get("player2Id");
        Long player1Id = p1IdNum.longValue();
        Long player2Id = p2IdNum.longValue();

        Map<String, Object> p1State = (Map<String, Object>) answerState.get("player1");
        Map<String, Object> p2State = (Map<String, Object>) answerState.get("player2");

        // 如果双方都没有答过题（都退出了），跳过结算，不插入记录
        int p1Answered = ((Number) p1State.getOrDefault("answeredCount", 0)).intValue();
        int p2Answered = ((Number) p2State.getOrDefault("answeredCount", 0)).intValue();
        if (p1Answered == 0 && p2Answered == 0) {
            log.info("Both players exited without answering in room {}, skipping settlement", roomId);
            // 清理 Redis 状态
            redisTemplate.delete(ANSWER_STATE_KEY + roomId);
            // 清理已预创建的记录（如果有）
            Number recordIdNum = (Number) answerState.get("recordId");
            if (recordIdNum != null) {
                settlementService.deleteRecord(recordIdNum.longValue());
            }
            return;
        }

        // 构造 playerAnswers 给 settlement service
        Map<Long, Map<String, Object>> playerAnswers = new HashMap<>();
        playerAnswers.put(player1Id, p1State);
        playerAnswers.put(player2Id, p2State);

        try {
            if (isSoloTraining(answerState)) {
                Map<String, Object> result = settlementService.settleSoloTraining(roomId, answerState);
                channelManager.sendToPlayer(player1Id, "game_settlement", result);
                if (!stateManager.transition(player1Id, PlayerState.ANSWERING, PlayerState.FINISHED)) {
                    log.warn("CAS ANSWERING→FINISHED failed for solo training player {}, forcing state", player1Id);
                    stateManager.forceState(player1Id, PlayerState.FINISHED);
                }
                channelManager.sendToPlayer(player1Id, "state_change", Map.of("state", "FINISHED"));
                scheduler.schedule(() -> cleanup(roomId), 10, TimeUnit.SECONDS);
                log.info("Solo training settlement completed for room {}", roomId);
                return;
            }

            Map<String, Object> result = settlementService.settleGame(
                    roomId,
                    Set.of(player1Id, player2Id),
                    Set.of(),
                    playerAnswers
            );

            // 推送结算结果给双方
            channelManager.sendToPlayer(player1Id, "game_settlement", result);
            channelManager.sendToPlayer(player2Id, "game_settlement", result);

            // 转换状态: ANSWERING → FINISHED（CAS 失败时用 forceState 兜底，覆盖断线玩家状态被清理的场景）
            if (!stateManager.transition(player1Id, PlayerState.ANSWERING, PlayerState.FINISHED)) {
                log.warn("CAS ANSWERING→FINISHED failed for player {}, forcing state", player1Id);
                stateManager.forceState(player1Id, PlayerState.FINISHED);
            }
            if (!stateManager.transition(player2Id, PlayerState.ANSWERING, PlayerState.FINISHED)) {
                log.warn("CAS ANSWERING→FINISHED failed for player {}, forcing state", player2Id);
                stateManager.forceState(player2Id, PlayerState.FINISHED);
            }

            // 推送状态变更
            channelManager.sendToPlayer(player1Id, "state_change", Map.of("state", "FINISHED"));
            channelManager.sendToPlayer(player2Id, "state_change", Map.of("state", "FINISHED"));

            // 延迟清理
            scheduler.schedule(() -> cleanup(roomId), 10, TimeUnit.SECONDS);

            log.info("Settlement completed for room {}", roomId);
        } catch (Exception e) {
            log.error("Settlement failed for room {}: {}", roomId, e.getMessage(), e);
        }
    }

    /**
     * 清理答题状态
     */
    private void cleanup(Long roomId) {
        redisTemplate.delete(ANSWER_STATE_KEY + roomId);
        log.info("Answer state cleaned up for room {}", roomId);
    }

    /**
     * 处理玩家断线（游戏中）
     */
    @SuppressWarnings("unchecked")
    public void handlePlayerDisconnect(Long roomId, Long userId) {
        cancelQuestionTimer(roomId, userId);

        String answerKey = ANSWER_STATE_KEY + roomId;
        Map<String, Object> answerState = (Map<String, Object>) redisTemplate.opsForValue().get(answerKey);
        if (answerState == null) return;

        Map<String, Object> playerState = getPlayerState(answerState, userId);
        if (playerState != null && !Boolean.TRUE.equals(playerState.get("finished"))) {
            playerState.put("finished", true);
            redisTemplate.opsForValue().set(answerKey, answerState, 5, TimeUnit.MINUTES);

            // 检查是否双方都完成
            Map<String, Object> p1State = (Map<String, Object>) answerState.get("player1");
            Map<String, Object> p2State = (Map<String, Object>) answerState.get("player2");
            if (Boolean.TRUE.equals(p1State.get("finished")) && Boolean.TRUE.equals(p2State.get("finished"))) {
                triggerSettlement(roomId, answerState);
            }
        }
    }

    /**
     * 构建答题阶段恢复数据，供重连/同步状态时恢复页面
     */
    @SuppressWarnings("unchecked")
    public Map<String, Object> buildResumePayload(Long roomId, Long userId) {
        String answerKey = ANSWER_STATE_KEY + roomId;
        Map<String, Object> answerState = (Map<String, Object>) redisTemplate.opsForValue().get(answerKey);
        if (answerState == null) {
            return null;
        }

        Map<String, Object> playerState = getPlayerState(answerState, userId);
        if (playerState == null) {
            return null;
        }

        Long opponentId = getOpponentId(answerState, userId);
        Map<String, Object> opponentState = opponentId != null ? getPlayerState(answerState, opponentId) : null;

        List<Map<String, Object>> questions = (List<Map<String, Object>>) playerState.get("questions");
        int currentIndex = ((Number) playerState.getOrDefault("currentIndex", 0)).intValue();
        boolean finished = Boolean.TRUE.equals(playerState.get("finished"));

        Map<String, Object> payload = new HashMap<>();
        payload.put("roomId", roomId);
        payload.put("phase", "answer");
        payload.put("myScore", ((Number) playerState.getOrDefault("score", 0)).intValue());
        payload.put("myCorrectCount", ((Number) playerState.getOrDefault("correctCount", 0)).intValue());
        payload.put("myAnsweredCount", ((Number) playerState.getOrDefault("answeredCount", 0)).intValue());
        payload.put("opponentScore", opponentState != null ? ((Number) opponentState.getOrDefault("score", 0)).intValue() : 0);
        payload.put("opponentCorrectCount", opponentState != null ? ((Number) opponentState.getOrDefault("correctCount", 0)).intValue() : 0);
        payload.put("opponentAnsweredCount", opponentState != null ? ((Number) opponentState.getOrDefault("answeredCount", 0)).intValue() : 0);
        payload.put("questionTotal", questions != null ? questions.size() : 0);
        payload.put("questionIndex", currentIndex);
        payload.put("waitingForOpponent", finished);

        if (!finished && questions != null && currentIndex < questions.size()) {
            Map<String, Object> question = questions.get(currentIndex);
            payload.put("questionWord", question.get("word"));
            payload.put("questionOptions", question.get("options"));
            payload.put("questionTimeLeft", QUESTION_TIME_SECONDS);
        }

        return payload;
    }

    // ==================== 工具方法 ====================

    /**
     * 为玩家生成题目列表（每题 = 单词 + 4选项 + 正确答案索引）
     */
    @SuppressWarnings("unchecked")
    private List<Map<String, Object>> generateQuestions(List<Map<String, Object>> playerWords,
                                                        List<Map<String, Object>> allWords) {
        List<Map<String, Object>> questions = new ArrayList<>();
        Random random = new Random();

        for (Map<String, Object> wordMap : playerWords) {
            String word = (String) wordMap.get("word");
            String meaning = (String) wordMap.get("meaning");
            Number diffNum = (Number) wordMap.get("difficulty");
            Number idNum = (Number) wordMap.get("id");
            int difficulty = diffNum != null ? diffNum.intValue() : 1;
            long wordId = idNum != null ? idNum.longValue() : 0;

            // 生成 3 个干扰选项
            List<String> wrongOptions = new ArrayList<>();
            List<Map<String, Object>> shuffled = new ArrayList<>(allWords);
            Collections.shuffle(shuffled, random);

            for (Map<String, Object> other : shuffled) {
                if (wrongOptions.size() >= 3) break;
                String otherMeaning = (String) other.get("meaning");
                Number otherId = (Number) other.get("id");
                long otherIdVal = otherId != null ? otherId.longValue() : -1;
                if (otherIdVal != wordId && !otherMeaning.equals(meaning)) {
                    wrongOptions.add(otherMeaning);
                }
            }

            // 组合 4 个选项并打乱
            List<String> options = new ArrayList<>();
            options.add(meaning);
            options.addAll(wrongOptions);
            Collections.shuffle(options, random);

            // 计算正确答案索引（1-4）
            int correctIndex = options.indexOf(meaning) + 1;

            Map<String, Object> question = new HashMap<>();
            question.put("wordId", wordId);
            question.put("word", word);
            question.put("meaning", meaning);
            question.put("difficulty", difficulty);
            question.put("options", options);
            question.put("correctIndex", correctIndex);

            questions.add(question);
        }

        return questions;
    }

    private boolean isSoloTraining(Map<String, Object> answerState) {
        return SOLO_TRAINING_MODE.equals(answerState.get("mode"));
    }

    private boolean isSoloTrainingRobot(Map<String, Object> answerState, Long userId) {
        if (!isSoloTraining(answerState)) {
            return false;
        }
        Number p2IdNum = (Number) answerState.get("player2Id");
        return p2IdNum != null && userId.equals(p2IdNum.longValue());
    }

    private double numberValue(Object value, double fallback) {
        return value instanceof Number number ? number.doubleValue() : fallback;
    }

    @SuppressWarnings("unchecked")
    private Map<String, Object> getPlayerState(Map<String, Object> answerState, Long userId) {
        Number p1IdNum = (Number) answerState.get("player1Id");
        Long player1Id = p1IdNum != null ? p1IdNum.longValue() : null;

        if (userId.equals(player1Id)) {
            return (Map<String, Object>) answerState.get("player1");
        } else {
            return (Map<String, Object>) answerState.get("player2");
        }
    }

    private Long getOpponentId(Map<String, Object> answerState, Long userId) {
        Number p1IdNum = (Number) answerState.get("player1Id");
        Number p2IdNum = (Number) answerState.get("player2Id");
        Long player1Id = p1IdNum != null ? p1IdNum.longValue() : null;
        Long player2Id = p2IdNum != null ? p2IdNum.longValue() : null;

        return userId.equals(player1Id) ? player2Id : player1Id;
    }

    /**
     * 保存单条答题详情到数据库
     */
    @SuppressWarnings("unchecked")
    private void saveAnswerDetail(Long roomId, Long userId, Map<String, Object> question,
                                   Integer selectedIndex, boolean isCorrect, int score, int answerTimeMs,
                                   Map<String, Object> answerState) {
        try {
            // 获取 recordId
            Long recordId = settlementService.findExistingRecordId(roomId);
            if (recordId == null) {
                log.warn("No record found for room {}, cannot save answer detail", roomId);
                return;
            }

            List<String> options = (List<String>) question.get("options");
            int correctIndex = ((Number) question.get("correctIndex")).intValue();

            Map<String, Object> detail = new HashMap<>();
            detail.put("wordId", question.get("wordId"));
            detail.put("wordContent", question.get("word"));
            detail.put("wordDifficulty", question.get("difficulty"));
            detail.put("options", options);
            detail.put("correctAnswerIndex", correctIndex);
            detail.put("selectedAnswerIndex", selectedIndex);
            detail.put("isCorrect", isCorrect);
            detail.put("score", score);
            detail.put("answerTimeMs", answerTimeMs);
            detail.put("userName", resolveAnswerUserName(userId, answerState));

            settlementService.saveSingleAnswerDetail(recordId, userId, detail);
        } catch (Exception e) {
            log.error("Failed to save answer detail for user {} in room {}: {}", userId, roomId, e.getMessage());
        }
    }

    @SuppressWarnings("unchecked")
    private String resolveAnswerUserName(Long userId, Map<String, Object> answerState) {
        if (!isSoloTrainingRobot(answerState, userId)) {
            return null;
        }
        Map<String, Object> profile = (Map<String, Object>) answerState.getOrDefault("robotProfile", Map.of());
        Object name = profile.get("name");
        return name != null ? name.toString() : "训练机器人";
    }
}
