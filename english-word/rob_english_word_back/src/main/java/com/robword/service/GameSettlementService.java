package com.robword.service;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.robword.entity.GameAnswerDetail;
import com.robword.entity.GameRecord;
import com.robword.entity.Room;
import com.robword.entity.User;
import com.robword.entity.Word;
import com.robword.mapper.GameAnswerDetailMapper;
import com.robword.mapper.GameRecordMapper;
import com.robword.mapper.RoomMapper;
import com.robword.mapper.UserMapper;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.LocalDateTime;
import java.util.*;

@Service
@RequiredArgsConstructor
@Slf4j
public class GameSettlementService {

    private final RedisTemplate<String, Object> redisTemplate;
    private final RoomMapper roomMapper;
    private final UserMapper userMapper;
    private final GameRecordMapper gameRecordMapper;
    private final GameAnswerDetailMapper gameAnswerDetailMapper;
    private final UserWordMasteryService userWordMasteryService;

    @Autowired(required = false)
    private WrongWordAgentNotificationService wrongWordAgentNotificationService;

    @Autowired(required = false)
    private WrongWordReviewProgressService wrongWordReviewProgressService;

    private static final String GAME_STATE_KEY = "game:state:";
    private static final String USER_ROOM_KEY = "game:user_room:";
    private static final int PK_WORD_COUNT = 5;
    private static final String MATCH_MODE = "match";
    private static final String SOLO_TRAINING_MODE = "solo_training";
    private static final ObjectMapper JSON_MAPPER = new ObjectMapper();

    /**
     * 执行游戏结算
     *
     * @param roomId              房间ID
     * @param finishedPlayers     已完成答题的玩家集合
     * @param disconnectedPlayers 已断开的玩家集合
     * @param playerAnswers       玩家答题数据
     * @return 结算结果
     */
    @Transactional
    @SuppressWarnings("unchecked")
    public Map<String, Object> settleGame(Long roomId,
                                          Set<Long> finishedPlayers,
                                          Set<Long> disconnectedPlayers,
                                          Map<Long, Map<String, Object>> playerAnswers) {
        log.info("Starting settlement for room {}, finished={}, disconnected={}",
                roomId, finishedPlayers, disconnectedPlayers);

        // 1. 从Redis获取游戏状态
        String gameKey = GAME_STATE_KEY + roomId;
        Map<String, Object> gameState = (Map<String, Object>) redisTemplate.opsForValue().get(gameKey);

        if (gameState == null) {
            throw new RuntimeException("Game state not found for room: " + roomId);
        }

        // 2. 获取玩家信息
        Number player1IdNum = (Number) gameState.get("player1Id");
        Number player2IdNum = (Number) gameState.get("player2Id");
        Long player1Id = player1IdNum != null ? player1IdNum.longValue() : null;
        Long player2Id = player2IdNum != null ? player2IdNum.longValue() : null;

        if (player1Id == null || player2Id == null) {
            throw new RuntimeException("Player info not found in game state");
        }

        User player1 = userMapper.selectById(player1Id);
        User player2 = userMapper.selectById(player2Id);

        // 3. 获取双方单词列表（用于记录答题详情）
        List<Word> player1Words = (List<Word>) gameState.get("player1Words");
        List<Word> player2Words = (List<Word>) gameState.get("player2Words");

        // 6. 检查是否已存在游戏记录（答题阶段开始时创建的）
        GameRecord record = findExistingRecord(roomId);
        Long recordId;
        if (record == null) {
            // 没有预创建的记录，创建新记录（兼容旧逻辑）
            record = new GameRecord();
            record.setRoomId(roomId);
            record.setMode(MATCH_MODE);
            applyMatchDifficulty(record, gameState);

            // 玩家1信息
            record.setPlayer1Id(player1Id);
            record.setPlayer1Name(player1.getNickname());

            // 玩家2信息
            record.setPlayer2Id(player2Id);
            record.setPlayer2Name(player2.getNickname());

            // 时间记录
            Number startTimeNum = (Number) gameState.get("startTime");
            if (startTimeNum != null) {
                record.setStartTime(java.time.LocalDateTime.ofInstant(
                    java.time.Instant.ofEpochSecond(startTimeNum.longValue()),
                    java.time.ZoneId.systemDefault()));
            }

            gameRecordMapper.insert(record);
            recordId = record.getId();
            log.info("Game record created (fallback): id={}", recordId);
        } else {
            recordId = record.getId();
        }

        // 4. 计算双方得分和答题统计（从数据库中已插入的答题详情计算）
        PlayerResult player1Result = calculatePlayerResultFromDb(recordId, player1Id);
        PlayerResult player2Result = calculatePlayerResultFromDb(recordId, player2Id);
        double expectedSetMaxScore = resolveExpectedSetMaxScore(gameState, player1, player2);

        // 5. 判定胜负
        Long winnerId = null;
        boolean isDraw = false;

        if (player1Result.score > player2Result.score) {
            winnerId = player1Id;
        } else if (player2Result.score > player1Result.score) {
            winnerId = player2Id;
        } else {
            isDraw = true; // 平局
        }

        // 更新比赛记录（无论是新创建的还是预创建的）
        record.setMode(MATCH_MODE);
        applyMatchDifficulty(record, gameState);

        // 玩家1信息
        record.setPlayer1Score(player1Result.score);
        record.setPlayer1CorrectCount(player1Result.correctCount);
        record.setPlayer1TotalCount(player1Result.totalCount);

        // 玩家2信息
        record.setPlayer2Score(player2Result.score);
        record.setPlayer2CorrectCount(player2Result.correctCount);
        record.setPlayer2TotalCount(player2Result.totalCount);

        // 胜负结果
        record.setWinnerId(winnerId);
        record.setIsDraw(isDraw ? 1 : 0);

        // 时间记录
        record.setEndTime(LocalDateTime.now());
        record.setDurationSeconds(calculateDuration(gameState));

        gameRecordMapper.updateById(record);
        log.info("Game record updated: id={}", record.getId());

        // 7. 补全缺失的答题详情（如果有玩家没有实时提交的数据）
        saveRemainingAnswerDetails(recordId, player1Id, player1Words, playerAnswers.get(player1Id));
        saveRemainingAnswerDetails(recordId, player2Id, player2Words, playerAnswers.get(player2Id));

        // 8. 更新用户统计
        SettlementOutcome player1Outcome = updateUserStats(player1, player1Result, player2Result, expectedSetMaxScore, winnerId, isDraw);
        SettlementOutcome player2Outcome = updateUserStats(player2, player2Result, player1Result, expectedSetMaxScore, winnerId, isDraw);

        // 9. 清理Redis数据
        cleanupRedisData(roomId, player1Id, player2Id);

        // 10. 构建结算结果返回
        Map<String, Object> result = new HashMap<>();
        result.put("recordId", recordId);
        result.put("roomId", roomId);
        result.put("winnerId", winnerId);
        result.put("isDraw", isDraw);

        Map<String, Object> player1Data = new HashMap<>();
        player1Data.put("userId", player1Id);
        player1Data.put("nickname", player1.getNickname());
        player1Data.put("score", player1Result.score);
        player1Data.put("correctCount", player1Result.correctCount);
        player1Data.put("totalCount", player1Result.totalCount);
        player1Data.put("expChange", player1Outcome.expChange);
        player1Data.put("completionRate", player1Outcome.completionRate);

        Map<String, Object> player2Data = new HashMap<>();
        player2Data.put("userId", player2Id);
        player2Data.put("nickname", player2.getNickname());
        player2Data.put("score", player2Result.score);
        player2Data.put("correctCount", player2Result.correctCount);
        player2Data.put("totalCount", player2Result.totalCount);
        player2Data.put("expChange", player2Outcome.expChange);
        player2Data.put("completionRate", player2Outcome.completionRate);

        result.put("player1", player1Data);
        result.put("player2", player2Data);

        log.info("Settlement completed for room {}: winner={}, score {}:{}",
                roomId, winnerId, player1Result.score, player2Result.score);

        return result;
    }

