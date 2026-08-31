package com.robword.service;

import com.robword.entity.User;
import com.robword.mapper.RoomMapper;
import com.robword.mapper.UserMapper;
import com.robword.netty.ChannelManager;
import com.robword.state.PlayerState;
import com.robword.state.PlayerStateManager;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.SetOperations;
import org.springframework.data.redis.core.ValueOperations;
import org.springframework.data.redis.core.ZSetOperations;

import java.util.LinkedHashSet;
import java.util.Optional;
import java.util.Set;
import java.util.concurrent.TimeUnit;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyDouble;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.ArgumentMatchers.argThat;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.lenient;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;
import static org.mockito.Mockito.doThrow;

@ExtendWith(MockitoExtension.class)
class MatchServiceTest {

    @Mock private RedisTemplate<String, Object> redisTemplate;
    @Mock private RoomMapper roomMapper;
    @Mock private UserMapper userMapper;
    @Mock private GameService gameService;
    @Mock private PlayerStateManager stateManager;
    @Mock private ChannelManager channelManager;
    @Mock private ValueOperations<String, Object> valueOperations;
    @Mock private ZSetOperations<String, Object> zSetOperations;
    @Mock private SetOperations<String, Object> setOperations;

    private MatchService matchService;
    private TrainingDifficultyCatalog catalog;

    @BeforeEach
    void setUp() {
        catalog = new TrainingDifficultyCatalog();
        matchService = new MatchService(
                redisTemplate,
                roomMapper,
                userMapper,
                gameService,
                stateManager,
                channelManager,
                catalog
        );
        lenient().when(redisTemplate.opsForValue()).thenReturn(valueOperations);
        lenient().when(redisTemplate.opsForZSet()).thenReturn(zSetOperations);
        lenient().when(redisTemplate.opsForSet()).thenReturn(setOperations);
    }

    @Test
    void addsUserToExactDifficultyQueueAndStoresPreference() {
        User user = user(1L, 6);
        when(userMapper.selectById(1L)).thenReturn(user);

        boolean added = matchService.addToMatchQueue(
                1L,
                catalog.resolve("junior", "junior_7_1").orElseThrow()
        );

        assertTrue(added);
        verify(valueOperations).set("match:preference:1", "junior|junior_7_1");
        verify(zSetOperations).add("match:queue:junior_7_1", "1", 6D);
        verify(setOperations).add("match:active_queues", "junior_7_1");
    }

    @Test
    void usersWithSameDifficultyCanMatch() {
        Set<ZSetOperations.TypedTuple<Object>> queue = ordered(tuple("1", 10), tuple("2", 15));
        arrangeScheduler("junior_7_1");
        when(zSetOperations.rangeWithScores("match:queue:junior_7_1", 0, -1)).thenReturn(queue);
        when(zSetOperations.rangeByScoreWithScores(eq("match:queue:junior_7_1"), anyDouble(), anyDouble()))
                .thenReturn(queue);
        preferences("junior|junior_7_1", "junior|junior_7_1");
        matchingAndOnline(1L, 2L);
        when(stateManager.transition(1L, PlayerState.MATCHING, PlayerState.MATCHED)).thenReturn(true);
        when(stateManager.transition(2L, PlayerState.MATCHING, PlayerState.MATCHED)).thenReturn(true);
        when(gameService.startGame(eq(1L), eq(2L), any(), any())).thenReturn(GameService.GameStartResult.ok());

        matchService.checkMatch();

        verify(gameService).startGame(eq(1L), eq(2L), any(), argThat(difficulty ->
                difficulty.group().equals("junior") && difficulty.level().equals("junior_7_1")
        ));
    }

    @Test
    void usersWithDifferentDifficultyNeverMatch() {
        arrangeScheduler("junior_7_1", "junior_7_2");
        when(zSetOperations.rangeWithScores("match:queue:junior_7_1", 0, -1))
                .thenReturn(Set.of(tuple("1", 10)));
        when(zSetOperations.rangeWithScores("match:queue:junior_7_2", 0, -1))
                .thenReturn(Set.of(tuple("2", 10)));
        preferences("junior|junior_7_1", "junior|junior_7_2");
        matchingAndOnline(1L, 2L);

        matchService.checkMatch();

        verify(gameService, never()).startGame(anyLong(), anyLong(), any(), any());
    }

    @Test
    void reconnectUsesOriginalDifficultyQueue() {
        User user = user(1L, 6);
        when(stateManager.getState(1L)).thenReturn(PlayerState.MATCHING);
        when(valueOperations.get("match:preference:1")).thenReturn("junior|junior_7_1");
        when(zSetOperations.score("match:queue:junior_7_1", "1")).thenReturn(null);
        when(userMapper.selectById(1L)).thenReturn(user);

        Optional<TrainingDifficultyCatalog.Difficulty> restored = matchService.handleReconnect(1L);

        assertEquals("junior_7_1", restored.orElseThrow().level());
        verify(zSetOperations).add("match:queue:junior_7_1", "1", 6D);
        verify(setOperations).add("match:active_queues", "junior_7_1");
    }

