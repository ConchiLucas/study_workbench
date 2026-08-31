package com.robword.entity;

import com.fasterxml.jackson.annotation.JsonIgnore;
import com.baomidou.mybatisplus.annotation.*;
import lombok.Data;
import java.time.LocalDateTime;

@Data
@TableName("word_library")
public class WordLibrary {

    /** 词库ID */
    @TableId(type = IdType.AUTO)
    private Long id;

    /** 词库名称 */
    private String libraryName;

    /** 词库名称中文说明 */
    private String libraryMeaning;

    /** 状态：0-禁用 1-启用 */
    private Integer status;

    /** 单词数量 */
    private Integer wordCount;

    /** 创建者ID */
    private Long createdBy;

    /** 创建时间 */
    @TableField(fill = FieldFill.INSERT)
    @JsonIgnore
    private LocalDateTime createTime;

    /** 更新时间 */
    @TableField(fill = FieldFill.INSERT_UPDATE)
    @JsonIgnore
    private LocalDateTime updateTime;
}