    /**
     * 单人训练结算。训练经验和正式匹配经验完全隔离。
     */
    @Transactional
    @SuppressWarnings("unchecked")
    public Map<String, Object> settleSoloTraining(Long roomId, Map<String, Object> answerState) {
        String gameKey = GAME_STATE_KEY + roomId;
        Map<String, Object> gameState = (Map<String, Object>) redisTemplate.opsForValue().get(gameKey);
        if (gameState == null) {
            throw new RuntimeException("Game state not found for solo training room: " + roomId);
        }

        Number playerIdNum = (Number) answerState.get("player1Id");
        Number robotIdNum = (Number) answerState.get("player2Id");
        Long playerId = playerIdNum.longValue();
        Long robotId = robotIdNum.longValue();

        Map<String, Object> playerState = (Map<String, Object>) answerState.get("player1");
        Map<String, Object> robotState = (Map<String, Object>) answerState.get("player2");
        Map<String, Object> robotProfile = (Map<String, Object>) gameState.getOrDefault("robotProfile", Map.of());

        PlayerResult playerResult = calculatePlayerResultFromState(playerId, playerState);
        PlayerResult robotResult = calculatePlayerResultFromState(robotId, robotState);
        GameRecord record = findExistingRecord(roomId);
        if (record == null) {
            Long recordId = createInitialSoloTrainingRecord(roomId);
            record = recordId != null ? gameRecordMapper.selectById(recordId) : null;
        }

        Long winnerId = null;
        boolean isDraw = false;
        if (playerResult.score > robotResult.score) {
            winnerId = playerId;
        } else if (robotResult.score > playerResult.score) {
            winnerId = robotId;
        } else {
            isDraw = true;
        }

        User player = userMapper.selectById(playerId);
        int expChange = calculateSoloTrainingExp(playerResult, robotProfile, playerId.equals(winnerId), isDraw);
        updateSoloTrainingStats(player, expChange, playerId.equals(winnerId));

        Long recordId = record != null ? record.getId() : null;
        if (record != null) {
            record.setMode(SOLO_TRAINING_MODE);
            record.setPlayer1Score(playerResult.score);
            record.setPlayer1CorrectCount(playerResult.correctCount);
            record.setPlayer1TotalCount(playerResult.totalCount);
            record.setPlayer2Score(robotResult.score);
            record.setPlayer2CorrectCount(robotResult.correctCount);
            record.setPlayer2TotalCount(robotResult.totalCount);
            record.setWinnerId(winnerId);
            record.setIsDraw(isDraw ? 1 : 0);
            record.setEndTime(LocalDateTime.now());
            record.setDurationSeconds(calculateDuration(gameState));
            record.setTrainingExpChange(expChange);
            record.setTrainingRankAfter(player != null ? safeInt(player.getTrainingRank(), 1) : null);
            applyTrainingDifficulty(record, gameState);
            applyRobotProfile(record, robotProfile);
            gameRecordMapper.updateById(record);
            log.info("Solo training record updated: id={}", recordId);
        }

        cleanupRedisData(roomId, playerId, robotId);

        Map<String, Object> result = new HashMap<>();
        result.put("recordId", recordId);
        result.put("mode", "solo_training");
        result.put("roomId", roomId);
        result.put("winnerId", winnerId);
        result.put("isDraw", isDraw);
        result.put("trainingExpChange", expChange);
        result.put("trainingExp", player != null ? safeInt(player.getTrainingExp(), 0) : 0);
        result.put("trainingRank", player != null ? safeInt(player.getTrainingRank(), 1) : 1);
        result.put("robotProfile", robotProfile);

        Map<String, Object> playerData = new HashMap<>();
        playerData.put("userId", playerId);
        playerData.put("nickname", player != null ? player.getNickname() : "玩家");
        playerData.put("score", playerResult.score);
        playerData.put("correctCount", playerResult.correctCount);
        playerData.put("totalCount", playerResult.totalCount);
        playerData.put("expChange", expChange);

        Map<String, Object> robotData = new HashMap<>();
        robotData.put("userId", robotId);
        robotData.put("nickname", robotName(robotProfile));
        robotData.put("score", robotResult.score);
        robotData.put("correctCount", robotResult.correctCount);
        robotData.put("totalCount", robotResult.totalCount);
        robotData.put("expChange", 0);

        result.put("player1", playerData);
        result.put("player2", robotData);

        log.info("Solo training settled for room {}: player={}, robot={}, expChange={}",
                roomId, playerResult.score, robotResult.score, expChange);
        return result;
    }

