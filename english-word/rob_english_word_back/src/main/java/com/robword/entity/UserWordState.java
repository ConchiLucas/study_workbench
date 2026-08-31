package com.robword.entity;

import com.baomidou.mybatisplus.annotation.FieldFill;
import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Data;

import java.time.LocalDateTime;

@Data
@TableName("user_word_state")
public class UserWordState {

    @TableId(type = IdType.AUTO)
    private Long id;

    private Long userId;

    private Long wordId;

    @TableField(fill = FieldFill.INSERT)
    private LocalDateTime createTime;
}
