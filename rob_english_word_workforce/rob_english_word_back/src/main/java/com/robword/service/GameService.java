package com.robword.service;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.robword.entity.Room;
import com.robword.entity.User;
import com.robword.entity.Word;
import com.robword.mapper.RoomMapper;
import com.robword.mapper.UserMapper;
import com.robword.netty.ChannelManager;
import com.robword.state.PlayerState;
import com.robword.state.PlayerStateManager;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Lazy;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;

import java.time.Instant;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ThreadLocalRandom;
import java.util.concurrent.TimeUnit;
import java.util.stream.Collectors;

/**
 * 游戏服务
 * 集成状态机 + Netty ChannelManager + AnswerService
 */
@Service
@Slf4j
public class GameService {

    public record GameStartResult(boolean success, String message) {
        public static GameStartResult ok() {
            return new GameStartResult(true, "");
        }

        public static GameStartResult failed(String message) {
            return new GameStartResult(false, message);
        }
    }

    @Value("${game.words-per-player:5}")
    private int wordsPerPlayer;

    private final RedisTemplate<String, Object> redisTemplate;
    private final RoomMapper roomMapper;
    private final UserMapper userMapper;
    private final WordService wordService;
    private final ChannelManager channelManager;
    private final PlayerStateManager stateManager;
    private final AnswerService answerService;
    private final GameSettlementService settlementService;
    private final UserWordMasteryService userWordMasteryService;
    private final ObjectMapper objectMapper;
    private final TrainingDifficultyCatalog difficultyCatalog;

    private static final String GAME_STATE_KEY = "game:state:";
    private static final String USER_ROOM_KEY = "game:user_room:";
    private static final String GRAB_LOCK_KEY = "game:lock:grab:";
    private static final String MATCH_MODE = "match";
    private static final String SOLO_TRAINING_MODE = "solo_training";
    private static final Long SOLO_ROBOT_ID = -1L;
    private static final String RANK_DIFFICULTY_LEVEL = "rank_current";

    public GameService(RedisTemplate<String, Object> redisTemplate,
                       RoomMapper roomMapper,
                       UserMapper userMapper,
                       WordService wordService,
                       ChannelManager channelManager,
                       PlayerStateManager stateManager,
                       @Lazy AnswerService answerService,
                       @Lazy GameSettlementService settlementService,
                       UserWordMasteryService userWordMasteryService,
                       ObjectMapper objectMapper,
                       TrainingDifficultyCatalog difficultyCatalog) {
        this.redisTemplate = redisTemplate;
        this.roomMapper = roomMapper;
        this.userMapper = userMapper;
        this.wordService = wordService;
        this.channelManager = channelManager;
        this.stateManager = stateManager;
        this.answerService = answerService;
        this.settlementService = settlementService;
        this.userWordMasteryService = userWordMasteryService;
        this.objectMapper = objectMapper;
        this.difficultyCatalog = difficultyCatalog;
    }

