package com.robword.util;

import java.math.BigDecimal;
import java.math.RoundingMode;

/**
 * 单词 PK 对战等级段位升级、难度浮动计算引擎工具类
 *
 * 十档难度分级（T1-T10），每档对应词库难度 100 分一档。
 * 概率分布基于实际词库存量校准，采用线性插值实现段位内平滑过渡。
 */
public class PKAlgorithmUtil {

    private static final int DEFAULT_PK_WORD_COUNT = 5;
    private static final int[] TIER_MIDPOINTS = {50, 150, 250, 350, 450, 550, 650, 750, 850, 950};

    /**
     * 游戏十大难度分级定义 (对应题库 1-1000 难度，每 100 分一档)
     *
     * 各 Tier 实际词库量参考（2026-03-30 更新）：
     * T1(1-100):730, T2(101-200):2490, T3(201-300):3323, T4(301-400):1842,
     * T5(401-500):3034, T6(501-600):4928, T7(601-700):3757, T8(701-800):2438,
     * T9(801-900):1147, T10(901-1000):193
     */
    public enum WordDifficultyTier {
        T1(1, 100),
        T2(101, 200),
        T3(201, 300),
        T4(301, 400),
        T5(401, 500),
        T6(501, 600),
        T7(601, 700),
        T8(701, 800),
        T9(801, 900),
        T10(901, 1000);

        public final int min;
        public final int max;

        WordDifficultyTier(int min, int max) {
            this.min = min;
            this.max = max;
        }
    }

    /** T10 词池仅 157 词，全局概率硬顶 8%，防止高段位重复 */
    private static final double T10_CAP = 0.08;

    /** 难度 Tier 总数 */
    private static final int TIER_COUNT = 10;

    /**
     * 枚举：各大段位配置参数
     *
     * baseProbabilities 数组长度为 10，对应 T1-T10 的基准概率（百分比，合计 100）。
     * 概率分布设计为钟形曲线，随段位升高向右平移，同时考虑各 Tier 词库量：
     * - 峰值尽量落在词池充裕的 Tier（T5-T7 合计 10847 词 = 49%）
     * - T1(714 词) 和 T10(157 词) 始终控制在安全范围
     */
    public enum RankTier {
        //                                                T1   T2   T3   T4   T5   T6   T7   T8   T9   T10
        BRONZE(    "倔强青铜", 1,  10,  100, new double[]{12,  28,  30,  15,  10,   5,   0,   0,   0,   0}),
        SILVER(    "秩序白银", 11, 20,  150, new double[]{ 5,  15,  28,  22,  18,   8,   4,   0,   0,   0}),
        GOLD(      "荣耀黄金", 21, 30,  200, new double[]{ 2,   5,  15,  18,  25,  22,   8,   5,   0,   0}),
        PLATINUM(  "尊贵铂金", 31, 40,  300, new double[]{ 0,   2,   8,  12,  18,  28,  20,   8,   4,   0}),
        DIAMOND(   "永恒钻石", 41, 50,  400, new double[]{ 0,   0,   3,   8,  12,  25,  28,  15,   7,   2}),
        MASTER(    "至尊星耀", 51, 60,  600, new double[]{ 0,   0,   0,   3,   8,  18,  30,  25,  12,   4}),
        CHALLENGER("最强王者", 61, 999, 9999, new double[]{ 0,   0,   0,   0,   5,  12,  28,  28,  20,   7});

        public final String name;
        public final int minLevel;
        public final int maxLevel;
        public final int expPerLevel;
        public final double[] baseProbabilities; // T1-T10 基准概率（百分比）

        RankTier(String name, int minLevel, int maxLevel, int expPerLevel, double[] baseProbs) {
            this.name = name;
            this.minLevel = minLevel;
            this.maxLevel = maxLevel;
            this.expPerLevel = expPerLevel;
            this.baseProbabilities = baseProbs;
        }

        public static RankTier getTierByLevel(int level) {
            for (RankTier tier : values()) {
                if (level >= tier.minLevel && level <= tier.maxLevel) {
                    return tier;
                }
            }
            return CHALLENGER;
        }