    /**
     * 创建初始游戏记录（答题阶段开始时调用）
     */
    @Transactional
    @SuppressWarnings("unchecked")
    public Long createInitialGameRecord(Long roomId) {
        log.info("Creating initial game record for room: {}", roomId);

        // 从Redis获取游戏状态
        String gameKey = GAME_STATE_KEY + roomId;
        Map<String, Object> gameState = (Map<String, Object>) redisTemplate.opsForValue().get(gameKey);

        if (gameState == null) {
            log.error("Game state not found for room: {}. Cannot create game record.", roomId);
            return null;
        }

        log.debug("Game state found for room {}: keys={}", roomId, gameState.keySet());

        // 获取玩家信息
        Number player1IdNum = (Number) gameState.get("player1Id");
        Number player2IdNum = (Number) gameState.get("player2Id");
        Long player1Id = player1IdNum != null ? player1IdNum.longValue() : null;
        Long player2Id = player2IdNum != null ? player2IdNum.longValue() : null;

        log.info("Room {} player info: player1Id={}, player2Id={}", roomId, player1Id, player2Id);

        if (player1Id == null || player2Id == null) {
            log.error("Player info not found in game state for room: {}. player1Id={}, player2Id={}",
                    roomId, player1Id, player2Id);
            return null;
        }

        User player1 = userMapper.selectById(player1Id);
        User player2 = userMapper.selectById(player2Id);

        if (player1 == null || player2 == null) {
            log.error("User not found in database: player1Id={}, player2Id={}", player1Id, player2Id);
            return null;
        }

        // 创建初始比赛记录（分数为0，状态为进行中）
        GameRecord record = new GameRecord();
        record.setRoomId(roomId);
        record.setMode(MATCH_MODE);
        applyMatchDifficulty(record, gameState);

        // 玩家1信息
        record.setPlayer1Id(player1Id);
        record.setPlayer1Name(player1.getNickname());
        record.setPlayer1Score(0);
        record.setPlayer1CorrectCount(0);
        record.setPlayer1TotalCount(0);

        // 玩家2信息
        record.setPlayer2Id(player2Id);
        record.setPlayer2Name(player2.getNickname());
        record.setPlayer2Score(0);
        record.setPlayer2CorrectCount(0);
        record.setPlayer2TotalCount(0);

        // 胜负结果（未确定）
        record.setWinnerId(null);
        record.setIsDraw(0);

        // 时间记录
        Number startTimeNum = (Number) gameState.get("startTime");
        if (startTimeNum != null) {
            record.setStartTime(java.time.LocalDateTime.ofInstant(
                java.time.Instant.ofEpochSecond(startTimeNum.longValue()),
                java.time.ZoneId.systemDefault()));
        }
        record.setEndTime(null); // 未结束
        record.setDurationSeconds(0);

        try {
            gameRecordMapper.insert(record);
            log.info("Initial game record created successfully: id={}, roomId={}, player1Id={}, player2Id={}",
                    record.getId(), roomId, player1Id, player2Id);
            return record.getId();
        } catch (Exception e) {
            log.error("Failed to insert game record for room {}: {}", roomId, e.getMessage(), e);
            return null;
        }
    }

    /**
     * 创建单人训练初始记录，供答题详情实时落库使用。
     */
    @Transactional
    @SuppressWarnings("unchecked")
    public Long createInitialSoloTrainingRecord(Long roomId) {
        log.info("Creating initial solo training record for room: {}", roomId);

        String gameKey = GAME_STATE_KEY + roomId;
        Map<String, Object> gameState = (Map<String, Object>) redisTemplate.opsForValue().get(gameKey);
        if (gameState == null) {
            log.error("Game state not found for solo training room: {}. Cannot create record.", roomId);
            return null;
        }

        Number playerIdNum = (Number) gameState.get("player1Id");
        Number robotIdNum = (Number) gameState.get("player2Id");
        Long playerId = playerIdNum != null ? playerIdNum.longValue() : null;
        Long robotId = robotIdNum != null ? robotIdNum.longValue() : null;
        if (playerId == null || robotId == null) {
            log.error("Player info not found in solo training state for room: {}", roomId);
            return null;
        }

        User player = userMapper.selectById(playerId);
        if (player == null) {
            log.error("Training player not found: playerId={}", playerId);
            return null;
        }

        Map<String, Object> robotProfile =
                (Map<String, Object>) gameState.getOrDefault("robotProfile", Map.of());

        GameRecord record = new GameRecord();
        record.setRoomId(roomId);
        record.setMode(SOLO_TRAINING_MODE);
        record.setPlayer1Id(playerId);
        record.setPlayer1Name(player.getNickname());
        record.setPlayer1Score(0);
        record.setPlayer1CorrectCount(0);
        record.setPlayer1TotalCount(0);
        record.setPlayer2Id(robotId);
        record.setPlayer2Name(robotName(robotProfile));
        record.setPlayer2Score(0);
        record.setPlayer2CorrectCount(0);
        record.setPlayer2TotalCount(0);
        record.setWinnerId(null);
        record.setIsDraw(0);
        record.setTrainingExpChange(0);
        record.setTrainingRankAfter(safeInt(player.getTrainingRank(), 1));
        applyTrainingDifficulty(record, gameState);
        applyRobotProfile(record, robotProfile);

        Number startTimeNum = (Number) gameState.get("startTime");
        if (startTimeNum != null) {
            record.setStartTime(java.time.LocalDateTime.ofInstant(
                    java.time.Instant.ofEpochSecond(startTimeNum.longValue()),
                    java.time.ZoneId.systemDefault()));
        }
        record.setEndTime(null);
        record.setDurationSeconds(0);

        try {
            gameRecordMapper.insert(record);
            log.info("Initial solo training record created: id={}, roomId={}, playerId={}",
                    record.getId(), roomId, playerId);
            return record.getId();
        } catch (Exception e) {
            log.error("Failed to insert solo training record for room {}: {}", roomId, e.getMessage(), e);
            return null;
        }
    }

