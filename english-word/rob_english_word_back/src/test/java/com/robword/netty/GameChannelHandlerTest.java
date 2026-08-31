package com.robword.netty;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.robword.service.AnswerService;
import com.robword.service.GameService;
import com.robword.service.MatchService;
import com.robword.service.TrainingDifficultyCatalog;
import com.robword.state.PlayerState;
import com.robword.state.PlayerStateManager;
import com.robword.util.JwtUtil;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.data.redis.core.RedisTemplate;

import java.util.Map;
import java.util.Optional;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class GameChannelHandlerTest {

    @Mock
    private ChannelManager channelManager;
    @Mock
    private PlayerStateManager stateManager;
    @Mock
    private MatchService matchService;
    @Mock
    private GameService gameService;
    @Mock
    private AnswerService answerService;
    @Mock
    private JwtUtil jwtUtil;
    @Mock
    private RedisTemplate<String, Object> redisTemplate;

    private GameChannelHandler handler;

    @BeforeEach
    void setUp() {
        handler = new GameChannelHandler(
                channelManager,
                stateManager,
                matchService,
                gameService,
                answerService,
                jwtUtil,
                new ObjectMapper(),
                redisTemplate,
                new TrainingDifficultyCatalog()
        );
    }

    @Test
    void shouldPreserveMatchingStateOnReconnect() {
        when(stateManager.getState(1L)).thenReturn(PlayerState.MATCHING);
        when(matchService.handleReconnect(1L)).thenReturn(Optional.of(
                new TrainingDifficultyCatalog().resolve("rank", "rank_current").orElseThrow()
        ));

        PlayerState state = handler.resolveStateOnConnect(1L);

        assertEquals(PlayerState.MATCHING, state);
        verify(stateManager, never()).forceState(1L, PlayerState.IDLE);
    }

    @Test
    void shouldInitializeIdleWhenNoStateExists() {
        when(stateManager.getState(1L)).thenReturn(null);

        PlayerState state = handler.resolveStateOnConnect(1L);

        assertEquals(PlayerState.IDLE, state);
        verify(stateManager).forceState(1L, PlayerState.IDLE);
    }

    @Test
    void shouldSendMatchingSnapshotWithWaitingMessage() {
        when(matchService.getPreference(1L)).thenReturn(Optional.of(
                new TrainingDifficultyCatalog().resolve("junior", "junior_7_1").orElseThrow()
        ));

        handler.pushStateSnapshot(1L, PlayerState.MATCHING);

        verify(channelManager).sendToPlayer(1L, "state_change", Map.of("state", "MATCHING"));
        verify(channelManager).sendToPlayer(1L, "match_waiting", Map.of(
                "difficultyGroup", "junior",
                "difficultyLevel", "junior_7_1",
                "difficultyLabel", "初中英语 · 7年级上册"
        ));
    }

    @Test
    void shouldPassCanonicalDifficultyToMatchService() {
        when(stateManager.getState(1L)).thenReturn(PlayerState.IDLE);
        when(stateManager.transition(1L, PlayerState.IDLE, PlayerState.MATCHING)).thenReturn(true);
        when(matchService.addToMatchQueue(
                org.mockito.ArgumentMatchers.eq(1L),
                org.mockito.ArgumentMatchers.any()
        )).thenReturn(true);

        handler.handleMatchStart(1L, Map.of(
                "difficultyGroup", "junior",
                "difficultyLevel", "junior_7_1"
        ));

        verify(matchService).addToMatchQueue(
                org.mockito.ArgumentMatchers.eq(1L),
                org.mockito.ArgumentMatchers.argThat(difficulty ->
                        difficulty.group().equals("junior")
                                && difficulty.level().equals("junior_7_1")
                )
        );
    }

    @Test
    void shouldRejectInvalidDifficultyBeforeStateTransition() {
        handler.handleMatchStart(1L, Map.of(
                "difficultyGroup", "junior",
                "difficultyLevel", "senior_1"
        ));

        verify(stateManager, never()).transition(1L, PlayerState.IDLE, PlayerState.MATCHING);
        verify(matchService, never()).addToMatchQueue(
                org.mockito.ArgumentMatchers.anyLong(),
                org.mockito.ArgumentMatchers.any()
        );
        verify(channelManager).sendToPlayer(
                org.mockito.ArgumentMatchers.eq(1L),
                org.mockito.ArgumentMatchers.eq("error"),
                org.mockito.ArgumentMatchers.argThat(data ->
                        data instanceof Map<?, ?> map
                                && map.get("message").toString().contains("难度")
                )
        );
    }

    @Test
    void shouldRejectChangingDifficultyWhileAlreadyMatching() {
        TrainingDifficultyCatalog catalog = new TrainingDifficultyCatalog();
        when(stateManager.getState(1L)).thenReturn(PlayerState.MATCHING);
        when(matchService.getPreference(1L)).thenReturn(catalog.resolve("junior", "junior_7_1"));

        handler.handleMatchStart(1L, Map.of(
                "difficultyGroup", "junior",
                "difficultyLevel", "junior_7_2"
        ));

        verify(matchService, never()).handleReconnect(1L);
        verify(channelManager).sendToPlayer(1L, "error", Map.of("message", "请先取消当前匹配"));
    }

    @Test
    void shouldReplyPongToHeartbeat() {
        handler.handlePing(1L);

        verify(stateManager).refreshStateTtl(1L);
        verify(channelManager).sendToPlayer(1L, "pong", Map.of());
    }

    @Test
    void shouldResumeGameWhenSyncingGrabState() {
        when(stateManager.getState(1L)).thenReturn(PlayerState.GRABBING);

        handler.handleSyncState(1L);

        verify(channelManager).sendToPlayer(1L, "state_change", Map.of("state", "GRABBING"));
        verify(gameService).resumeGame(1L);
    }
}
