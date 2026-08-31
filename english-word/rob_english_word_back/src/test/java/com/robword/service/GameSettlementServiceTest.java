package com.robword.service;

import com.robword.entity.User;
import com.robword.entity.GameAnswerDetail;
import com.robword.entity.GameRecord;
import com.robword.mapper.GameAnswerDetailMapper;
import com.robword.mapper.GameRecordMapper;
import com.robword.mapper.RoomMapper;
import com.robword.mapper.UserMapper;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.ArgumentCaptor;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.ValueOperations;
import org.springframework.test.util.ReflectionTestUtils;

import java.time.LocalDateTime;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class GameSettlementServiceTest {

    @Mock
    private RedisTemplate<String, Object> redisTemplate;

    @Mock
    private ValueOperations<String, Object> valueOperations;

    @Mock
    private RoomMapper roomMapper;

    @Mock
    private UserMapper userMapper;

    @Mock
    private GameRecordMapper gameRecordMapper;

    @Mock
    private GameAnswerDetailMapper gameAnswerDetailMapper;

    @Mock
    private UserWordMasteryService userWordMasteryService;

    @Mock
    private WrongWordReviewProgressService wrongWordReviewProgressService;

    private GameSettlementService service;

    @BeforeEach
    void setUp() {
        service = new GameSettlementService(
                redisTemplate,
                roomMapper,
                userMapper,
                gameRecordMapper,
                gameAnswerDetailMapper,
                userWordMasteryService
        );
        ReflectionTestUtils.setField(
                service,
                "wrongWordReviewProgressService",
                wrongWordReviewProgressService
        );
    }

    @Test
    void shouldActivateWordReviewProgressBeforeNotifyingWrongAnswer() {
        LocalDateTime wrongAt = LocalDateTime.of(2026, 7, 26, 20, 30);
        User user = buildUser(1L, 1, 0, 0, 0);
        when(userMapper.selectById(1L)).thenReturn(user);
        when(gameAnswerDetailMapper.insert(any(GameAnswerDetail.class))).thenAnswer(invocation -> {
            GameAnswerDetail detail = invocation.getArgument(0);
            detail.setId(501L);
            detail.setCreateTime(wrongAt);
            return 1;
        });

        Map<String, Object> answer = new HashMap<>();
        answer.put("roundNo", 1);
        answer.put("wordId", 12L);
        answer.put("wordContent", "momentum");
        answer.put("wordDifficulty", 486);
        answer.put("options", List.of("动力", "石头", "排水", "军队"));
        answer.put("correctAnswerIndex", 1);
        answer.put("selectedAnswerIndex", 2);
        answer.put("isCorrect", false);
        answer.put("score", 0);
        answer.put("answerTimeMs", 1200);

        service.saveSingleAnswerDetail(99L, 1L, answer);

        verify(wrongWordReviewProgressService)
                .recordWrong(1L, 12L, "momentum", wrongAt, 501L);
    }

    @Test
    void shouldRewardHighQualitySecondStreakWin() {
        User user = buildUser(1L, 1, 100, 0, 0);
        user.setCurrentWinStreak(1);
        GameSettlementService.PlayerResult self = buildResult(1L, 1600, 2000, 4, 5);
        GameSettlementService.PlayerResult opponent = buildResult(2L, 1200, 2000, 3, 5);

        GameSettlementService.SettlementOutcome outcome =
                service.updateUserStats(user, self, opponent, 2000.0, 1L, false);

        assertEquals(8, outcome.masteryExp);
        assertEquals(6, outcome.battleExp);
        assertEquals(4, outcome.streakExp);
        assertEquals(18, outcome.expChange);
        assertEquals(2, outcome.effectiveStreak());
        assertTrue(outcome.effectiveWin());
        assertEquals(118, user.getExp());
        assertEquals(2, user.getRank());
        assertEquals(1, user.getTotalWins());
        assertEquals(1, user.getTotalGames());
        assertEquals(2, user.getCurrentWinStreak());
        verify(userMapper).updateById(user);
    }

    @Test
    void shouldCopyFormalMatchDifficultyIntoInitialRecord() {
        User player1 = buildUser(1L, 1, 0, 0, 0);
        player1.setNickname("p1");
        User player2 = buildUser(2L, 1, 0, 0, 0);
        player2.setNickname("p2");
        Map<String, Object> gameState = Map.of(
                "player1Id", 1L,
                "player2Id", 2L,
                "startTime", 1_700_000_000L,
                "matchDifficultyGroup", "junior",
                "matchDifficultyLevel", "junior_7_1",
                "matchDifficultyLabel", "初中英语 · 7年级上册"
        );
        when(redisTemplate.opsForValue()).thenReturn(valueOperations);
        when(valueOperations.get("game:state:99")).thenReturn(gameState);
        when(userMapper.selectById(1L)).thenReturn(player1);
        when(userMapper.selectById(2L)).thenReturn(player2);

        service.createInitialGameRecord(99L);

        ArgumentCaptor<GameRecord> captor = ArgumentCaptor.forClass(GameRecord.class);
        verify(gameRecordMapper).insert(captor.capture());
        GameRecord record = captor.getValue();
        assertEquals("junior", record.getMatchDifficultyGroup());
        assertEquals("junior_7_1", record.getMatchDifficultyLevel());
        assertEquals("初中英语 · 7年级上册", record.getMatchDifficultyLabel());
    }

    @Test
    void shouldProtectHighMasteryLoserAndResetStreak() {
        User user = buildUser(1L, 3, 100, 5, 9);
        user.setCurrentWinStreak(3);
        GameSettlementService.PlayerResult self = buildResult(1L, 1804, 2200, 4, 5);
        GameSettlementService.PlayerResult opponent = buildResult(2L, 2900, 3000, 5, 5);

        GameSettlementService.SettlementOutcome outcome =
                service.updateUserStats(user, self, opponent, 2000.0, 2L, false);

        assertEquals(13, outcome.masteryExp);
        assertEquals(-8, outcome.battleExp);
        assertEquals(0, outcome.streakExp);
        assertEquals(8, outcome.expChange);
        assertEquals(0, outcome.effectiveStreak());
        assertFalse(outcome.effectiveWin());
        assertEquals(108, user.getExp());
        assertEquals(2, user.getRank());
        assertEquals(5, user.getTotalWins());
        assertEquals(10, user.getTotalGames());
        assertEquals(0, user.getCurrentWinStreak());
        verify(userMapper).updateById(user);
    }

    @Test
    void shouldResetStreakForLowQualityWinEvenWhenPlayerWins() {
        User user = buildUser(1L, 2, 200, 3, 6);
        user.setCurrentWinStreak(4);
        GameSettlementService.PlayerResult self = buildResult(1L, 720, 1600, 2, 5);
        GameSettlementService.PlayerResult opponent = buildResult(2L, 560, 1600, 1, 5);

        GameSettlementService.SettlementOutcome outcome =
                service.updateUserStats(user, self, opponent, 2000.0, 1L, false);

        assertEquals(-13, outcome.masteryExp);
        assertEquals(0, outcome.battleExp);
        assertEquals(0, outcome.streakExp);
        assertEquals(-13, outcome.expChange);
        assertEquals(0, outcome.effectiveStreak());
        assertFalse(outcome.effectiveWin());
        assertEquals(187, user.getExp());
        assertEquals(2, user.getRank());
        assertEquals(4, user.getTotalWins());
        assertEquals(7, user.getTotalGames());
        assertEquals(0, user.getCurrentWinStreak());
        verify(userMapper).updateById(user);
    }

    private User buildUser(Long id, int rank, int exp, int totalWins, int totalGames) {
        User user = new User();
        user.setId(id);
        user.setRank(rank);
        user.setExp(exp);
        user.setTotalWins(totalWins);
        user.setTotalGames(totalGames);
        user.setCurrentWinStreak(0);
        user.setNickname("tester");
        return user;
    }

    private GameSettlementService.PlayerResult buildResult(Long userId, int score, int setMaxScore, int correctCount, int totalCount) {
        GameSettlementService.PlayerResult result = new GameSettlementService.PlayerResult();
        result.userId = userId;
        result.score = score;
        result.setMaxScore = setMaxScore;
        result.correctCount = correctCount;
        result.totalCount = totalCount;
        return result;
    }
}
