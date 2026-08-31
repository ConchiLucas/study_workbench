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
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.ArgumentCaptor;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.ValueOperations;
import org.springframework.test.util.ReflectionTestUtils;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.mockito.ArgumentMatchers.anyInt;
import static org.mockito.ArgumentMatchers.anyList;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.lenient;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class GameServiceTest {

    @Mock
    private RedisTemplate<String, Object> redisTemplate;
    @Mock
    private ValueOperations<String, Object> valueOperations;
    @Mock
    private RoomMapper roomMapper;
    @Mock
    private UserMapper userMapper;
    @Mock
    private WordService wordService;
    @Mock
    private ChannelManager channelManager;
    @Mock
    private PlayerStateManager stateManager;
    @Mock
    private AnswerService answerService;
    @Mock
    private GameSettlementService settlementService;
    @Mock
    private UserWordMasteryService userWordMasteryService;

    private GameService gameService;

    @BeforeEach
    void setUp() {
        gameService = new GameService(
                redisTemplate,
                roomMapper,
                userMapper,
                wordService,
                channelManager,
                stateManager,
                answerService,
                settlementService,
                userWordMasteryService,
                new ObjectMapper(),
                new TrainingDifficultyCatalog()
        );
        ReflectionTestUtils.setField(gameService, "wordsPerPlayer", 5);
    }

    @Test
    void shouldAbortStartGameWhenAnyPlayerChannelIsOffline() {
        Room room = new Room();
        room.setId(233L);

        when(stateManager.transition(1L, PlayerState.MATCHED, PlayerState.GRABBING)).thenReturn(true);
        when(stateManager.transition(2L, PlayerState.MATCHED, PlayerState.GRABBING)).thenReturn(true);
        when(channelManager.isOnline(1L)).thenReturn(false);

        gameService.startGame(1L, 2L, room,
                new TrainingDifficultyCatalog().resolve("rank", "rank_current").orElseThrow());

        verify(stateManager).forceState(1L, PlayerState.IDLE);
        verify(stateManager).forceState(2L, PlayerState.IDLE);
        verify(userMapper, never()).selectById(1L);
        verify(roomMapper).deleteById(233L);
    }

    @Test
    void shouldNotifyPlayersBackToIdleWhenGameStartPushFails() {
        Room room = new Room();
        room.setId(233L);

        lenient().when(redisTemplate.opsForValue()).thenReturn(valueOperations);
        User player1 = new User();
        player1.setId(1L);
        player1.setNickname("p1");
        player1.setRank(6);

        User player2 = new User();
        player2.setId(2L);
        player2.setNickname("p2");
        player2.setRank(9);

        when(stateManager.transition(1L, PlayerState.MATCHED, PlayerState.GRABBING)).thenReturn(true);
        when(stateManager.transition(2L, PlayerState.MATCHED, PlayerState.GRABBING)).thenReturn(true);
        when(channelManager.isOnline(1L)).thenReturn(true);
        when(channelManager.isOnline(2L)).thenReturn(true);
        when(userMapper.selectById(1L)).thenReturn(player1);
        when(userMapper.selectById(2L)).thenReturn(player2);
        when(channelManager.sendToPlayer(1L, "state_change", Map.of("state", "GRABBING"))).thenReturn(true);
        when(channelManager.sendToPlayer(2L, "state_change", Map.of("state", "GRABBING"))).thenReturn(true);
        when(channelManager.sendToPlayer(org.mockito.ArgumentMatchers.eq(1L), org.mockito.ArgumentMatchers.eq("game_start"), org.mockito.ArgumentMatchers.any()))
                .thenReturn(true);
        when(channelManager.sendToPlayer(org.mockito.ArgumentMatchers.eq(2L), org.mockito.ArgumentMatchers.eq("game_start"), org.mockito.ArgumentMatchers.any()))
                .thenReturn(false);

        gameService.startGame(1L, 2L, room,
                new TrainingDifficultyCatalog().resolve("rank", "rank_current").orElseThrow());

        verify(stateManager).forceState(1L, PlayerState.IDLE);
        verify(stateManager).forceState(2L, PlayerState.IDLE);
        verify(channelManager).sendToPlayer(1L, "state_change", Map.of("state", "IDLE"));
        verify(channelManager).sendToPlayer(2L, "state_change", Map.of("state", "IDLE"));
        verify(roomMapper).deleteById(233L);
    }

    @Test
    void formalMatchUsesOnlySelectedDifficultyLibraries() {
        Room room = readyRoomAndPlayers();
        List<Word> words = words(10);
        when(wordService.getRandomWordsForTrainingLibraries(List.of("PEPChuZhong7_1"), 10)).thenReturn(words);

        GameService.GameStartResult result = gameService.startGame(
                1L,
                2L,
                room,
                new TrainingDifficultyCatalog().resolve("junior", "junior_7_1").orElseThrow()
        );

        assertTrue(result.success());
        verify(wordService).getRandomWordsForTrainingLibraries(List.of("PEPChuZhong7_1"), 10);
        verify(wordService, never()).getRandomWordsForMatch(anyInt(), anyInt(), anyInt());
    }

    @Test
    void insufficientSelectedDifficultyWordsFailsWithoutFallback() {
        Room room = readyRoomAndPlayers();
        when(wordService.getRandomWordsForTrainingLibraries(anyList(), eq(10))).thenReturn(words(9));

        GameService.GameStartResult result = gameService.startGame(
                1L,
                2L,
                room,
                new TrainingDifficultyCatalog().resolve("junior", "junior_7_1").orElseThrow()
        );

        assertFalse(result.success());
        verify(wordService, never()).getRandomWordsForMatch(anyInt(), anyInt(), anyInt());
        verify(roomMapper).deleteById(233L);
    }

    @Test
    void grabResumeIncludesFormalMatchDifficulty() {
        when(redisTemplate.opsForValue()).thenReturn(valueOperations);
        when(valueOperations.get("game:user_room:1")).thenReturn(233L);
        when(valueOperations.get("game:state:233")).thenReturn(Map.of(
                "roomId", 233L,
                "mode", "match",
                "player1Id", 1L,
                "player2Id", 2L,
                "phase", "grab",
                "allWords", List.of(),
                "grabTimeLeft", 4,
                "matchDifficultyGroup", "junior",
                "matchDifficultyLevel", "junior_7_1",
                "matchDifficultyLabel", "初中英语 · 7年级上册"
        ));
        User opponent = new User();
        opponent.setId(2L);
        opponent.setNickname("p2");
        opponent.setRank(9);
        when(userMapper.selectById(2L)).thenReturn(opponent);

        gameService.resumeGame(1L);

        ArgumentCaptor<Object> payload = ArgumentCaptor.forClass(Object.class);
        verify(channelManager).sendToPlayer(eq(1L), eq("game_resume"), payload.capture());
        @SuppressWarnings("unchecked")
        Map<String, Object> response = (Map<String, Object>) payload.getValue();
        assertEquals("junior", response.get("matchDifficultyGroup"));
        assertEquals("junior_7_1", response.get("matchDifficultyLevel"));
        assertEquals("初中英语 · 7年级上册", response.get("matchDifficultyLabel"));
    }

    private Room readyRoomAndPlayers() {
        Room room = new Room();
        room.setId(233L);
        lenient().when(redisTemplate.opsForValue()).thenReturn(valueOperations);

        User player1 = new User();
        player1.setId(1L);
        player1.setNickname("p1");
        player1.setRank(6);
        User player2 = new User();
        player2.setId(2L);
        player2.setNickname("p2");
        player2.setRank(9);

        when(stateManager.transition(1L, PlayerState.MATCHED, PlayerState.GRABBING)).thenReturn(true);
        when(stateManager.transition(2L, PlayerState.MATCHED, PlayerState.GRABBING)).thenReturn(true);
        when(channelManager.isOnline(1L)).thenReturn(true);
        when(channelManager.isOnline(2L)).thenReturn(true);
        when(userMapper.selectById(1L)).thenReturn(player1);
        when(userMapper.selectById(2L)).thenReturn(player2);
        when(channelManager.sendToPlayer(eq(1L), eq("state_change"), org.mockito.ArgumentMatchers.any())).thenReturn(true);
        when(channelManager.sendToPlayer(eq(2L), eq("state_change"), org.mockito.ArgumentMatchers.any())).thenReturn(true);
        lenient().when(channelManager.sendToPlayer(eq(1L), eq("game_start"), org.mockito.ArgumentMatchers.any())).thenReturn(true);
        lenient().when(channelManager.sendToPlayer(eq(2L), eq("game_start"), org.mockito.ArgumentMatchers.any())).thenReturn(true);
        return room;
    }

    private List<Word> words(int count) {
        List<Word> words = new ArrayList<>();
        for (int i = 1; i <= count; i++) {
            Word word = new Word();
            word.setId((long) i);
            word.setWord("word" + i);
            word.setDifficulty(100 + i);
            words.add(word);
        }
        return words;
    }
}