        /**
         * 获取当前段位的下一个段位（用于线性插值）
         * 王者段位返回自身（不再向上偏移）
         */
        public RankTier getNextTier() {
            RankTier[] tiers = values();
            int idx = this.ordinal();
            if (idx + 1 < tiers.length) {
                return tiers[idx + 1];
            }
            return this; // 王者段位锁定自身
        }
    }

    // ==========================================
    // 1. 经验结算模块
    // ==========================================

    /**
     * 计算玩家 PK 胜利后获得的经验值
     * @param myLevel 胜利玩家目前等级
     * @param opponentLevel 对手等级
     * @return 实际增加的经验数值（有保底）
     */
    public static int calculateWinExp(int myLevel, int opponentLevel) {
        int baseWin = 20;
        int kFactor = 2;
        int deltaL = opponentLevel - myLevel;

        int winScore = baseWin + (deltaL * kFactor);
        return Math.max(5, winScore);
    }

    /**
     * 计算玩家 PK 失败后扣除的经验值
     * @param myLevel 失败玩家等级
     * @param opponentLevel 对手等级
     * @return 实际需要扣除的经验数值（绝对值为扣分量）
     */
    public static int calculateLoseExp(int myLevel, int opponentLevel) {
        int baseLose = -10;
        int kFactor = 2;
        int deltaL = opponentLevel - myLevel;

        int loseScore = baseLose + (deltaL * kFactor);
        return Math.min(0, loseScore);
    }

    /**
     * 根据玩家目前总积累的所有经验值（Total Exp），反向推算出真实对应的最新等级（Level）
     */
    public static int calculateLevelFromTotalExp(int totalExp) {
        if (totalExp <= 0) return 1;

        int currentLevel = 1;
        int remainingExp = totalExp;

        for (RankTier tier : RankTier.values()) {
            int levelsInThisTier = tier.maxLevel - tier.minLevel + 1;
            int totalMaxExpInTier = levelsInThisTier * tier.expPerLevel;

            if (remainingExp >= totalMaxExpInTier && tier != RankTier.CHALLENGER) {
                remainingExp -= totalMaxExpInTier;
                currentLevel += levelsInThisTier;
            } else {
                int levelsGained = remainingExp / tier.expPerLevel;
                currentLevel += levelsGained;
                break;
            }
        }
        return currentLevel;
    }

    // ==========================================
    // 2. 动态单词难度概率分布抽取引擎（T1-T10 线性插值版）
    // ==========================================

    /**
     * T1-T10 难度抽取概率配置
     */
    public static class DifficultyConfig {
        /** T1-T10 的概率值，下标 0=T1 ... 9=T10，每个值为 0.0~1.0 之间的比例 */
        public double[] probabilities = new double[TIER_COUNT];

        /** 获取指定 Tier 的概率（0-indexed） */
        public double get(int tierIndex) {
            return probabilities[tierIndex];
        }
    }

    public record BattleAdjustment(int adjustedBattleExp, boolean extremeLowQuality) {
    }

