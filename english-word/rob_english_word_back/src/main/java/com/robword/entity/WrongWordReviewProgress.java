package com.robword.entity;

import com.baomidou.mybatisplus.annotation.FieldFill;
import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Data;

import java.time.LocalDateTime;

@Data
@TableName("wrong_word_review_progress")
public class WrongWordReviewProgress {

    @TableId(type = IdType.AUTO)
    private Long id;

    private Long userId;
    private Long wordId;
    private String word;
    private String normalizedWord;
    private String status;
    private Integer reviewStage;
    private LocalDateTime nextReviewTime;
    private Long activeClozeItemId;
    private Integer activeBlankIndex;
    private Integer wrongCount;
    private LocalDateTime firstWrongTime;
    private LocalDateTime lastWrongTime;
    private Long lastAnswerRecordId;
    private LocalDateTime completedTime;

    @TableField(fill = FieldFill.INSERT)
    private LocalDateTime createTime;

    @TableField(fill = FieldFill.INSERT_UPDATE)
    private LocalDateTime updateTime;
}
