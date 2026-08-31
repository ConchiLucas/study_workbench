package com.robword.util;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

class PKAlgorithmUtilTest {

    @Test
    void shouldCalculateExpectedSetMaxScoreFromDifficultyConfig() {
        PKAlgorithmUtil.DifficultyConfig config = new PKAlgorithmUtil.DifficultyConfig();
        config.probabilities[3] = 0.5;
        config.probabilities[4] = 0.5;

        double expectedSetMaxScore = PKAlgorithmUtil.calculateExpectedSetMaxScore(config, 5);

        assertEquals(2000.0, expectedSetMaxScore, 0.0001);
    }

    @Test
    void shouldGiveMaxMasteryBonusForHighChallengePerfectCompletion() {
        int masteryExp = PKAlgorithmUtil.calculateMasteryExp(1.0, 1.20);

        assertEquals(26, masteryExp);
    }

    @Test
    void shouldGiveMaxMasteryPenaltyForEasySetCompleteFailure() {
        int masteryExp = PKAlgorithmUtil.calculateMasteryExp(0.0, 0.80);

        assertEquals(-26, masteryExp);
    }

    @Test
    void shouldCompressBattleExpWhenBothSidesPlayPoorly() {
        int baseBattleExp = PKAlgorithmUtil.calculateBattleExp(false, true, 0.10);
        PKAlgorithmUtil.BattleAdjustment adjustment =
                PKAlgorithmUtil.adjustBattleExpForQuality(baseBattleExp, 0.55, 0.48);

        assertEquals(2, adjustment.adjustedBattleExp());
        assertEquals(false, adjustment.extremeLowQuality());
    }

    @Test
    void shouldZeroBattleExpForExtremeLowQualityMatch() {
        int baseBattleExp = PKAlgorithmUtil.calculateBattleExp(false, true, 0.24);
        PKAlgorithmUtil.BattleAdjustment adjustment =
                PKAlgorithmUtil.adjustBattleExpForQuality(baseBattleExp, 0.45, 0.42);

        assertEquals(0, adjustment.adjustedBattleExp());
        assertTrue(adjustment.extremeLowQuality());
    }

    @Test
    void shouldCapStreakRewardForNonEliteWin() {
        int streakExp = PKAlgorithmUtil.calculateStreakExp(5, 0.78, 1.00, false);

        assertEquals(4, streakExp);
    }

    @Test
    void shouldRemoveStreakRewardForExtremeLowQualityWin() {
        int streakExp = PKAlgorithmUtil.calculateStreakExp(3, 0.45, 0.90, true);

        assertEquals(0, streakExp);
    }

    @Test
    void shouldApplyHighestLoserProtectionFloor() {
        int protectedExp = PKAlgorithmUtil.applyLoserProtection(-3, true, 0.82, 1.12);

        assertEquals(8, protectedExp);
    }

    @Test
    void shouldCapExtremeLowQualityWinnerFinalExp() {
        int cappedExp = PKAlgorithmUtil.applyWinnerCap(12, true, true);

        assertEquals(4, cappedExp);
    }
}