    /**
     * 开始游戏
     * 由 MatchService 匹配成功后调用
     * 此时双方状态已是 MATCHED
     */
    public GameStartResult startGame(Long player1Id,
                                     Long player2Id,
                                     Room room,
                                     TrainingDifficultyCatalog.Difficulty matchDifficulty) {
        log.info("Starting game: roomId={}, p1={}, p2={}", room.getId(), player1Id, player2Id);

        // CAS: MATCHED → GRABBING
        boolean cas1 = stateManager.transition(player1Id, PlayerState.MATCHED, PlayerState.GRABBING);
        boolean cas2 = stateManager.transition(player2Id, PlayerState.MATCHED, PlayerState.GRABBING);

        if (!cas1 || !cas2) {
            log.error("Failed to transition to GRABBING: p1={} cas1={}, p2={} cas2={}",
                    player1Id, cas1, player2Id, cas2);
            rollbackGameStart(room.getId(), player1Id, player2Id, "failed to transition to grabbing");
            return GameStartResult.failed("匹配状态已失效，请重新匹配");
        }

        if (!channelManager.isOnline(player1Id) || !channelManager.isOnline(player2Id)) {
            rollbackGameStart(room.getId(), player1Id, player2Id, "player channel unavailable before game start push");
            return GameStartResult.failed("对手已离线，请重新匹配");
        }

        User player1 = userMapper.selectById(player1Id);
        User player2 = userMapper.selectById(player2Id);
        if (player1 == null || player2 == null) {
            rollbackGameStart(room.getId(), player1Id, player2Id, "player not found");
            return GameStartResult.failed("用户不存在，请重新登录");
        }

        int totalWords = wordsPerPlayer * 2;
        com.robword.util.PKAlgorithmUtil.DifficultyConfig diffConfig = null;
        List<Word> words;
        if (matchDifficulty.rankBased()) {
            diffConfig = com.robword.util.PKAlgorithmUtil.generatePKDifficulty(player1.getRank(), player2.getRank());
            words = drawWords(totalWords, diffConfig);
        } else {
            words = new ArrayList<>(wordService.getRandomWordsForTrainingLibraries(
                    matchDifficulty.libraryNames(), totalWords));
            if (words.size() != totalWords) {
                String message = "所选难度可用单词不足 " + totalWords + " 个，请选择其他难度";
                rollbackGameStart(room.getId(), player1Id, player2Id,
                        "selected difficulty returned " + words.size() + " words");
                return GameStartResult.failed(message);
            }
            Collections.shuffle(words);
        }

        String wordDetails = words.stream()
                .map(w -> w.getWord() + "(" + w.getDifficulty() + ")")
                .collect(Collectors.joining(", "));
        log.info("Words: [{}]", wordDetails);

        // 将 Word 实体转换为普通 Map，避免 Redis 序列化/反序列化类型不一致
        List<Map<String, Object>> wordMaps = words.stream()
                .map(this::wordToMap)
                .collect(Collectors.toList());

        Map<String, Object> gameState = new ConcurrentHashMap<>();
        gameState.put("roomId", room.getId());
        gameState.put("mode", MATCH_MODE);
        gameState.put("player1Id", player1Id);
        gameState.put("player2Id", player2Id);
        gameState.put("player1Words", new ArrayList<Map<String, Object>>());
        gameState.put("player2Words", new ArrayList<Map<String, Object>>());
        gameState.put("allWords", wordMaps);
        gameState.put("difficultyProbabilities", diffConfig == null
                ? List.of()
                : Arrays.stream(diffConfig.probabilities).boxed().toList());
        gameState.put("matchDifficultyGroup", matchDifficulty.group());
        gameState.put("matchDifficultyLevel", matchDifficulty.level());
        gameState.put("matchDifficultyLabel", matchDifficulty.label());
        gameState.put("grabbedWords", new ArrayList<Long>());
        gameState.put("phase", "grab");
        gameState.put("grabTimeLeft", 6);
        gameState.put("startTime", Instant.now().getEpochSecond());

        String gameKey = GAME_STATE_KEY + room.getId();
        redisTemplate.opsForValue().set(gameKey, gameState, 5, TimeUnit.MINUTES);

        // 存储 userId → roomId 映射
        redisTemplate.opsForValue().set(USER_ROOM_KEY + player1Id, room.getId(), 5, TimeUnit.MINUTES);
        redisTemplate.opsForValue().set(USER_ROOM_KEY + player2Id, room.getId(), 5, TimeUnit.MINUTES);

        // 推送 game_start 给双方（通过 ChannelManager）
        boolean player1StatePushed = channelManager.sendToPlayer(player1Id, "state_change", Map.of("state", "GRABBING"));
        boolean player2StatePushed = channelManager.sendToPlayer(player2Id, "state_change", Map.of("state", "GRABBING"));
        if (!player1StatePushed || !player2StatePushed) {
            rollbackGameStart(room.getId(), player1Id, player2Id, "failed to push grabbing state");
            return GameStartResult.failed("推送游戏状态失败，请重新匹配");
        }

        Map<String, Object> p1GameStart = new HashMap<>();
        p1GameStart.put("roomId", room.getId());
        p1GameStart.put("mode", MATCH_MODE);
        p1GameStart.put("words", wordMaps);
        p1GameStart.put("phase", "grab");
        p1GameStart.put("timeLeft", 6);
        p1GameStart.put("maxWordsPerPlayer", wordsPerPlayer);
        putMatchDifficulty(p1GameStart, matchDifficulty);
        p1GameStart.put("opponent", Map.of(
                "userId", player2Id,
                "nickname", player2.getNickname(),
                "rank", player2.getRank()
        ));
        boolean player1Started = channelManager.sendToPlayer(player1Id, "game_start", p1GameStart);

        Map<String, Object> p2GameStart = new HashMap<>();
        p2GameStart.put("roomId", room.getId());
        p2GameStart.put("mode", MATCH_MODE);
        p2GameStart.put("words", wordMaps);
        p2GameStart.put("phase", "grab");
        p2GameStart.put("timeLeft", 6);
        p2GameStart.put("maxWordsPerPlayer", wordsPerPlayer);
        putMatchDifficulty(p2GameStart, matchDifficulty);
        p2GameStart.put("opponent", Map.of(
                "userId", player1Id,
                "nickname", player1.getNickname(),
                "rank", player1.getRank()
        ));
        boolean player2Started = channelManager.sendToPlayer(player2Id, "game_start", p2GameStart);
        if (!player1Started || !player2Started) {
            rollbackGameStart(room.getId(), player1Id, player2Id, "failed to push game_start");
            return GameStartResult.failed("推送游戏开始消息失败，请重新匹配");
        }

        log.info("Game started for room {}", room.getId());
        return GameStartResult.ok();
    }

    private void putMatchDifficulty(Map<String, Object> payload,
                                    TrainingDifficultyCatalog.Difficulty difficulty) {
        payload.put("matchDifficultyGroup", difficulty.group());
        payload.put("matchDifficultyLevel", difficulty.level());
        payload.put("matchDifficultyLabel", difficulty.label());
    }

    /**
     * 开始单人训练：跳过抢词倒计时，直接进入答题阶段。
     */
    public void startSoloTraining(Long userId) {
        startSoloTraining(userId, null, null);
    }