    /**
     * 保存单条答题详情（立即插入数据库）
     */
    @Transactional
    @SuppressWarnings("unchecked")
    public void saveSingleAnswerDetail(Long recordId, Long userId, Map<String, Object> answerDetail) {
        if (recordId == null || userId == null || answerDetail == null) {
            log.warn("Invalid parameters for saveSingleAnswerDetail: recordId={}, userId={}", recordId, userId);
            return;
        }

        User user = userMapper.selectById(userId);

        GameAnswerDetail detail = new GameAnswerDetail();
        detail.setRecordId(recordId);
        detail.setUserId(userId);
        Object fallbackUserName = answerDetail.get("userName");
        detail.setUserName(user != null ? user.getNickname()
                : (fallbackUserName != null ? fallbackUserName.toString() : null));

        // 轮次序号（从answerDetail获取，如果没有则默认为1）
        Integer roundNo = (Integer) answerDetail.get("roundNo");
        if (roundNo == null) {
            // 查询当前用户已答多少题，+1作为本轮次号
            roundNo = countUserAnswers(recordId, userId) + 1;
        }
        detail.setRoundNo(roundNo);

        // 获取单词信息
        Number wordIdNum = (Number) answerDetail.get("wordId");
        Long wordId = wordIdNum != null ? wordIdNum.longValue() : null;
        detail.setWordId(wordId);

        // 单词内容（前端直接传入）
        String wordContent = (String) answerDetail.get("wordContent");
        detail.setWordContent(wordContent);

        // 单词难度（前端直接传入）
        Number wordDifficultyNum = (Number) answerDetail.get("wordDifficulty");
        detail.setWordDifficulty(wordDifficultyNum != null ? wordDifficultyNum.intValue() : 0);

        // 四个选项（前端直接传入数组）
        List<String> options = (List<String>) answerDetail.get("options");
        if (options != null && options.size() >= 4) {
            detail.setOption1(options.get(0));
            detail.setOption2(options.get(1));
            detail.setOption3(options.get(2));
            detail.setOption4(options.get(3));
        }

        // 正确答案索引（1-4，前端传入）
        Number correctAnswerIndexNum = (Number) answerDetail.get("correctAnswerIndex");
        detail.setCorrectAnswerIndex(correctAnswerIndexNum != null ? correctAnswerIndexNum.intValue() : null);

        // 玩家选择的答案索引（1-4，未选择为null，前端传入）
        Number selectedAnswerIndexNum = (Number) answerDetail.get("selectedAnswerIndex");
        detail.setSelectedAnswerIndex(selectedAnswerIndexNum != null ? selectedAnswerIndexNum.intValue() : null);

        // 答题结果
        Boolean isCorrect = (Boolean) answerDetail.get("isCorrect");
        detail.setIsCorrect(isCorrect != null && isCorrect ? 1 : 0);

        Number scoreNum = (Number) answerDetail.get("score");
        detail.setScore(scoreNum != null ? scoreNum.intValue() : 0);

        Number answerTimeMsNum = (Number) answerDetail.get("answerTimeMs");
        detail.setAnswerTimeMs(answerTimeMsNum != null ? answerTimeMsNum.intValue() : 0);

        gameAnswerDetailMapper.insert(detail);
        recordMasteryProgress(recordId, detail);
        notifyWrongAnswer(detail);
        log.debug("Single answer detail saved: recordId={}, userId={}, roundNo={}, wordId={}",
                recordId, userId, roundNo, wordId);
    }

    /**
     * 统计用户已答多少题
     */
    private int countUserAnswers(Long recordId, Long userId) {
        return gameAnswerDetailMapper.selectCount(
            new com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper<GameAnswerDetail>()
                .eq(GameAnswerDetail::getRecordId, recordId)
                .eq(GameAnswerDetail::getUserId, userId)
        ).intValue();
    }

    /**
     * 从Redis游戏状态中获取单词信息
     */
    @SuppressWarnings("unchecked")
    private Word getWordFromGameState(Long recordId, Long userId, Long wordId) {
        // 从recordId反推roomId（通过查询game_record表）
        GameRecord record = gameRecordMapper.selectById(recordId);
        if (record == null) {
            return null;
        }
        Long roomId = record.getRoomId();

        // 从Redis获取游戏状态
        String gameKey = GAME_STATE_KEY + roomId;
        Map<String, Object> gameState = (Map<String, Object>) redisTemplate.opsForValue().get(gameKey);
        if (gameState == null) {
            return null;
        }

        // 确定是玩家1还是玩家2
        Number player1IdNum = (Number) gameState.get("player1Id");
        Long player1Id = player1IdNum != null ? player1IdNum.longValue() : null;

        String wordsKey = userId.equals(player1Id) ? "player1Words" : "player2Words";
        List<Word> words = (List<Word>) gameState.get(wordsKey);

        if (words != null) {
            return words.stream()
                    .filter(w -> w.getId().equals(wordId))
                    .findFirst()
                    .orElse(null);
        }
        return null;
    }

