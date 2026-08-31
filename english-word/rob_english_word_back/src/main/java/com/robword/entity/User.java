package com.robword.entity;

import com.fasterxml.jackson.annotation.JsonIgnore;
import com.baomidou.mybatisplus.annotation.*;
import lombok.Data;
import java.time.LocalDateTime;

@Data
@TableName("users")
public class User {

    /** id */
    @TableId(type = IdType.AUTO)
    private Long id;

    /** 用户名 */
    private String username;

    /** 密码 */
    private String password;

    /** 昵称 */
    private String nickname;

    /** 头像 */
    private String avatar;

    /** 等级 */
    private Integer rank;

    /** 经验值 */
    private Integer exp;

    /** 总胜场数 */
    private Integer totalWins;

    /** 总对战数 */
    private Integer totalGames;

    /** 当前有效连胜数 */
    @TableField("current_win_streak")
    private Integer currentWinStreak;

    /** 单人训练等级 */
    @TableField("training_rank")
    private Integer trainingRank;

    /** 单人训练经验值 */
    @TableField("training_exp")
    private Integer trainingExp;

    /** 单人训练胜场数 */
    @TableField("training_total_wins")
    private Integer trainingTotalWins;

    /** 单人训练总场次 */
    @TableField("training_total_games")
    private Integer trainingTotalGames;

    /** 挖空练习单独训练难度分组 */
    @TableField("solo_difficulty_group")
    private String soloDifficultyGroup;

    /** 挖空练习单独训练难度 */
    @TableField("solo_difficulty_level")
    private String soloDifficultyLevel;

    /** 创建时间 */
    @TableField(fill = FieldFill.INSERT)
    @JsonIgnore
    private LocalDateTime createTime;

    /** 更新时间 */
    @TableField(fill = FieldFill.INSERT_UPDATE)
    @JsonIgnore
    private LocalDateTime updateTime;
}