    public void startSoloTraining(Long userId, String difficultyGroup, String difficultyLevel) {
        log.info("Starting solo training for user {}, difficultyGroup={}, difficultyLevel={}",
                userId, difficultyGroup, difficultyLevel);

        if (!channelManager.isOnline(userId)) {
            stateManager.forceState(userId, PlayerState.IDLE);
            return;
        }

        User player = userMapper.selectById(userId);
        if (player == null) {
            channelManager.sendToPlayer(userId, "error", Map.of("message", "用户不存在"));
            stateManager.forceState(userId, PlayerState.IDLE);
            return;
        }

        stateManager.forceState(userId, PlayerState.ANSWERING);

        Room room = createSoloTrainingRoom(userId);
        int totalWords = wordsPerPlayer * 2;
        int trainingRank = player.getTrainingRank() != null ? player.getTrainingRank() : 1;
        Map<String, Object> robotProfile = generateRobotProfile(trainingRank);

        com.robword.util.PKAlgorithmUtil.DifficultyConfig diffConfig =
                com.robword.util.PKAlgorithmUtil.generatePKDifficulty(trainingRank, Math.min(trainingRank + 8, 999));
        List<Word> words = drawSoloTrainingWords(userId, totalWords, diffConfig, difficultyGroup, difficultyLevel);
        List<Map<String, Object>> wordMaps = words.stream()
                .map(this::wordToMap)
                .collect(Collectors.toList());
        SoloWordSets wordSets = splitSoloTrainingWords(wordMaps);

        Map<String, Object> gameState = new ConcurrentHashMap<>();
        gameState.put("roomId", room.getId());
        gameState.put("mode", SOLO_TRAINING_MODE);
        gameState.put("player1Id", userId);
        gameState.put("player2Id", SOLO_ROBOT_ID);
        gameState.put("player1Words", wordSets.playerWords());
        gameState.put("player2Words", wordSets.robotWords());
        gameState.put("allWords", wordMaps);
        gameState.put("difficultyProbabilities", Arrays.stream(diffConfig.probabilities).boxed().toList());
        gameState.put("trainingDifficultyGroup", safeDifficultyValue(difficultyGroup, "rank"));
        gameState.put("trainingDifficultyLevel", safeDifficultyValue(difficultyLevel, RANK_DIFFICULTY_LEVEL));
        gameState.put("robotProfile", robotProfile);
        gameState.put("grabbedWords", wordMaps.stream()
                .map(w -> ((Number) w.get("id")).longValue())
                .collect(Collectors.toList()));
        gameState.put("phase", "answer");
        gameState.put("grabTimeLeft", 0);
        gameState.put("startTime", Instant.now().getEpochSecond());

        String gameKey = GAME_STATE_KEY + room.getId();
        redisTemplate.opsForValue().set(gameKey, gameState, 5, TimeUnit.MINUTES);
        redisTemplate.opsForValue().set(USER_ROOM_KEY + userId, room.getId(), 5, TimeUnit.MINUTES);

        Long recordId = settlementService.createInitialSoloTrainingRecord(room.getId());
        if (recordId == null) {
            rollbackSoloTrainingStart(room.getId(), userId, "failed to create solo training record");
            channelManager.sendToPlayer(userId, "error", Map.of("message", "训练记录创建失败，请稍后重试"));
            return;
        }

        boolean statePushed = channelManager.sendToPlayer(userId, "state_change", Map.of("state", "ANSWERING"));
        Map<String, Object> payload = new HashMap<>();
        payload.put("roomId", room.getId());
        payload.put("mode", SOLO_TRAINING_MODE);
        payload.put("phase", "answer");
        payload.put("words", wordMaps);
        payload.put("myWords", wordSets.playerWords());
        payload.put("timeLeft", 0);
        payload.put("maxWordsPerPlayer", wordsPerPlayer);
        payload.put("trainingDifficultyGroup", safeDifficultyValue(difficultyGroup, "rank"));
        payload.put("trainingDifficultyLevel", safeDifficultyValue(difficultyLevel, RANK_DIFFICULTY_LEVEL));
        payload.put("robotProfile", robotProfile);
        payload.put("opponent", Map.of(
                "userId", SOLO_ROBOT_ID,
                "nickname", robotProfile.getOrDefault("name", "训练机器人"),
                "rank", robotProfile.getOrDefault("tier", "normal")
        ));
        boolean started = channelManager.sendToPlayer(userId, "game_start", payload);

        if (!statePushed || !started) {
            settlementService.deleteRecord(recordId);
            rollbackSoloTrainingStart(room.getId(), userId, "failed to push solo training start");
            return;
        }

        answerService.initAnswerPhase(room.getId());
    }

    private void rollbackGameStart(Long roomId, Long player1Id, Long player2Id, String reason) {
        log.warn("Rolling back game start for room {}: {}", roomId, reason);
        stateManager.forceState(player1Id, PlayerState.IDLE);
        stateManager.forceState(player2Id, PlayerState.IDLE);
        channelManager.sendToPlayer(player1Id, "state_change", Map.of("state", "IDLE"));
        channelManager.sendToPlayer(player2Id, "state_change", Map.of("state", "IDLE"));
        redisTemplate.delete(GAME_STATE_KEY + roomId);
        redisTemplate.delete(USER_ROOM_KEY + player1Id);
        redisTemplate.delete(USER_ROOM_KEY + player2Id);
        roomMapper.deleteById(roomId);
    }

    private void rollbackSoloTrainingStart(Long roomId, Long userId, String reason) {
        log.warn("Rolling back solo training start for room {}: {}", roomId, reason);
        stateManager.forceState(userId, PlayerState.IDLE);
        channelManager.sendToPlayer(userId, "state_change", Map.of("state", "IDLE"));
        redisTemplate.delete(GAME_STATE_KEY + roomId);
        redisTemplate.delete(USER_ROOM_KEY + userId);
        roomMapper.deleteById(roomId);
    }