    /**
     * 更新游戏记录的最终结果（结算时调用）
     */
    @Transactional
    @SuppressWarnings("unchecked")
    public void updateGameRecord(Long recordId, Long roomId, Set<Long> finishedPlayers,
                                  Set<Long> disconnectedPlayers,
                                  Map<Long, Map<String, Object>> playerAnswers) {
        // 从Redis获取游戏状态
        String gameKey = GAME_STATE_KEY + roomId;
        Map<String, Object> gameState = (Map<String, Object>) redisTemplate.opsForValue().get(gameKey);

        if (gameState == null) {
            throw new RuntimeException("Game state not found for room: " + roomId);
        }

        // 获取玩家信息
        Number player1IdNum = (Number) gameState.get("player1Id");
        Number player2IdNum = (Number) gameState.get("player2Id");
        Long player1Id = player1IdNum != null ? player1IdNum.longValue() : null;
        Long player2Id = player2IdNum != null ? player2IdNum.longValue() : null;

        if (player1Id == null || player2Id == null) {
            throw new RuntimeException("Player info not found in game state");
        }

        // 计算双方得分和答题统计
        PlayerResult player1Result = calculatePlayerResultFromDb(recordId, player1Id);
        PlayerResult player2Result = calculatePlayerResultFromDb(recordId, player2Id);

        // 判定胜负
        Long winnerId = null;
        boolean isDraw = false;

        if (player1Result.score > player2Result.score) {
            winnerId = player1Id;
        } else if (player2Result.score > player1Result.score) {
            winnerId = player2Id;
        } else {
            isDraw = true;
        }

        // 更新比赛记录
        GameRecord record = gameRecordMapper.selectById(recordId);
        if (record == null) {
            throw new RuntimeException("Game record not found: " + recordId);
        }

        record.setMode(MATCH_MODE);

        // 更新玩家1信息
        record.setPlayer1Score(player1Result.score);
        record.setPlayer1CorrectCount(player1Result.correctCount);
        record.setPlayer1TotalCount(player1Result.totalCount);

        // 更新玩家2信息
        record.setPlayer2Score(player2Result.score);
        record.setPlayer2CorrectCount(player2Result.correctCount);
        record.setPlayer2TotalCount(player2Result.totalCount);

        // 更新胜负结果
        record.setWinnerId(winnerId);
        record.setIsDraw(isDraw ? 1 : 0);

        // 更新时间记录
        record.setEndTime(LocalDateTime.now());
        record.setDurationSeconds(calculateDuration(gameState));

        gameRecordMapper.updateById(record);
        log.info("Game record updated: id={}", record.getId());

        // 更新用户统计
        User player1 = userMapper.selectById(player1Id);
        User player2 = userMapper.selectById(player2Id);
        double expectedSetMaxScore = resolveExpectedSetMaxScore(gameState, player1, player2);
        updateUserStats(player1, player1Result, player2Result, expectedSetMaxScore, winnerId, isDraw);
        updateUserStats(player2, player2Result, player1Result, expectedSetMaxScore, winnerId, isDraw);
    }

    /**
     * 从数据库计算玩家答题结果（根据已插入的答题详情）
     */
    private PlayerResult calculatePlayerResultFromDb(Long recordId, Long userId) {
        PlayerResult result = new PlayerResult();
        result.userId = userId;

        // 从数据库查询该用户的所有答题记录
        List<GameAnswerDetail> details = gameAnswerDetailMapper.selectList(
            new com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper<GameAnswerDetail>()
                .eq(GameAnswerDetail::getRecordId, recordId)
                .eq(GameAnswerDetail::getUserId, userId)
        );

        result.totalCount = details.size();
        result.correctCount = (int) details.stream()
                .filter(d -> d.getIsCorrect() != null && d.getIsCorrect() == 1)
                .count();
        result.score = details.stream()
                .mapToInt(d -> d.getScore() != null ? d.getScore() : 0)
                .sum();
        result.setMaxScore = details.stream()
                .mapToInt(d -> d.getWordDifficulty() != null ? d.getWordDifficulty() : 1)
                .sum();

        return result;
    }

    /**
     * 保存答题详情
     */
    @SuppressWarnings("unchecked")
    private void saveAnswerDetails(Long recordId, Long userId, List<Word> words,
                                   Map<String, Object> answerData) {
        if (answerData == null) {
            return;
        }

        User user = userMapper.selectById(userId);
        List<Map<String, Object>> answers = (List<Map<String, Object>>) answerData.get("answers");

        if (answers == null) {
            return;
        }

        int roundNo = 1;
        for (Map<String, Object> answer : answers) {
            GameAnswerDetail detail = new GameAnswerDetail();
            detail.setRecordId(recordId);
            detail.setUserId(userId);
            detail.setUserName(user != null ? user.getNickname() : null);
            detail.setRoundNo(roundNo++);

            // 获取单词信息
            Number wordIdNum = (Number) answer.get("wordId");
            Long wordId = wordIdNum != null ? wordIdNum.longValue() : null;
            detail.setWordId(wordId);

            // 单词内容（前端传入）
            detail.setWordContent((String) answer.get("wordContent"));

            // 单词难度（前端传入）
            Number difficultyNum = (Number) answer.get("wordDifficulty");
            detail.setWordDifficulty(difficultyNum != null ? difficultyNum.intValue() : 0);

            // 四个选项（前端传入数组）
            List<String> options = (List<String>) answer.get("options");
            if (options != null && options.size() >= 4) {
                detail.setOption1(options.get(0));
                detail.setOption2(options.get(1));
                detail.setOption3(options.get(2));
                detail.setOption4(options.get(3));
            }

            // 正确答案索引和选择索引（前端传入）
            Number correctIndexNum = (Number) answer.get("correctAnswerIndex");
            detail.setCorrectAnswerIndex(correctIndexNum != null ? correctIndexNum.intValue() : null);

            Number selectedIndexNum = (Number) answer.get("selectedAnswerIndex");
            detail.setSelectedAnswerIndex(selectedIndexNum != null ? selectedIndexNum.intValue() : null);

            Boolean isCorrect = (Boolean) answer.get("isCorrect");
            detail.setIsCorrect(isCorrect != null && isCorrect ? 1 : 0);

            Number scoreNum = (Number) answer.get("score");
            detail.setScore(scoreNum != null ? scoreNum.intValue() : 0);

            Number answerTimeMsNum = (Number) answer.get("answerTimeMs");
            detail.setAnswerTimeMs(answerTimeMsNum != null ? answerTimeMsNum.intValue() : 0);

            gameAnswerDetailMapper.insert(detail);
            recordMasteryProgress(recordId, detail);
            notifyWrongAnswer(detail);
        }
    }

