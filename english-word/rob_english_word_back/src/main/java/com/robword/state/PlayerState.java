package com.robword.state;

/**
 * 玩家状态枚举
 * 状态转换规则：
 * IDLE → MATCHING → MATCHED → GRABBING → ANSWERING → FINISHED → IDLE
 */
public enum PlayerState {
    /** 在线空闲 */
    IDLE,
    /** 匹配排队中 */
    MATCHING,
    /** 已匹配，等待游戏开始 */
    MATCHED,
    /** 抢词阶段 */
    GRABBING,
    /** 答题阶段 */
    ANSWERING,
    /** 游戏结束 */
    FINISHED;

    /**
     * 检查从当前状态到目标状态的转换是否合法
     */
    public boolean canTransitionTo(PlayerState target) {
        return switch (this) {
            case IDLE -> target == MATCHING;
            case MATCHING -> target == MATCHED || target == IDLE;
            case MATCHED -> target == GRABBING || target == IDLE;
            case GRABBING -> target == ANSWERING || target == IDLE;
            case ANSWERING -> target == FINISHED || target == IDLE;
            case FINISHED -> target == IDLE;
        };
    }
}