    private Room createSoloTrainingRoom(Long playerId) {
        Room room = new Room();
        room.setRoomCode(("TR" + UUID.randomUUID().toString().replace("-", "")).substring(0, 8).toUpperCase());
        room.setStatus(1);
        room.setPlayer1Id(playerId);
        room.setPlayer2Id(SOLO_ROBOT_ID);
        roomMapper.insert(room);
        return room;
    }

    private List<Word> drawWords(int totalWords, com.robword.util.PKAlgorithmUtil.DifficultyConfig diffConfig) {
        com.robword.util.PKAlgorithmUtil.WordDifficultyTier[] tiers =
                com.robword.util.PKAlgorithmUtil.WordDifficultyTier.values();
        int[] tierCounts = new int[tiers.length];
        int allocated = 0;
        for (int i = 0; i < tiers.length - 1; i++) {
            tierCounts[i] = (int) Math.round(totalWords * diffConfig.get(i));
            allocated += tierCounts[i];
        }
        tierCounts[tiers.length - 1] = Math.max(0, totalWords - allocated);

        List<Word> words = new ArrayList<>();
        for (int i = 0; i < tiers.length; i++) {
            if (tierCounts[i] > 0) {
                words.addAll(wordService.getRandomWordsForMatch(tiers[i].min, tiers[i].max, tierCounts[i]));
            }
        }
        if (words.size() < totalWords) {
            words.addAll(wordService.getRandomWordsForMatch(1, 1000, totalWords - words.size()));
        }
        if (words.size() > totalWords) {
            words = words.subList(0, totalWords);
        }
        Collections.shuffle(words);
        return words;
    }

    private List<Word> drawSoloTrainingWords(Long userId,
                                             int totalWords,
                                             com.robword.util.PKAlgorithmUtil.DifficultyConfig diffConfig,
                                             String difficultyGroup,
                                             String difficultyLevel) {
        String resolvedGroup = safeDifficultyValue(difficultyGroup, "rank");
        String resolvedLevel = safeDifficultyValue(difficultyLevel, RANK_DIFFICULTY_LEVEL);
        Optional<TrainingDifficultyCatalog.Difficulty> selection = difficultyCatalog.resolve(resolvedGroup, resolvedLevel);
        if (selection.isEmpty()) {
            log.warn("Unknown solo training difficulty group/level {}/{}, falling back to rank difficulty",
                    resolvedGroup, resolvedLevel);
            return drawSoloTrainingRankWords(userId, totalWords, diffConfig, new ArrayList<>());
        }

        TrainingDifficultyCatalog.Difficulty difficulty = selection.get();
        if (difficulty.rankBased()) {
            return drawSoloTrainingRankWords(userId, totalWords, diffConfig, new ArrayList<>());
        }
        List<String> libraryNames = difficulty.libraryNames();

        List<Word> words = new ArrayList<>(userWordMasteryService.dueReviewWords(userId, libraryNames, totalWords));
        List<Long> excludeWordIds = wordIds(words);
        int remaining = totalWords - words.size();
        if (remaining > 0) {
            words.addAll(wordService.getRandomWordsForTrainingLibraries(userId, libraryNames, remaining, excludeWordIds));
        }
        if (words.size() < totalWords) {
            log.warn("Solo training difficulty level {} only returned {} words, filling {} words with rank difficulty",
                    difficultyLevel, words.size(), totalWords - words.size());
            words.addAll(drawSoloTrainingRankWords(userId, totalWords - words.size(), diffConfig, wordIds(words)));
        }
        if (words.size() > totalWords) {
            words = words.subList(0, totalWords);
        }
        Collections.shuffle(words);
        return words;
    }

    private List<Word> drawSoloTrainingRankWords(Long userId,
                                                 int totalWords,
                                                 com.robword.util.PKAlgorithmUtil.DifficultyConfig diffConfig,
                                                 List<Long> excludeWordIds) {
        if (totalWords <= 0) {
            return List.of();
        }

        List<Long> selectedIds = new ArrayList<>();
        if (excludeWordIds != null) {
            selectedIds.addAll(excludeWordIds);
        }
        List<Word> words = userWordMasteryService.dueReviewWords(userId, null, totalWords)
                .stream()
                .filter(word -> word.getId() != null && !selectedIds.contains(word.getId()))
                .collect(Collectors.toCollection(ArrayList::new));
        selectedIds.addAll(wordIds(words));

        int remaining = totalWords - words.size();
        if (remaining <= 0) {
            Collections.shuffle(words);
            return words.size() > totalWords ? words.subList(0, totalWords) : words;
        }

        com.robword.util.PKAlgorithmUtil.WordDifficultyTier[] tiers =
                com.robword.util.PKAlgorithmUtil.WordDifficultyTier.values();
        int[] tierCounts = new int[tiers.length];
        int allocated = 0;
        for (int i = 0; i < tiers.length - 1; i++) {
            tierCounts[i] = (int) Math.round(remaining * diffConfig.get(i));
            allocated += tierCounts[i];
        }
        tierCounts[tiers.length - 1] = Math.max(0, remaining - allocated);

        for (int i = 0; i < tiers.length; i++) {
            if (tierCounts[i] <= 0) {
                continue;
            }
            List<Word> tierWords = wordService.getRandomWordsForSoloTraining(
                    userId,
                    tiers[i].min,
                    tiers[i].max,
                    tierCounts[i],
                    selectedIds
            );
            words.addAll(tierWords);
            selectedIds.addAll(wordIds(tierWords));
        }

        if (words.size() < totalWords) {
            List<Word> filler = wordService.getRandomWordsForSoloTraining(
                    userId,
                    1,
                    1000,
                    totalWords - words.size(),
                    selectedIds
            );
            words.addAll(filler);
        }
        if (words.size() > totalWords) {
            words = words.subList(0, totalWords);
        }
        Collections.shuffle(words);
        return words;
    }