    /**
     * 更新用户统计数据
     */
    SettlementOutcome updateUserStats(User user,
                                      PlayerResult result,
                                      PlayerResult opponentResult,
                                      double expectedSetMaxScore,
                                      Long winnerId,
                                      boolean isDraw) {
        if (user == null) {
            return SettlementOutcome.empty();
        }

        // 更新总对战数
        user.setTotalGames(user.getTotalGames() + 1);

        boolean isWinner = !isDraw && winnerId != null && winnerId.equals(user.getId());
        boolean isLoser = !isDraw && !isWinner;

        // 更新胜场
        if (isWinner) {
            user.setTotalWins(user.getTotalWins() + 1);
        }

        double completionRate = com.robword.util.PKAlgorithmUtil.calculateCompletionRate(result.score, result.setMaxScore);
        double challengeIndex = com.robword.util.PKAlgorithmUtil.calculateChallengeIndex(result.setMaxScore, expectedSetMaxScore);
        int masteryExp = com.robword.util.PKAlgorithmUtil.calculateMasteryExp(completionRate, challengeIndex);

        double gapRate = com.robword.util.PKAlgorithmUtil.calculateGapRate(
                result.score, opponentResult.score, result.setMaxScore, opponentResult.setMaxScore);
        int battleExp = com.robword.util.PKAlgorithmUtil.calculateBattleExp(isDraw, isWinner, gapRate);

        double opponentCompletionRate = com.robword.util.PKAlgorithmUtil.calculateCompletionRate(
                opponentResult.score, opponentResult.setMaxScore);
        com.robword.util.PKAlgorithmUtil.BattleAdjustment battleAdjustment =
                com.robword.util.PKAlgorithmUtil.adjustBattleExpForQuality(battleExp, completionRate, opponentCompletionRate);
        int adjustedBattleExp = battleAdjustment.adjustedBattleExp();

        int currentStreak = getCurrentWinStreak(user);
        boolean effectiveWin = com.robword.util.PKAlgorithmUtil.isEffectiveWin(
                isWinner, completionRate, masteryExp, adjustedBattleExp);
        int effectiveStreak = effectiveWin ? currentStreak + 1 : 0;
        int streakExp = com.robword.util.PKAlgorithmUtil.calculateStreakExp(
                effectiveStreak, completionRate, challengeIndex, battleAdjustment.extremeLowQuality());

        int expChange = masteryExp + adjustedBattleExp + streakExp;
        expChange = com.robword.util.PKAlgorithmUtil.applyLoserProtection(
                expChange, isLoser, completionRate, challengeIndex);
        expChange = com.robword.util.PKAlgorithmUtil.applyWinnerCap(
                expChange, isWinner, battleAdjustment.extremeLowQuality());

        updateCurrentWinStreak(user, effectiveStreak);

        int newExp = user.getExp() + expChange;
        if (newExp < 0) newExp = 0; // 防止经验掉成负数
        user.setExp(newExp);

        // 梯队化升级动态推断
        int newRank = com.robword.util.PKAlgorithmUtil.calculateLevelFromTotalExp(newExp);
        if (newRank != user.getRank()) {
            user.setRank(newRank);
        }

        userMapper.updateById(user);
        log.info("User stats updated: id={}, totalGames={}, totalWins={}, exp={}, level={}, expChange={}, completionRate={}, challengeIndex={}",
                user.getId(), user.getTotalGames(), user.getTotalWins(), user.getExp(), user.getRank(),
                expChange, completionRate, challengeIndex);
        return new SettlementOutcome(masteryExp, adjustedBattleExp, streakExp, expChange,
                completionRate, challengeIndex, effectiveStreak, effectiveWin);
    }

    @SuppressWarnings("unchecked")
    private PlayerResult calculatePlayerResultFromState(Long userId, Map<String, Object> state) {
        PlayerResult result = new PlayerResult();
        result.userId = userId;
        result.score = ((Number) state.getOrDefault("score", 0)).intValue();
        result.correctCount = ((Number) state.getOrDefault("correctCount", 0)).intValue();
        result.totalCount = ((Number) state.getOrDefault("answeredCount", 0)).intValue();

        List<Map<String, Object>> questions = (List<Map<String, Object>>) state.getOrDefault("questions", List.of());
        result.setMaxScore = questions.stream()
                .map(q -> (Number) q.getOrDefault("difficulty", 0))
                .mapToInt(Number::intValue)
                .sum();
        return result;
    }

    private int calculateSoloTrainingExp(PlayerResult result,
                                         Map<String, Object> robotProfile,
                                         boolean isWinner,
                                         boolean isDraw) {
        if (result.totalCount <= 0) {
            return 0;
        }

        double completionRate = com.robword.util.PKAlgorithmUtil.calculateCompletionRate(result.score, result.setMaxScore);
        double robotMultiplier = numberValue(robotProfile.get("challengeMultiplier"), 1.0);
        int masteryExp = (int) Math.round((result.score / 80.0) * (0.6 + completionRate));
        int battleBonus = isWinner ? (int) Math.round(10 * robotMultiplier) : 0;
        int drawBonus = isDraw ? (int) Math.round(4 * robotMultiplier) : 0;
        int floor = result.score > 0 ? 3 : 1;
        return Math.max(floor, masteryExp + battleBonus + drawBonus);
    }

    private void updateSoloTrainingStats(User user, int expChange, boolean isWinner) {
        if (user == null) {
            return;
        }

        int totalGames = safeInt(user.getTrainingTotalGames(), 0) + 1;
        int totalWins = safeInt(user.getTrainingTotalWins(), 0) + (isWinner ? 1 : 0);
        int totalExp = Math.max(0, safeInt(user.getTrainingExp(), 0) + expChange);
        int trainingRank = com.robword.util.PKAlgorithmUtil.calculateLevelFromTotalExp(totalExp);

        user.setTrainingTotalGames(totalGames);
        user.setTrainingTotalWins(totalWins);
        user.setTrainingExp(totalExp);
        user.setTrainingRank(trainingRank);
        userMapper.updateById(user);
    }

