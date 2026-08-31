package com.robword.service;

import com.robword.entity.Room;
import com.robword.entity.User;
import com.robword.mapper.RoomMapper;
import com.robword.mapper.UserMapper;
import com.robword.netty.ChannelManager;
import com.robword.state.PlayerState;
import com.robword.state.PlayerStateManager;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.ZSetOperations;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;

import java.util.Map;
import java.util.Optional;
import java.util.Set;
import java.util.UUID;
import java.util.concurrent.TimeUnit;

/**
 * 按精确难度拆分的匹配服务。
 * 同一难度队列内继续使用段位分数寻找实力接近的对手。
 */
@Service
@RequiredArgsConstructor
@Slf4j
public class MatchService {

    @Value("${game.matching.reconnect-grace-seconds:15}")
    private long reconnectGraceSeconds;

    private final RedisTemplate<String, Object> redisTemplate;
    private final RoomMapper roomMapper;
    private final UserMapper userMapper;
    private final GameService gameService;
    private final PlayerStateManager stateManager;
    private final ChannelManager channelManager;
    private final TrainingDifficultyCatalog difficultyCatalog;

    private static final String MATCH_QUEUE_PREFIX = "match:queue:";
    private static final String MATCH_PREFERENCE_PREFIX = "match:preference:";
    private static final String MATCH_ACTIVE_QUEUES_KEY = "match:active_queues";
    private static final String MATCH_GLOBAL_LOCK = "match:global_lock";
    private static final String MATCH_OFFLINE_KEY_PREFIX = "match:offline:";

    /** 状态转换由 GameChannelHandler 在调用前完成。 */
    public boolean addToMatchQueue(Long userId, TrainingDifficultyCatalog.Difficulty difficulty) {
        User user = userMapper.selectById(userId);
        if (user == null) {
            log.error("User {} not found, cannot add to match queue", userId);
            stateManager.forceState(userId, PlayerState.IDLE);
            channelManager.sendToPlayer(userId, "error", Map.of("message", "用户不存在，请重新登录"));
            return false;
        }
        int rank = user.getRank() == null ? 0 : user.getRank();
        redisTemplate.opsForValue().set(preferenceKey(userId), serialize(difficulty));
        redisTemplate.opsForZSet().add(queueKey(difficulty.level()), userId.toString(), rank);
        redisTemplate.opsForSet().add(MATCH_ACTIVE_QUEUES_KEY, difficulty.level());
        redisTemplate.delete(MATCH_OFFLINE_KEY_PREFIX + userId);
        log.info("User {} (rank={}) added to difficulty queue {}", userId, rank, difficulty.level());
        return true;
    }

    public Optional<TrainingDifficultyCatalog.Difficulty> getPreference(Long userId) {
        Object stored = redisTemplate.opsForValue().get(preferenceKey(userId));
        if (stored == null) {
            return Optional.empty();
        }
        String[] parts = stored.toString().split("\\|", 2);
        if (parts.length != 2) {
            return Optional.empty();
        }
        return difficultyCatalog.resolve(parts[0], parts[1]);
    }

    /** 用户主动取消匹配，队列和偏好一起清除。 */
    public void removeFromMatchQueue(Long userId) {
        getPreference(userId).ifPresent(difficulty -> removeQueueMember(userId, difficulty));
        clearPreference(userId);
        redisTemplate.delete(MATCH_OFFLINE_KEY_PREFIX + userId);
        log.info("User {} removed from match queue", userId);
    }

    public void handleDisconnect(Long userId) {
        if (getPreference(userId).isEmpty()) {
            stateManager.forceState(userId, PlayerState.IDLE);
            return;
        }
        redisTemplate.opsForValue().set(
                MATCH_OFFLINE_KEY_PREFIX + userId,
                "1",
                reconnectGraceSeconds,
                TimeUnit.SECONDS
        );
        log.info("User {} disconnected while matching, waiting {}s for reconnect grace", userId, reconnectGraceSeconds);
    }