    private List<Long> wordIds(List<Word> words) {
        if (words == null || words.isEmpty()) {
            return new ArrayList<>();
        }
        return words.stream()
                .map(Word::getId)
                .filter(Objects::nonNull)
                .distinct()
                .collect(Collectors.toCollection(ArrayList::new));
    }

    private String safeDifficultyValue(String value, String fallback) {
        return value == null || value.isBlank() ? fallback : value;
    }

    private SoloWordSets splitSoloTrainingWords(List<Map<String, Object>> wordMaps) {
        List<Map<String, Object>> sorted = new ArrayList<>(wordMaps);
        sorted.sort(Comparator.comparingInt(this::wordDifficulty).reversed());

        List<Map<String, Object>> playerWords = new ArrayList<>();
        List<Map<String, Object>> robotWords = new ArrayList<>();
        ThreadLocalRandom random = ThreadLocalRandom.current();

        for (int i = 0; i < sorted.size(); i++) {
            Map<String, Object> word = sorted.get(i);
            double hardRank = sorted.size() <= 1 ? 0.5 : 1.0 - (double) i / (sorted.size() - 1);
            double playerChance = Math.max(0.15, Math.min(0.90,
                    0.20 + hardRank * 0.65 + random.nextDouble(-0.10, 0.10)));
            boolean toPlayer = random.nextDouble() < playerChance;

            if ((toPlayer && playerWords.size() < wordsPerPlayer) || robotWords.size() >= wordsPerPlayer) {
                playerWords.add(word);
            } else {
                robotWords.add(word);
            }
        }

        rebalanceSoloTrainingWords(playerWords, robotWords);
        return new SoloWordSets(playerWords, robotWords);
    }

    private void rebalanceSoloTrainingWords(List<Map<String, Object>> playerWords,
                                            List<Map<String, Object>> robotWords) {
        int guard = 0;
        while (totalDifficulty(playerWords) <= totalDifficulty(robotWords) && guard < wordsPerPlayer) {
            Optional<Map<String, Object>> easiestPlayerWord = playerWords.stream()
                    .min(Comparator.comparingInt(this::wordDifficulty));
            Optional<Map<String, Object>> hardestRobotWord = robotWords.stream()
                    .max(Comparator.comparingInt(this::wordDifficulty));
            if (easiestPlayerWord.isEmpty() || hardestRobotWord.isEmpty()
                    || wordDifficulty(hardestRobotWord.get()) <= wordDifficulty(easiestPlayerWord.get())) {
                return;
            }
            playerWords.remove(easiestPlayerWord.get());
            robotWords.remove(hardestRobotWord.get());
            playerWords.add(hardestRobotWord.get());
            robotWords.add(easiestPlayerWord.get());
            guard++;
        }
    }

    private int totalDifficulty(List<Map<String, Object>> words) {
        return words.stream().mapToInt(this::wordDifficulty).sum();
    }

    private int wordDifficulty(Map<String, Object> word) {
        Object difficulty = word.get("difficulty");
        return difficulty instanceof Number number ? number.intValue() : 1;
    }

    private Map<String, Object> generateRobotProfile(int trainingRank) {
        ThreadLocalRandom random = ThreadLocalRandom.current();
        String tier = chooseRobotTier(trainingRank, random.nextDouble());

        double minAccuracy;
        double maxAccuracy;
        double minResistance;
        double maxResistance;
        double minVolatility;
        double maxVolatility;
        double minCareless;
        double maxCareless;
        double minBurst;
        double maxBurst;
        double challengeMultiplier;
        String name;

        switch (tier) {
            case "strong" -> {
                minAccuracy = 0.70; maxAccuracy = 0.90;
                minResistance = 0.52; maxResistance = 0.86;
                minVolatility = 0.05; maxVolatility = 0.16;
                minCareless = 0.02; maxCareless = 0.11;
                minBurst = 0.07; maxBurst = 0.18;
                challengeMultiplier = 1.35;
                name = "训练机器人·天才型";
            }
            case "normal" -> {
                minAccuracy = 0.56; maxAccuracy = 0.78;
                minResistance = 0.30; maxResistance = 0.62;
                minVolatility = 0.08; maxVolatility = 0.19;
                minCareless = 0.05; maxCareless = 0.17;
                minBurst = 0.04; maxBurst = 0.13;
                challengeMultiplier = 1.00;
                name = "训练机器人·稳健型";
            }
            default -> {
                minAccuracy = 0.42; maxAccuracy = 0.64;
                minResistance = 0.12; maxResistance = 0.38;
                minVolatility = 0.10; maxVolatility = 0.24;
                minCareless = 0.10; maxCareless = 0.24;
                minBurst = 0.02; maxBurst = 0.09;
                challengeMultiplier = 0.75;
                name = "训练机器人·摸鱼型";
            }
        }

        int aptitude = random.nextInt(800, 1601);
        double growth = round2(random.nextDouble(0.85, 1.26));
        double quality = Math.pow(random.nextDouble(), 0.72);
        double baseAccuracy = minAccuracy + (maxAccuracy - minAccuracy) * quality
                + ((aptitude - 1200) / 400.0) * 0.025
                + (growth - 1.0) * 0.03;

        Map<String, Object> profile = new HashMap<>();
        profile.put("tier", tier);
        profile.put("name", name);
        profile.put("aptitude", aptitude);
        profile.put("growth", growth);
        profile.put("baseAccuracy", round2(Math.max(0.32, Math.min(0.94, baseAccuracy))));
        profile.put("difficultyResistance", round2(random.nextDouble(minResistance, maxResistance)));
        profile.put("volatility", round2(random.nextDouble(minVolatility, maxVolatility)));
        profile.put("carelessRate", round2(random.nextDouble(minCareless, maxCareless)));
        profile.put("burstRate", round2(random.nextDouble(minBurst, maxBurst)));
        profile.put("challengeMultiplier", challengeMultiplier);
        return profile;
    }

