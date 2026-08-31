package com.robword.state;

import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.core.script.DefaultRedisScript;
import org.springframework.stereotype.Component;

import java.util.Collections;
import java.util.concurrent.TimeUnit;

/**
 * 玩家状态管理器
 * 使用 Redis Lua 脚本实现原子 CAS（Compare-And-Swap）状态转换
 * 注意：使用 StringRedisTemplate 避免 JSON 序列化导致 Lua 脚本字符串比较不匹配
 */
@Component
@Slf4j
public class PlayerStateManager {

    private final StringRedisTemplate redisTemplate;

    private static final String STATE_KEY_PREFIX = "player:state:";

    @Value("${game.player-state.ttl-seconds:300}")
    private long stateTtlSeconds;

    /**
     * Lua CAS 脚本：仅当当前值等于期望值时才设置新值
     * KEYS[1] = player:state:{userId}
     * ARGV[1] = expectedState
     * ARGV[2] = newState
     * ARGV[3] = TTL秒
     * 返回 1=成功，0=失败
     */
    private static final String CAS_LUA_SCRIPT =
            "local cur = redis.call('GET', KEYS[1]) " +
            "if cur == ARGV[1] then " +
            "    redis.call('SET', KEYS[1], ARGV[2], 'EX', tonumber(ARGV[3])) " +
            "    return 1 " +
            "end " +
            "return 0";

    private final DefaultRedisScript<Long> casScript;

    public PlayerStateManager(StringRedisTemplate redisTemplate) {
        this.redisTemplate = redisTemplate;
        this.casScript = new DefaultRedisScript<>(CAS_LUA_SCRIPT, Long.class);
    }

    /**
     * 获取玩家当前状态
     */
    public PlayerState getState(Long userId) {
        String key = STATE_KEY_PREFIX + userId;
        String value = redisTemplate.opsForValue().get(key);
        if (value == null) {
            return null;
        }
        try {
            return PlayerState.valueOf(value);
        } catch (IllegalArgumentException e) {
            log.warn("Invalid player state for user {}: {}", userId, value);
            return null;
        }
    }

    /**
     * 原子 CAS 状态转换
     * @return true 转换成功，false 当前状态不匹配
     */
    public boolean transition(Long userId, PlayerState expectedState, PlayerState newState) {
        if (!expectedState.canTransitionTo(newState)) {
            log.warn("Illegal state transition for user {}: {} → {}", userId, expectedState, newState);
            return false;
        }

        String key = STATE_KEY_PREFIX + userId;
        Long result = redisTemplate.execute(
                casScript,
                Collections.singletonList(key),
                expectedState.name(),
                newState.name(),
                String.valueOf(stateTtlSeconds)
        );

        boolean success = result != null && result == 1;
        if (success) {
            log.info("State transition for user {}: {} → {}", userId, expectedState, newState);
        } else {
            PlayerState actual = getState(userId);
            log.warn("State transition failed for user {}: expected {} but was {}, target was {}",
                    userId, expectedState, actual, newState);
        }
        return success;
    }

    /**
     * 强制设置状态（仅用于连接建立/断开等已知安全场景）
     */
    public void forceState(Long userId, PlayerState state) {
        String key = STATE_KEY_PREFIX + userId;
        redisTemplate.opsForValue().set(key, state.name(), stateTtlSeconds, TimeUnit.SECONDS);
        log.info("Force state for user {}: {}", userId, state);
    }

    /**
     * 续期玩家当前状态，避免长时间停留在同一阶段时状态 key 过期
     */
    public void refreshStateTtl(Long userId) {
        String key = STATE_KEY_PREFIX + userId;
        Boolean refreshed = redisTemplate.expire(key, stateTtlSeconds, TimeUnit.SECONDS);
        if (Boolean.TRUE.equals(refreshed)) {
            log.debug("Refreshed state TTL for user {} to {}s", userId, stateTtlSeconds);
        }
    }

    /**
     * 清除玩家状态（断开连接时调用）
     */
    public void clearState(Long userId) {
        String key = STATE_KEY_PREFIX + userId;
        redisTemplate.delete(key);
        log.info("Cleared state for user {}", userId);
    }
}