    /**
     * 根据对战双方等级，生成动态偏移后的 T1-T10 难度概率分布。
     *
     * 算法：线性插值
     * 1. 计算双方平均等级 → 确定基准段位
     * 2. 计算段位内进度 progress（0.0 ~ 1.0）
     * 3. 对每个 Tier：actualProb[i] = base[i] + (next[i] - base[i]) × progress
     * 4. 归一化处理 + T10 安全帽检查
     */
    public static DifficultyConfig generatePKDifficulty(int levelA, int levelB) {
        int avgLevel = Math.max(1, (levelA + levelB) / 2);

        RankTier currentTier = RankTier.getTierByLevel(avgLevel);
        RankTier nextTier = currentTier.getNextTier();

        // 计算段位内进度（0.0 = 刚进入该段位, 1.0 = 即将晋升）
        int tierRange = currentTier.maxLevel - currentTier.minLevel + 1;
        int deltaLv = avgLevel - currentTier.minLevel;
        double progress;

        if (currentTier == RankTier.CHALLENGER) {
            // 王者段位不偏移，锁定基准值
            progress = 0.0;
        } else {
            progress = Math.min(1.0, (double) deltaLv / tierRange);
        }

        // 线性插值计算每个 Tier 的实际概率
        double[] probs = new double[TIER_COUNT];
        for (int i = 0; i < TIER_COUNT; i++) {
            double base = currentTier.baseProbabilities[i];
            double next = nextTier.baseProbabilities[i];
            probs[i] = base + (next - base) * progress;
            probs[i] = Math.max(0.0, probs[i]); // 不允许负数
        }

        // T10 安全帽：概率不得超过 T10_CAP（8%）
        if (probs[9] > T10_CAP * 100) {
            double excess = probs[9] - T10_CAP * 100;
            probs[9] = T10_CAP * 100;
            // 多余的概率均匀分配给 T7 和 T8（词池最充裕的高阶 Tier）
            probs[6] += excess * 0.5;
            probs[7] += excess * 0.5;
        }

        // 归一化：确保概率总和精确为 1.0
        double sum = 0;
        for (double p : probs) sum += p;

        DifficultyConfig config = new DifficultyConfig();
        if (sum > 0) {
            for (int i = 0; i < TIER_COUNT; i++) {
                config.probabilities[i] = round(probs[i] / sum, 4);
            }
        }

        // 最终安全检查：归一化后再次确认总和为 1.0
        double finalSum = 0;
        for (double p : config.probabilities) finalSum += p;
        if (Math.abs(finalSum - 1.0) > 0.001) {
            // 修正浮点误差，差值加到最大概率的 Tier 上
            int maxIdx = 0;
            for (int i = 1; i < TIER_COUNT; i++) {
                if (config.probabilities[i] > config.probabilities[maxIdx]) maxIdx = i;
            }
            config.probabilities[maxIdx] += (1.0 - finalSum);
        }

        return config;
    }

    public static int getTierMidpoint(int tierIndex) {
        if (tierIndex < 0 || tierIndex >= TIER_MIDPOINTS.length) {
            throw new IllegalArgumentException("Invalid tier index: " + tierIndex);
        }
        return TIER_MIDPOINTS[tierIndex];
    }

    public static WordDifficultyTier getTierByDifficulty(int difficulty) {
        int safeDifficulty = Math.max(1, Math.min(1000, difficulty));
        int tierIndex = Math.min((safeDifficulty - 1) / 100, TIER_COUNT - 1);
        return WordDifficultyTier.values()[tierIndex];
    }

    public static double calculateExpectedWordScore(DifficultyConfig config) {
        if (config == null) {
            return 0.0;
        }

        double expectedWordScore = 0.0;
        for (int i = 0; i < TIER_COUNT; i++) {
            expectedWordScore += config.get(i) * getTierMidpoint(i);
        }
        return expectedWordScore;
    }

    public static double calculateExpectedSetMaxScore(DifficultyConfig config, int wordCount) {
        int safeWordCount = wordCount <= 0 ? DEFAULT_PK_WORD_COUNT : wordCount;
        return calculateExpectedWordScore(config) * safeWordCount;
    }

    public static double calculateCompletionRate(int actualScore, int setMaxScore) {
        if (setMaxScore <= 0) {
            return 0.0;
        }
        return (double) actualScore / setMaxScore;
    }

    public static double calculateChallengeIndex(int setMaxScore, double expectedSetMaxScore) {
        if (expectedSetMaxScore <= 0) {
            return 0.0;
        }
        return setMaxScore / expectedSetMaxScore;
    }

    public static double calculateGapRate(int myScore, int opponentScore, int mySetMaxScore, int opponentSetMaxScore) {
        int denominator = Math.max(Math.max(mySetMaxScore, opponentSetMaxScore), 1);
        return (double) Math.abs(myScore - opponentScore) / denominator;
    }