    private String chooseRobotTier(int trainingRank, double roll) {
        double weakWeight;
        double normalWeight;
        double strongWeight;
        if (trainingRank <= 10) {
            weakWeight = 0.55; normalWeight = 0.40; strongWeight = 0.05;
        } else if (trainingRank <= 30) {
            weakWeight = 0.28; normalWeight = 0.54; strongWeight = 0.18;
        } else if (trainingRank <= 50) {
            weakWeight = 0.14; normalWeight = 0.50; strongWeight = 0.36;
        } else {
            weakWeight = 0.08; normalWeight = 0.42; strongWeight = 0.50;
        }

        if (roll < weakWeight) return "weak";
        if (roll < weakWeight + normalWeight) return "normal";
        return "strong";
    }

    private double round2(double value) {
        return Math.round(value * 100.0) / 100.0;
    }

    /**
     * 抢词逻辑
     * 注意：Redis 反序列化后，Word 对象变成 LinkedHashMap
     */
    @SuppressWarnings("unchecked")
    public void grabWord(Long userId, Long wordId) {
        Number roomIdNum = (Number) redisTemplate.opsForValue().get(USER_ROOM_KEY + userId);
        if (roomIdNum == null) {
            channelManager.sendToPlayer(userId, "grab_error", Map.of("error", "Game not found"));
            return;
        }
        Long roomId = roomIdNum.longValue();

        String gameKey = GAME_STATE_KEY + roomId;
        String lockKey = GRAB_LOCK_KEY + roomId + ":" + wordId;

        Boolean locked = redisTemplate.opsForValue().setIfAbsent(lockKey, "1", 3, TimeUnit.SECONDS);
        if (!Boolean.TRUE.equals(locked)) {
            channelManager.sendToPlayer(userId, "grab_error", Map.of("error", "Word is being grabbed"));
            return;
        }

        try {
            Map<String, Object> gameState = (Map<String, Object>) redisTemplate.opsForValue().get(gameKey);
            if (gameState == null) {
                channelManager.sendToPlayer(userId, "grab_error", Map.of("error", "Game not found"));
                return;
            }

            List<Number> grabbedWords = (List<Number>) gameState.get("grabbedWords");
            if (grabbedWords.stream().anyMatch(id -> id.longValue() == wordId)) {
                channelManager.sendToPlayer(userId, "grab_error", Map.of("error", "Word already grabbed"));
                return;
            }

            // 检查该玩家是否已达到最大抢词数量
            Number player1IdNum = (Number) gameState.get("player1Id");
            Long player1Id = player1IdNum != null ? player1IdNum.longValue() : null;

            List<Map<String, Object>> playerWords;
            if (userId.equals(player1Id)) {
                playerWords = (List<Map<String, Object>>) gameState.get("player1Words");
            } else {
                playerWords = (List<Map<String, Object>>) gameState.get("player2Words");
            }

            if (playerWords.size() >= wordsPerPlayer) {
                channelManager.sendToPlayer(userId, "grab_error", Map.of("error", "已达到最大抢词数量"));
                return;
            }

            // 查找单词
            List<Map<String, Object>> allWords = (List<Map<String, Object>>) gameState.get("allWords");
            Map<String, Object> grabbedWord = allWords.stream()
                    .filter(w -> ((Number) w.get("id")).longValue() == wordId)
                    .findFirst().orElse(null);
            if (grabbedWord == null) {
                channelManager.sendToPlayer(userId, "grab_error", Map.of("error", "Word not found"));
                return;
            }

            // 添加到玩家词汇列表和已抢列表
            grabbedWords.add(wordId);
            gameState.put("grabbedWords", grabbedWords);
            playerWords.add(grabbedWord);

            redisTemplate.opsForValue().set(gameKey, gameState, 5, TimeUnit.MINUTES);

            // 广播抢词结果给双方
            Number p2Num = (Number) gameState.get("player2Id");
            Long player2Id = p2Num != null ? p2Num.longValue() : null;

            Map<String, Object> result = new HashMap<>();
            result.put("success", true);
            result.put("wordId", wordId);
            result.put("grabbedBy", userId);
            result.put("word", grabbedWord);

            channelManager.sendToPlayer(player1Id, "grab_result", result);
            channelManager.sendToPlayer(player2Id, "grab_result", result);

        } finally {
            redisTemplate.delete(lockKey);
        }
    }