    /** 匹配中重连；缺失偏好视为脏状态，不猜测默认难度。 */
    public Optional<TrainingDifficultyCatalog.Difficulty> handleReconnect(Long userId) {
        redisTemplate.delete(MATCH_OFFLINE_KEY_PREFIX + userId);
        if (stateManager.getState(userId) != PlayerState.MATCHING) {
            return Optional.empty();
        }

        Optional<TrainingDifficultyCatalog.Difficulty> preference = getPreference(userId);
        if (preference.isEmpty()) {
            log.warn("User {} is MATCHING without a difficulty preference; resetting to IDLE", userId);
            stateManager.forceState(userId, PlayerState.IDLE);
            return Optional.empty();
        }

        TrainingDifficultyCatalog.Difficulty difficulty = preference.get();
        String queueKey = queueKey(difficulty.level());
        Double score = redisTemplate.opsForZSet().score(queueKey, userId.toString());
        if (score == null) {
            User user = userMapper.selectById(userId);
            if (user == null) {
                removeFromMatchQueue(userId);
                stateManager.forceState(userId, PlayerState.IDLE);
                return Optional.empty();
            }
            int rank = user.getRank() == null ? 0 : user.getRank();
            redisTemplate.opsForZSet().add(queueKey, userId.toString(), rank);
            redisTemplate.opsForSet().add(MATCH_ACTIVE_QUEUES_KEY, difficulty.level());
        }
        return preference;
    }

    @Scheduled(fixedRate = 500)
    public void checkMatch() {
        Boolean locked = redisTemplate.opsForValue()
                .setIfAbsent(MATCH_GLOBAL_LOCK, "1", 2, TimeUnit.SECONDS);
        if (!Boolean.TRUE.equals(locked)) {
            return;
        }

        try {
            doMatch();
        } catch (Exception e) {
            log.error("Error in checkMatch: {}", e.getMessage(), e);
        } finally {
            redisTemplate.delete(MATCH_GLOBAL_LOCK);
        }
    }

    private void doMatch() {
        Set<Object> activeLevels = redisTemplate.opsForSet().members(MATCH_ACTIVE_QUEUES_KEY);
        if (activeLevels == null || activeLevels.isEmpty()) {
            return;
        }

        for (Object activeLevel : activeLevels) {
            String level = activeLevel.toString();
            Optional<TrainingDifficultyCatalog.Difficulty> difficulty = difficultyCatalog.resolveLevel(level);
            if (difficulty.isEmpty()) {
                redisTemplate.opsForSet().remove(MATCH_ACTIVE_QUEUES_KEY, level);
                continue;
            }
            if (doMatchInQueue(difficulty.get())) {
                return; // 每次调度最多启动一场，控制数据库和推送压力
            }
        }
    }

    private boolean doMatchInQueue(TrainingDifficultyCatalog.Difficulty difficulty) {
        String queueKey = queueKey(difficulty.level());
        Set<ZSetOperations.TypedTuple<Object>> allCandidates = redisTemplate.opsForZSet()
                .rangeWithScores(queueKey, 0, -1);

        if (allCandidates == null || allCandidates.isEmpty()) {
            redisTemplate.opsForSet().remove(MATCH_ACTIVE_QUEUES_KEY, difficulty.level());
            return false;
        }
        if (allCandidates.size() < 2) {
            evictInvalidCandidates(allCandidates, difficulty);
            return false;
        }

        for (ZSetOperations.TypedTuple<Object> candidate : allCandidates) {
            Long userId = Long.parseLong(candidate.getValue().toString());
            Double rank = candidate.getScore();
            if (rank == null) {
                continue;
            }
            if (shouldEvictCandidate(userId, difficulty)) {
                evictCandidate(userId, difficulty);
                continue;
            }
            if (!isMatchEligible(userId)) {
                continue;
            }

            Long matchedUserId = findMatch(queueKey, difficulty, userId, rank.intValue());
            if (matchedUserId == null) {
                continue;
            }

            boolean cas1 = stateManager.transition(userId, PlayerState.MATCHING, PlayerState.MATCHED);
            if (!cas1) {
                evictCandidate(userId, difficulty);
                continue;
            }
            boolean cas2 = stateManager.transition(matchedUserId, PlayerState.MATCHING, PlayerState.MATCHED);
            if (!cas2) {
                stateManager.transition(userId, PlayerState.MATCHED, PlayerState.MATCHING);
                evictCandidate(matchedUserId, difficulty);
                continue;
            }

            removeQueueMember(userId, difficulty);
            removeQueueMember(matchedUserId, difficulty);
            Room room = null;
            try {
                room = createRoom(userId, matchedUserId);
                GameService.GameStartResult result = gameService.startGame(userId, matchedUserId, room, difficulty);
                clearPreference(userId);
                clearPreference(matchedUserId);
                if (!result.success()) {
                    sendMatchError(userId, matchedUserId, result.message());
                }
            } catch (Exception e) {
                log.error("Failed to start matched game for users {} and {}", userId, matchedUserId, e);
                if (room != null && room.getId() != null) {
                    roomMapper.deleteById(room.getId());
                }
                clearPreference(userId);
                clearPreference(matchedUserId);
                stateManager.forceState(userId, PlayerState.IDLE);
                stateManager.forceState(matchedUserId, PlayerState.IDLE);
                channelManager.sendToPlayer(userId, "state_change", Map.of("state", "IDLE"));
                channelManager.sendToPlayer(matchedUserId, "state_change", Map.of("state", "IDLE"));
                sendMatchError(userId, matchedUserId, "游戏启动失败，请重新匹配");
            }
            return true;
        }
        return false;
    }

