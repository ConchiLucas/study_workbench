package com.robword.entity;

import com.baomidou.mybatisplus.annotation.*;
import lombok.Data;
import java.time.LocalDateTime;

@Data
@TableName("game_record")
public class GameRecord {

    /** 记录ID */
    @TableId(type = IdType.AUTO)
    private Long id;

    /** 房间ID */
    private Long roomId;

    /** 记录模式：match-正式匹配 solo_training-单人训练 */
    private String mode;

    // ==================== 玩家1信息 ====================
    /** 玩家1ID */
    private Long player1Id;

    /** 玩家1名称 */
    private String player1Name;

    /** 玩家1总得分 */
    private Integer player1Score;

    /** 玩家1答对题数 */
    private Integer player1CorrectCount;

    /** 玩家1总答题数 */
    private Integer player1TotalCount;

    // ==================== 玩家2信息 ====================
    /** 玩家2ID */
    private Long player2Id;

    /** 玩家2名称 */
    private String player2Name;

    /** 玩家2总得分 */
    private Integer player2Score;

    /** 玩家2答对题数 */
    private Integer player2CorrectCount;

    /** 玩家2总答题数 */
    private Integer player2TotalCount;

    // ==================== 胜负结果 ====================
    /** 获胜者ID（NULL表示平局） */
    private Long winnerId;

    /** 是否平局：0-否 1-是 */
    private Integer isDraw;

    // ==================== 时间记录 ====================
    /** 比赛开始时间 */
    private LocalDateTime startTime;

    /** 比赛结束时间 */
    private LocalDateTime endTime;

    /** 比赛持续时间（秒） */
    private Integer durationSeconds;

    // ==================== 正式匹配难度 ====================
    /** 正式匹配选择难度父级 */
    private String matchDifficultyGroup;

    /** 正式匹配选择难度 */
    private String matchDifficultyLevel;

    /** 正式匹配难度展示名称 */
    private String matchDifficultyLabel;

    // ==================== 单人训练信息 ====================
    /** 本局训练经验变化 */
    private Integer trainingExpChange;

    /** 本局结束后的训练等级 */
    private Integer trainingRankAfter;

    /** 训练选择难度父级 */
    private String trainingDifficultyGroup;

    /** 训练选择难度 */
    private String trainingDifficultyLevel;

    /** 机器人档位 */
    private String robotTier;

    /** 机器人资质 */
    private Integer robotAptitude;

    /** 机器人成长 */
    private Double robotGrowth;

    /** 机器人完整面板快照 */
    private String robotProfileJson;

    /** 创建时间 */
    @TableField(fill = FieldFill.INSERT)
    private LocalDateTime createTime;

    /** 更新时间 */
    @TableField(fill = FieldFill.INSERT_UPDATE)
    private LocalDateTime updateTime;
}