    /**
     * 抢词阶段倒计时（每秒检查）
     */
    @Scheduled(fixedRate = 1000)
    @SuppressWarnings("unchecked")
    public void checkGrabPhaseTimeout() {
        Set<String> keys = redisTemplate.keys(GAME_STATE_KEY + "*");
        if (keys == null || keys.isEmpty()) return;

        for (String gameKey : keys) {
            try {
                Map<String, Object> gameState = (Map<String, Object>) redisTemplate.opsForValue().get(gameKey);
                if (gameState == null) continue;

                String phase = (String) gameState.get("phase");
                if (!"grab".equals(phase)) continue;

                Number roomIdNum = (Number) gameState.get("roomId");
                Long roomId = roomIdNum != null ? roomIdNum.longValue() : null;
                Integer timeLeft = (Integer) gameState.get("grabTimeLeft");
                Number player1IdNum = (Number) gameState.get("player1Id");
                Number player2IdNum = (Number) gameState.get("player2Id");
                Long player1Id = player1IdNum != null ? player1IdNum.longValue() : null;
                Long player2Id = player2IdNum != null ? player2IdNum.longValue() : null;

                if (timeLeft == null) timeLeft = 6;
                timeLeft--;
                gameState.put("grabTimeLeft", timeLeft);

                if (timeLeft <= 0) {
                    // 检查双方是否仍在游戏中
                    PlayerState p1State = stateManager.getState(player1Id);
                    PlayerState p2State = stateManager.getState(player2Id);

                    // 如果双方都已退出（不在 GRABBING 状态），清理并跳过
                    if (p1State != PlayerState.GRABBING && p2State != PlayerState.GRABBING) {
                        log.info("Both players left during grab phase in room {}, cleaning up", roomId);
                        redisTemplate.delete(gameKey);
                        continue;
                    }

                    // 抢词结束，自动分配剩余单词
                    distributeRemainingWords(gameState, player1Id, player2Id);
                    gameState.put("phase", "answer");

                    // CAS: GRABBING → ANSWERING（只对仍在游戏中的玩家转换）
                    stateManager.transition(player1Id, PlayerState.GRABBING, PlayerState.ANSWERING);
                    stateManager.transition(player2Id, PlayerState.GRABBING, PlayerState.ANSWERING);

                    List<Map<String, Object>> p1Words = (List<Map<String, Object>>) gameState.get("player1Words");
                    List<Map<String, Object>> p2Words = (List<Map<String, Object>>) gameState.get("player2Words");

                    // 推送 grab_phase_end
                    channelManager.sendToPlayer(player1Id, "state_change", Map.of("state", "ANSWERING"));
                    channelManager.sendToPlayer(player2Id, "state_change", Map.of("state", "ANSWERING"));

                    channelManager.sendToPlayer(player1Id, "grab_phase_end", Map.of(
                            "roomId", roomId,
                            "phase", "answer",
                            "myWords", p1Words,
                            "wordCount", p1Words.size()
                    ));
                    channelManager.sendToPlayer(player2Id, "grab_phase_end", Map.of(
                            "roomId", roomId,
                            "phase", "answer",
                            "myWords", p2Words,
                            "wordCount", p2Words.size()
                    ));

                    // 保存更新后的状态
                    redisTemplate.opsForValue().set(gameKey, gameState, 5, TimeUnit.MINUTES);

                    // 创建初始游戏记录
                    try {
                        settlementService.createInitialGameRecord(roomId);
                    } catch (Exception e) {
                        log.error("Failed to create game record for room {}: {}", roomId, e.getMessage());
                    }

                    // 初始化答题阶段（后端出题）
                    answerService.initAnswerPhase(roomId);
                } else {
                    // 广播倒计时
                    channelManager.sendToPlayer(player1Id, "grab_time_update", Map.of("timeLeft", timeLeft));
                    channelManager.sendToPlayer(player2Id, "grab_time_update", Map.of("timeLeft", timeLeft));

                    redisTemplate.opsForValue().set(gameKey, gameState, 5, TimeUnit.MINUTES);
                }
            } catch (Exception e) {
                log.error("Error in checkGrabPhaseTimeout for {}: {}", gameKey, e.getMessage());
            }
        }
    }

    /**
     * 自动分配剩余未被抢的单词
     * 注意：Redis 反序列化后，Word 对象变成 LinkedHashMap
     */
    @SuppressWarnings("unchecked")
    private void distributeRemainingWords(Map<String, Object> gameState, Long player1Id, Long player2Id) {
        List<Map<String, Object>> allWords = (List<Map<String, Object>>) gameState.get("allWords");
        List<Number> grabbedWordIds = (List<Number>) gameState.get("grabbedWords");
        List<Map<String, Object>> player1Words = (List<Map<String, Object>>) gameState.get("player1Words");
        List<Map<String, Object>> player2Words = (List<Map<String, Object>>) gameState.get("player2Words");

        // 找出未被抢的单词
        Set<Long> grabbedIdSet = grabbedWordIds.stream()
                .map(Number::longValue)
                .collect(Collectors.toSet());

        List<Map<String, Object>> remainingWords = new ArrayList<>();
        for (Map<String, Object> word : allWords) {
            long wId = ((Number) word.get("id")).longValue();
            if (!grabbedIdSet.contains(wId)) {
                remainingWords.add(word);
            }
        }

        Collections.shuffle(remainingWords);

        int player1Needed = Math.max(0, wordsPerPlayer - player1Words.size());
        int player2Needed = Math.max(0, wordsPerPlayer - player2Words.size());

        int index = 0;
        while (index < remainingWords.size() && (player1Needed > 0 || player2Needed > 0)) {
            Map<String, Object> word = remainingWords.get(index);
            long wId = ((Number) word.get("id")).longValue();
            if (player1Needed >= player2Needed && player1Needed > 0) {
                player1Words.add(word);
                grabbedWordIds.add(wId);
                player1Needed--;
            } else if (player2Needed > 0) {
                player2Words.add(word);
                grabbedWordIds.add(wId);
                player2Needed--;
            }
            index++;
        }
    }