    private String robotName(Map<String, Object> robotProfile) {
        Object name = robotProfile.get("name");
        return name != null ? name.toString() : "训练机器人";
    }

    private void applyRobotProfile(GameRecord record, Map<String, Object> robotProfile) {
        if (record == null || robotProfile == null) {
            return;
        }
        record.setRobotTier(stringValue(robotProfile.get("tier")));
        record.setRobotAptitude(intValue(robotProfile.get("aptitude")));
        record.setRobotGrowth(doubleValue(robotProfile.get("growth")));
        try {
            record.setRobotProfileJson(JSON_MAPPER.writeValueAsString(robotProfile));
        } catch (Exception e) {
            log.warn("Failed to serialize robot profile for record {}: {}", record.getId(), e.getMessage());
            record.setRobotProfileJson(robotProfile.toString());
        }
    }

    private void applyTrainingDifficulty(GameRecord record, Map<String, Object> gameState) {
        if (record == null || gameState == null) {
            return;
        }
        record.setTrainingDifficultyGroup(stringValue(gameState.get("trainingDifficultyGroup")));
        record.setTrainingDifficultyLevel(stringValue(gameState.get("trainingDifficultyLevel")));
    }

    private void applyMatchDifficulty(GameRecord record, Map<String, Object> gameState) {
        record.setMatchDifficultyGroup(stringValue(gameState.get("matchDifficultyGroup")));
        record.setMatchDifficultyLevel(stringValue(gameState.get("matchDifficultyLevel")));
        record.setMatchDifficultyLabel(stringValue(gameState.get("matchDifficultyLabel")));
    }

    private int safeInt(Integer value, int fallback) {
        return value != null ? value : fallback;
    }

    private String stringValue(Object value) {
        return value != null ? value.toString() : null;
    }

    private Integer intValue(Object value) {
        return value instanceof Number number ? number.intValue() : null;
    }

    private Double doubleValue(Object value) {
        return value instanceof Number number ? number.doubleValue() : null;
    }

    private double numberValue(Object value, double fallback) {
        return value instanceof Number number ? number.doubleValue() : fallback;
    }

    private double resolveExpectedSetMaxScore(Map<String, Object> gameState, User player1, User player2) {
        Object probabilities = gameState.get("difficultyProbabilities");
        if (probabilities instanceof List<?> probabilityList && probabilityList.size() == 10) {
            com.robword.util.PKAlgorithmUtil.DifficultyConfig config = new com.robword.util.PKAlgorithmUtil.DifficultyConfig();
            for (int i = 0; i < probabilityList.size(); i++) {
                Object value = probabilityList.get(i);
                if (value instanceof Number number) {
                    config.probabilities[i] = number.doubleValue();
                }
            }
            return com.robword.util.PKAlgorithmUtil.calculateExpectedSetMaxScore(config, PK_WORD_COUNT);
        }

        com.robword.util.PKAlgorithmUtil.DifficultyConfig fallbackConfig =
                com.robword.util.PKAlgorithmUtil.generatePKDifficulty(player1.getRank(), player2.getRank());
        return com.robword.util.PKAlgorithmUtil.calculateExpectedSetMaxScore(fallbackConfig, PK_WORD_COUNT);
    }

    private int getCurrentWinStreak(User user) {
        if (user == null || user.getCurrentWinStreak() == null) {
            return 0;
        }
        return Math.max(user.getCurrentWinStreak(), 0);
    }

    private void updateCurrentWinStreak(User user, int streak) {
        if (user == null) {
            return;
        }
        user.setCurrentWinStreak(Math.max(streak, 0));
    }

    /**
     * 计算比赛持续时间
     */
    private Integer calculateDuration(Map<String, Object> gameState) {
        Number startTimeNum = (Number) gameState.get("startTime");
        if (startTimeNum == null) {
            return 0;
        }
        // startTime 存储的是 Unix 时间戳（秒）
        long startEpochSecond = startTimeNum.longValue();
        long nowEpochSecond = java.time.Instant.now().getEpochSecond();
        return (int) (nowEpochSecond - startEpochSecond);
    }

    /**
     * 清理Redis数据
     */
    private void cleanupRedisData(Long roomId, Long player1Id, Long player2Id) {
        redisTemplate.delete(GAME_STATE_KEY + roomId);

        // 只删除指向当前房间的USER_ROOM_KEY，避免删除新游戏的映射
        Number currentRoom1 = (Number) redisTemplate.opsForValue().get(USER_ROOM_KEY + player1Id);
        if (currentRoom1 != null && currentRoom1.longValue() == roomId) {
            redisTemplate.delete(USER_ROOM_KEY + player1Id);
        }

        Number currentRoom2 = (Number) redisTemplate.opsForValue().get(USER_ROOM_KEY + player2Id);
        if (currentRoom2 != null && currentRoom2.longValue() == roomId) {
            redisTemplate.delete(USER_ROOM_KEY + player2Id);
        }

        log.info("Redis data cleaned for room {}", roomId);
    }

    /**
     * 查找房间已有的游戏记录（用于判断是否在答题阶段开始时已创建）
     */
    private GameRecord findExistingRecord(Long roomId) {
        // 查询该房间最新的游戏记录（按创建时间倒序）
        return gameRecordMapper.selectOne(
            new com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper<GameRecord>()
                .eq(GameRecord::getRoomId, roomId)
                .orderByDesc(GameRecord::getCreateTime)
                .last("LIMIT 1")
        );
    }

    /**
     * 查找房间已有的游戏记录ID（供WebSocketHandler使用）
     */
    public Long findExistingRecordId(Long roomId) {
        GameRecord record = findExistingRecord(roomId);
        return record != null ? record.getId() : null;
    }

