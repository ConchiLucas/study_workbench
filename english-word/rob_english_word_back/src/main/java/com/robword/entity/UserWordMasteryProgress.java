package com.robword.entity;

import com.baomidou.mybatisplus.annotation.FieldFill;
import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Data;

import java.time.LocalDateTime;

@Data
@TableName("user_word_mastery_progress")
public class UserWordMasteryProgress {

    @TableId(type = IdType.AUTO)
    private Long id;

    private Long userId;

    private Long wordId;

    private String wordContent;

    private String correctMeaning;

    private String status;

    private Integer stage;

    private Integer correctCount;

    private LocalDateTime firstCorrectTime;

    private LocalDateTime day1CorrectTime;

    private LocalDateTime day7CorrectTime;

    private LocalDateTime nextReviewTime;

    private LocalDateTime lastCorrectTime;

    private LocalDateTime masteredTime;

    private Long lastAnswerDetailId;

    @TableField(fill = FieldFill.INSERT)
    private LocalDateTime createTime;

    @TableField(fill = FieldFill.INSERT_UPDATE)
    private LocalDateTime updateTime;
}