    /**
     * 处理玩家断线（GRABBING/ANSWERING 阶段）
     */
    public void handlePlayerDisconnect(Long userId) {
        Number roomIdNum = (Number) redisTemplate.opsForValue().get(USER_ROOM_KEY + userId);
        if (roomIdNum == null) return;
        Long roomId = roomIdNum.longValue();

        PlayerState state = stateManager.getState(userId);
        log.info("Player {} disconnected in state {} for room {}", userId, state, roomId);

        if (state == PlayerState.ANSWERING) {
            // 取消房间内所有玩家的定时器（不只是断线玩家的）
            answerService.cancelAllRoomTimers(roomId);
            answerService.handlePlayerDisconnect(roomId, userId);
        }
        // GRABBING 阶段断线的清理也取消定时器
        if (state == PlayerState.GRABBING) {
            answerService.cancelAllRoomTimers(roomId);
        }
    }

    /**
     * 重连/同步状态时恢复游戏页面
     */
    @SuppressWarnings("unchecked")
    public void resumeGame(Long userId) {
        Number roomIdNum = (Number) redisTemplate.opsForValue().get(USER_ROOM_KEY + userId);
        if (roomIdNum == null) {
            return;
        }

        Long roomId = roomIdNum.longValue();
        Map<String, Object> gameState = (Map<String, Object>) redisTemplate.opsForValue().get(GAME_STATE_KEY + roomId);
        if (gameState == null) {
            return;
        }

        Number player1IdNum = (Number) gameState.get("player1Id");
        Number player2IdNum = (Number) gameState.get("player2Id");
        Long player1Id = player1IdNum != null ? player1IdNum.longValue() : null;
        Long player2Id = player2IdNum != null ? player2IdNum.longValue() : null;
        Long opponentId = userId.equals(player1Id) ? player2Id : player1Id;

        boolean soloTraining = SOLO_TRAINING_MODE.equals(gameState.get("mode"));
        Map<String, Object> opponentInfo;
        if (soloTraining) {
            @SuppressWarnings("unchecked")
            Map<String, Object> robotProfile = (Map<String, Object>) gameState.getOrDefault("robotProfile", Map.of());
            opponentInfo = Map.of(
                    "userId", SOLO_ROBOT_ID,
                    "nickname", robotProfile.getOrDefault("name", "训练机器人"),
                    "rank", robotProfile.getOrDefault("tier", "normal")
            );
        } else {
            User opponent = opponentId != null ? userMapper.selectById(opponentId) : null;
            opponentInfo = opponent == null
                    ? Map.of()
                    : Map.of(
                            "userId", opponentId,
                            "nickname", opponent.getNickname(),
                            "rank", opponent.getRank()
                    );
        }

        String phase = (String) gameState.get("phase");
        if ("grab".equals(phase)) {
            Map<String, Object> response = new HashMap<>();
            response.put("roomId", roomId);
            response.put("mode", gameState.get("mode"));
            response.put("phase", "grab");
            response.put("words", gameState.get("allWords"));
            response.put("timeLeft", gameState.getOrDefault("grabTimeLeft", 6));
            response.put("maxWordsPerPlayer", wordsPerPlayer);
            response.put("opponent", opponentInfo);
            copyMatchDifficulty(gameState, response);
            channelManager.sendToPlayer(userId, "game_resume", response);
            return;
        }

        if ("answer".equals(phase)) {
            Map<String, Object> payload = answerService.buildResumePayload(roomId, userId);
            if (payload == null) {
                return;
            }
            Map<String, Object> response = new HashMap<>(payload);
            response.put("opponent", opponentInfo);
            response.put("mode", gameState.get("mode"));
            response.put("robotProfile", gameState.get("robotProfile"));
            copyMatchDifficulty(gameState, response);
            channelManager.sendToPlayer(userId, "game_resume", response);
        }
    }

    private void copyMatchDifficulty(Map<String, Object> gameState, Map<String, Object> response) {
        copyIfPresent(gameState, response, "matchDifficultyGroup");
        copyIfPresent(gameState, response, "matchDifficultyLevel");
        copyIfPresent(gameState, response, "matchDifficultyLabel");
    }

    private void copyIfPresent(Map<String, Object> source, Map<String, Object> target, String key) {
        Object value = source.get(key);
        if (value != null) {
            target.put(key, value);
        }
    }

    private record SoloWordSets(List<Map<String, Object>> playerWords,
                                List<Map<String, Object>> robotWords) {
    }

    /**
     * 将 Word 实体转换为普通 Map，避免 Redis 序列化类型问题
     */
    private Map<String, Object> wordToMap(Word word) {
        Map<String, Object> map = new HashMap<>();
        map.put("id", word.getId());
        map.put("word", word.getWord());
        map.put("meaning", word.getMeaning());
        map.put("difficulty", word.getDifficulty());
        map.put("frequency", word.getFrequency());
        map.put("pronunciationUs", word.getPronunciationUs());
        map.put("pronunciationUk", word.getPronunciationUk());
        return map;
    }
}
