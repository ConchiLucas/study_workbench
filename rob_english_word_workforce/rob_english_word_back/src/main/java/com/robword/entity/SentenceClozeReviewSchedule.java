package com.robword.entity;

import com.baomidou.mybatisplus.annotation.FieldFill;
import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Data;

import java.time.LocalDateTime;

@Data
@TableName("sentence_cloze_review_schedule")
public class SentenceClozeReviewSchedule {

    @TableId(type = IdType.AUTO)
    private Long id;

    /** 用户ID */
    private Long userId;

    /** 挖空题内容ID */
    private Long clozeItemId;

    /** 进入复习后连续答对次数 */
    private Integer correctStreak;

    /** 复习阶段：0立即，1七天，2十五天，3已完成 */
    private Integer reviewStage;

    /** 复习状态：active或completed */
    private String status;

    /** 下次可复习时间 */
    private LocalDateTime nextReviewTime;

    /** 整句错误提交次数 */
    private Integer wrongCount;

    /** 首次答错时间 */
    private LocalDateTime firstWrongTime;

    /** 最近一次答题记录ID */
    private Long lastAnswerRecordId;

    /** 最近一次错误答题记录ID */
    private Long lastWrongAnswerRecordId;

    /** 最近一次答错时间 */
    private LocalDateTime lastWrongTime;

    /** 最近一次答对时间 */
    private LocalDateTime lastCorrectTime;

    /** 完成三阶段复习时间 */
    private LocalDateTime completedTime;

    @TableField(fill = FieldFill.INSERT)
    private LocalDateTime createTime;

    @TableField(fill = FieldFill.INSERT_UPDATE)
    private LocalDateTime updateTime;
}