    public static int calculateMasteryExp(double completionRate, double challengeIndex) {
        int masteryBase;
        if (completionRate >= 0.90) {
            masteryBase = 18;
        } else if (completionRate >= 0.75) {
            masteryBase = 8;
        } else if (completionRate >= 0.55) {
            masteryBase = 0;
        } else if (completionRate >= 0.40) {
            masteryBase = -8;
        } else {
            masteryBase = -18;
        }

        int challengeBonus = 0;
        if (completionRate >= 0.90 && challengeIndex >= 1.15) {
            challengeBonus = 8;
        } else if (completionRate >= 0.80 && challengeIndex >= 1.10) {
            challengeBonus = 5;
        } else if (completionRate >= 0.70 && challengeIndex >= 1.05) {
            challengeBonus = 3;
        }

        int challengePenalty = 0;
        if (completionRate < 0.40 && challengeIndex <= 0.90) {
            challengePenalty = -8;
        } else if (completionRate < 0.50 && challengeIndex <= 0.95) {
            challengePenalty = -5;
        } else if (completionRate < 0.55 && challengeIndex < 1.00) {
            challengePenalty = -3;
        }

        return Math.max(-26, Math.min(26, masteryBase + challengeBonus + challengePenalty));
    }

    public static int calculateBattleExp(boolean isDraw, boolean isWinner, double gapRate) {
        if (isDraw) {
            return 0;
        }

        int battleExp;
        if (gapRate < 0.08) {
            battleExp = 2;
        } else if (gapRate < 0.18) {
            battleExp = 4;
        } else if (gapRate < 0.30) {
            battleExp = 6;
        } else {
            battleExp = 8;
        }

        return isWinner ? battleExp : -battleExp;
    }

    public static BattleAdjustment adjustBattleExpForQuality(int battleExp, double myCompletionRate, double opponentCompletionRate) {
        if (myCompletionRate < 0.50 && opponentCompletionRate < 0.50) {
            return new BattleAdjustment(0, true);
        }
        if (myCompletionRate < 0.60 && opponentCompletionRate < 0.60) {
            return new BattleAdjustment(battleExp / 2, false);
        }
        return new BattleAdjustment(battleExp, false);
    }

    public static boolean isEffectiveWin(boolean isWinner, double completionRate, int masteryExp, int adjustedBattleExp) {
        return isWinner && completionRate >= 0.75 && masteryExp + adjustedBattleExp > 0;
    }

    public static int calculateStreakExp(int effectiveStreak, double completionRate, double challengeIndex, boolean extremeLowQuality) {
        if (effectiveStreak <= 1 || extremeLowQuality) {
            return 0;
        }

        int streakExp;
        if (effectiveStreak == 2) {
            streakExp = 4;
        } else if (effectiveStreak == 3) {
            streakExp = 8;
        } else if (effectiveStreak == 4) {
            streakExp = 12;
        } else {
            streakExp = 16;
        }

        if (completionRate < 0.80) {
            streakExp = Math.min(streakExp, 8);
        }
        if (completionRate < 0.90 && challengeIndex < 1.05) {
            streakExp = Math.min(streakExp, 4);
        }

        return streakExp;
    }

    public static int applyLoserProtection(int finalExp, boolean isLoser, double completionRate, double challengeIndex) {
        if (!isLoser) {
            return finalExp;
        }

        int floor = Integer.MIN_VALUE;
        if (completionRate >= 0.80) {
            floor = Math.max(floor, 0);
        }
        if (completionRate >= 0.90) {
            floor = Math.max(floor, 4);
        }
        if (completionRate >= 0.80 && challengeIndex >= 1.10) {
            floor = Math.max(floor, 8);
        }

        return floor == Integer.MIN_VALUE ? finalExp : Math.max(finalExp, floor);
    }

    public static int applyWinnerCap(int finalExp, boolean isWinner, boolean extremeLowQuality) {
        if (isWinner && extremeLowQuality) {
            return Math.min(finalExp, 4);
        }
        return finalExp;
    }

    private static double round(double value, int places) {
        if (places < 0) throw new IllegalArgumentException();
        BigDecimal bd = new BigDecimal(Double.toString(value));
        bd = bd.setScale(places, RoundingMode.HALF_UP);
        return bd.doubleValue();
    }
}
