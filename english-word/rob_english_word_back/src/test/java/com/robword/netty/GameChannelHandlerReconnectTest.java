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

import java.util.Optional;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class GameChannelHandlerReconnectTest {

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
    void shouldRestoreMatchQueueWhenMatchingUserReconnects() {
        when(stateManager.getState(1L)).thenReturn(PlayerState.MATCHING);
        when(matchService.handleReconnect(1L)).thenReturn(Optional.of(
                new TrainingDifficultyCatalog().resolve("rank", "rank_current").orElseThrow()
        ));

        PlayerState state = handler.resolveStateOnConnect(1L);

        assertEquals(PlayerState.MATCHING, state);
        verify(matchService).handleReconnect(1L);
        verify(stateManager, never()).forceState(1L, PlayerState.IDLE);
    }
}
