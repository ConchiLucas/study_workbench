package com.robword.entity;

import com.baomidou.mybatisplus.annotation.*;
import lombok.Data;
import java.time.LocalDateTime;

@Data
@TableName("game_answer_detail")
public class GameAnswerDetail {

    /** 详情ID */
    @TableId(type = IdType.AUTO)
    private Long id;

    /** 关联的game_record.id */
    private Long recordId;

    /** 答题用户ID */
    private Long userId;

    /** 答题用户名称 */
    private String userName;

    // ==================== 轮次信息 ====================
    /** 轮次序号（1-8） */
    private Integer roundNo;

    // ==================== 单词信息 ====================
    /** 单词ID */
    private Long wordId;

    /** 单词内容 */
    private String wordContent;

    /** 单词难度 */
    private Integer wordDifficulty;

    // ==================== 四个选项内容 ====================
    /** 选项1内容 */
    @TableField("option_1")
    private String option1;

    /** 选项2内容 */
    @TableField("option_2")
    private String option2;

    /** 选项3内容 */
    @TableField("option_3")
    private String option3;

    /** 选项4内容 */
    @TableField("option_4")
    private String option4;

    // ==================== 答案索引 ====================
    /** 正确答案序号（1-4） */
    @TableField("correct_answer_index")
    private Integer correctAnswerIndex;

    /** 玩家选择的答案序号（1-4，未回答为null） */
    @TableField("selected_answer_index")
    private Integer selectedAnswerIndex;

    // ==================== 答题结果 ====================
    /** 是否答对：0-错 1-对 */
    @TableField("is_correct")
    private Integer isCorrect;

    /** 本题记分（答对得难度分，答错得0） */
    private Integer score;

    // ==================== 答题用时 ====================
    /** 答题用时（毫秒） */
    @TableField("answer_time_ms")
    private Integer answerTimeMs;

    /** 创建时间 */
    @TableField(fill = FieldFill.INSERT)
    private LocalDateTime createTime;
}