    @Test
    void matchingUserWithoutPreferenceIsResetAsDirtyState() {
        when(stateManager.getState(1L)).thenReturn(PlayerState.MATCHING);
        when(valueOperations.get("match:preference:1")).thenReturn(null);

        Optional<TrainingDifficultyCatalog.Difficulty> restored = matchService.handleReconnect(1L);

        assertTrue(restored.isEmpty());
        verify(stateManager).forceState(1L, PlayerState.IDLE);
        verify(zSetOperations, never()).add(anyString(), anyString(), anyDouble());
    }

    @Test
    void evictsOfflineCandidateOnlyFromItsDifficultyQueueAfterGraceExpires() {
        Set<ZSetOperations.TypedTuple<Object>> queue = ordered(tuple("2", 6), tuple("1", 6));
        arrangeScheduler("junior_7_1");
        when(zSetOperations.rangeWithScores("match:queue:junior_7_1", 0, -1)).thenReturn(queue);
        preferences("junior|junior_7_1", "junior|junior_7_1");
        when(stateManager.getState(1L)).thenReturn(PlayerState.MATCHING);
        when(stateManager.getState(2L)).thenReturn(PlayerState.MATCHING);
        when(channelManager.isOnline(1L)).thenReturn(true);
        when(channelManager.isOnline(2L)).thenReturn(false);
        when(redisTemplate.hasKey("match:offline:2")).thenReturn(Boolean.FALSE);

        matchService.checkMatch();

        verify(zSetOperations).remove("match:queue:junior_7_1", "2");
        verify(stateManager).forceState(2L, PlayerState.IDLE);
        verify(gameService, never()).startGame(anyLong(), anyLong(), any(), any());
    }

    @Test
    void gameStartExceptionCleansBothPlayersAndPreferences() {
        Set<ZSetOperations.TypedTuple<Object>> queue = ordered(tuple("1", 10), tuple("2", 15));
        arrangeScheduler("junior_7_1");
        when(zSetOperations.rangeWithScores("match:queue:junior_7_1", 0, -1)).thenReturn(queue);
        when(zSetOperations.rangeByScoreWithScores(eq("match:queue:junior_7_1"), anyDouble(), anyDouble()))
                .thenReturn(queue);
        preferences("junior|junior_7_1", "junior|junior_7_1");
        matchingAndOnline(1L, 2L);
        when(stateManager.transition(1L, PlayerState.MATCHING, PlayerState.MATCHED)).thenReturn(true);
        when(stateManager.transition(2L, PlayerState.MATCHING, PlayerState.MATCHED)).thenReturn(true);
        doThrow(new IllegalStateException("boom"))
                .when(gameService).startGame(eq(1L), eq(2L), any(), any());

        matchService.checkMatch();

        verify(stateManager).forceState(1L, PlayerState.IDLE);
        verify(stateManager).forceState(2L, PlayerState.IDLE);
        verify(redisTemplate).delete("match:preference:1");
        verify(redisTemplate).delete("match:preference:2");
    }

    private void arrangeScheduler(String... levels) {
        when(valueOperations.setIfAbsent("match:global_lock", "1", 2, TimeUnit.SECONDS)).thenReturn(true);
        Set<Object> activeLevels = new LinkedHashSet<>();
        activeLevels.addAll(java.util.List.of(levels));
        when(setOperations.members("match:active_queues")).thenReturn(activeLevels);
    }

    private void preferences(String first, String second) {
        when(valueOperations.get("match:preference:1")).thenReturn(first);
        when(valueOperations.get("match:preference:2")).thenReturn(second);
    }

    private void matchingAndOnline(Long... ids) {
        for (Long id : ids) {
            when(stateManager.getState(id)).thenReturn(PlayerState.MATCHING);
            when(channelManager.isOnline(id)).thenReturn(true);
        }
    }

    private User user(Long id, int rank) {
        User user = new User();
        user.setId(id);
        user.setRank(rank);
        return user;
    }

    @SafeVarargs
    private final Set<ZSetOperations.TypedTuple<Object>> ordered(ZSetOperations.TypedTuple<Object>... values) {
        return new LinkedHashSet<>(java.util.List.of(values));
    }

    private ZSetOperations.TypedTuple<Object> tuple(String value, double score) {
        return new ZSetOperations.TypedTuple<>() {
            @Override public Object getValue() { return value; }
            @Override public Double getScore() { return score; }
            @Override public int compareTo(ZSetOperations.TypedTuple<Object> other) { return 0; }
        };
    }
}