    /**
     * 保存缺失的答题详情（用于结算时补全未实时提交的数据）
     */
    @SuppressWarnings("unchecked")
    private void saveRemainingAnswerDetails(Long recordId, Long userId, List<Word> words,
                                             Map<String, Object> answerData) {
        if (answerData == null) {
            return;
        }

        // 查询该用户已保存的答题数量
        int savedCount = countUserAnswers(recordId, userId);

        List<Map<String, Object>> answers = (List<Map<String, Object>>) answerData.get("answers");
        if (answers == null) {
            return;
        }

        User user = userMapper.selectById(userId);

        // 只保存未插入的答题详情
        int roundNo = savedCount + 1;
        for (int i = savedCount; i < answers.size(); i++) {
            Map<String, Object> answer = answers.get(i);

            GameAnswerDetail detail = new GameAnswerDetail();
            detail.setRecordId(recordId);
            detail.setUserId(userId);
            detail.setUserName(user != null ? user.getNickname() : null);
            detail.setRoundNo(roundNo++);

            // 获取单词信息
            Number wordIdNum = (Number) answer.get("wordId");
            Long wordId = wordIdNum != null ? wordIdNum.longValue() : null;
            detail.setWordId(wordId);

            // 单词内容（前端传入）
            detail.setWordContent((String) answer.get("wordContent"));

            // 单词难度（前端传入）
            Number difficultyNum = (Number) answer.get("wordDifficulty");
            detail.setWordDifficulty(difficultyNum != null ? difficultyNum.intValue() : 0);

            // 四个选项（前端传入数组）
            List<String> options = (List<String>) answer.get("options");
            if (options != null && options.size() >= 4) {
                detail.setOption1(options.get(0));
                detail.setOption2(options.get(1));
                detail.setOption3(options.get(2));
                detail.setOption4(options.get(3));
            }

            // 正确答案索引和选择索引（前端传入）
            Number correctIndexNum = (Number) answer.get("correctAnswerIndex");
            detail.setCorrectAnswerIndex(correctIndexNum != null ? correctIndexNum.intValue() : null);

            Number selectedIndexNum = (Number) answer.get("selectedAnswerIndex");
            detail.setSelectedAnswerIndex(selectedIndexNum != null ? selectedIndexNum.intValue() : null);

            Boolean isCorrect = (Boolean) answer.get("isCorrect");
            detail.setIsCorrect(isCorrect != null && isCorrect ? 1 : 0);

            Number scoreNum = (Number) answer.get("score");
            detail.setScore(scoreNum != null ? scoreNum.intValue() : 0);

            Number answerTimeMsNum = (Number) answer.get("answerTimeMs");
            detail.setAnswerTimeMs(answerTimeMsNum != null ? answerTimeMsNum.intValue() : 0);

            gameAnswerDetailMapper.insert(detail);
            recordMasteryProgress(recordId, detail);
            notifyWrongAnswer(detail);
            log.debug("Remaining answer detail saved: recordId={}, userId={}, roundNo={}",
                    recordId, userId, detail.getRoundNo());
        }
    }

    /**
     * 删除预创建的游戏记录（双方都退出未答题时调用）
     */
    public void deleteRecord(Long recordId) {
        try {
            gameRecordMapper.deleteById(recordId);
            // 同时删除可能已插入的答题详情
            gameAnswerDetailMapper.delete(
                new com.baomidou.mybatisplus.core.conditions.query.QueryWrapper<com.robword.entity.GameAnswerDetail>()
                    .eq("record_id", recordId)
            );
            log.info("Deleted abandoned game record: id={}", recordId);
        } catch (Exception e) {
            log.error("Failed to delete game record {}: {}", recordId, e.getMessage());
        }
    }

    private void notifyWrongAnswer(GameAnswerDetail detail) {
        if (wrongWordReviewProgressService != null
                && detail != null
                && Integer.valueOf(0).equals(detail.getIsCorrect())) {
            wrongWordReviewProgressService.recordWrong(
                    detail.getUserId(),
                    detail.getWordId(),
                    detail.getWordContent(),
                    detail.getCreateTime(),
                    detail.getId()
            );
        }
        if (wrongWordAgentNotificationService != null) {
            wrongWordAgentNotificationService.notifyWrongAnswer(detail);
        }
    }

    private void recordMasteryProgress(Long recordId, GameAnswerDetail detail) {
        try {
            GameRecord record = recordId != null ? gameRecordMapper.selectById(recordId) : null;
            userWordMasteryService.recordAnswer(detail, record != null ? record.getMode() : null);
        } catch (Exception e) {
            log.warn("Failed to update word mastery progress: detailId={}, wordId={}, error={}",
                    detail != null ? detail.getId() : null,
                    detail != null ? detail.getWordId() : null,
                    e.getMessage());
        }
    }

    /**
     * 玩家结果内部类
     */
    static class PlayerResult {
        Long userId;
        int score;
        int correctCount;
        int totalCount;
        int setMaxScore;
    }

    static class SettlementOutcome {
        final int masteryExp;
        final int battleExp;
        final int streakExp;
        final int expChange;
        final double completionRate;
        final double challengeIndex;
        final int effectiveStreak;
        final boolean effectiveWin;

        SettlementOutcome(int masteryExp, int battleExp, int streakExp, int expChange,
                          double completionRate, double challengeIndex, int effectiveStreak,
                          boolean effectiveWin) {
            this.masteryExp = masteryExp;
            this.battleExp = battleExp;
            this.streakExp = streakExp;
            this.expChange = expChange;
            this.completionRate = completionRate;
            this.challengeIndex = challengeIndex;
            this.effectiveStreak = effectiveStreak;
            this.effectiveWin = effectiveWin;
        }

        static SettlementOutcome empty() {
            return new SettlementOutcome(0, 0, 0, 0, 0.0, 0.0, 0, false);
        }

        int masteryExp() {
            return masteryExp;
        }

        int battleExp() {
            return battleExp;
        }

        int streakExp() {
            return streakExp;
        }

        int expChange() {
            return expChange;
        }

        double completionRate() {
            return completionRate;
        }

        int effectiveStreak() {
            return effectiveStreak;
        }

        boolean effectiveWin() {
            return effectiveWin;
        }
    }
}