    private void evictInvalidCandidates(Set<ZSetOperations.TypedTuple<Object>> candidates,
                                        TrainingDifficultyCatalog.Difficulty difficulty) {
        for (ZSetOperations.TypedTuple<Object> candidate : candidates) {
            Long userId = Long.parseLong(candidate.getValue().toString());
            if (shouldEvictCandidate(userId, difficulty)) {
                evictCandidate(userId, difficulty);
            }
        }
    }

    private Long findMatch(String queueKey,
                           TrainingDifficultyCatalog.Difficulty difficulty,
                           Long userId,
                           int rank) {
        for (int range = 0; range <= 50; range += 5) {
            Set<ZSetOperations.TypedTuple<Object>> candidates = redisTemplate.opsForZSet()
                    .rangeByScoreWithScores(queueKey, rank - range, rank + range);
            if (candidates == null) {
                continue;
            }
            for (ZSetOperations.TypedTuple<Object> candidate : candidates) {
                Long candidateId = Long.parseLong(candidate.getValue().toString());
                if (candidateId.equals(userId)) {
                    continue;
                }
                if (shouldEvictCandidate(candidateId, difficulty)) {
                    evictCandidate(candidateId, difficulty);
                    continue;
                }
                if (isMatchEligible(candidateId)) {
                    return candidateId;
                }
            }
        }
        return null;
    }

    private boolean shouldEvictCandidate(Long userId, TrainingDifficultyCatalog.Difficulty queueDifficulty) {
        Optional<TrainingDifficultyCatalog.Difficulty> preference = getPreference(userId);
        if (preference.isEmpty() || !preference.get().level().equals(queueDifficulty.level())) {
            return true;
        }
        PlayerState state = stateManager.getState(userId);
        boolean online = channelManager.isOnline(userId);
        boolean hasGraceKey = Boolean.TRUE.equals(redisTemplate.hasKey(MATCH_OFFLINE_KEY_PREFIX + userId));
        return state != PlayerState.MATCHING || (!online && !hasGraceKey);
    }

    private void evictCandidate(Long userId, TrainingDifficultyCatalog.Difficulty difficulty) {
        removeQueueMember(userId, difficulty);
        clearPreference(userId);
        stateManager.forceState(userId, PlayerState.IDLE);
        log.info("Evicted user {} from difficulty queue {}", userId, difficulty.level());
    }

    private boolean isMatchEligible(Long userId) {
        return stateManager.getState(userId) == PlayerState.MATCHING && channelManager.isOnline(userId);
    }

    private void removeQueueMember(Long userId, TrainingDifficultyCatalog.Difficulty difficulty) {
        String queueKey = queueKey(difficulty.level());
        redisTemplate.opsForZSet().remove(queueKey, userId.toString());
        redisTemplate.delete(MATCH_OFFLINE_KEY_PREFIX + userId);
        Long remaining = redisTemplate.opsForZSet().size(queueKey);
        if (remaining != null && remaining == 0) {
            redisTemplate.opsForSet().remove(MATCH_ACTIVE_QUEUES_KEY, difficulty.level());
        }
    }

    private void clearPreference(Long userId) {
        redisTemplate.delete(preferenceKey(userId));
    }

    private void sendMatchError(Long userId, Long matchedUserId, String message) {
        channelManager.sendToPlayer(userId, "error", Map.of("message", message));
        channelManager.sendToPlayer(matchedUserId, "error", Map.of("message", message));
    }

    private String serialize(TrainingDifficultyCatalog.Difficulty difficulty) {
        return difficulty.group() + "|" + difficulty.level();
    }

    private String queueKey(String level) {
        return MATCH_QUEUE_PREFIX + level;
    }

    private String preferenceKey(Long userId) {
        return MATCH_PREFERENCE_PREFIX + userId;
    }

    private Room createRoom(Long player1Id, Long player2Id) {
        Room room = new Room();
        room.setRoomCode(UUID.randomUUID().toString().substring(0, 8).toUpperCase());
        room.setStatus(0);
        room.setPlayer1Id(player1Id);
        room.setPlayer2Id(player2Id);
        roomMapper.insert(room);
        return room;
    }
}
