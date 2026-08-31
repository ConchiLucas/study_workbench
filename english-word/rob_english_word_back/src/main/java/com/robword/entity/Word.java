package com.robword.entity;

import com.fasterxml.jackson.annotation.JsonIgnore;
import com.baomidou.mybatisplus.annotation.*;
import lombok.Data;
import java.time.LocalDateTime;

@Data
@TableName("word")
public class Word {

    /** 单词ID */
    @TableId(type = IdType.AUTO)
    private Long id;

    /** 词库ID */
    private Long libraryId;

    /** 单词内容 */
    private String word;

    /** 中文释义 */
    private String meaning;

    /** 美式音标 */
    private String pronunciationUs;

    /** 英式音标 */
    private String pronunciationUk;

    /** 词频 */
    private Integer frequency;

    /** 难度：1-简单 2-中等 3-困难 */
    private Integer difficulty;

    /** 状态：0-禁用 1-启用 */
    private Integer status;

    /** 短语示例 */
    private String phrase;

    /** 短语翻译 */
    private String phraseTranslation;

    /** 例句 */
    private String sentence;

    /** 例句翻译 */
    private String sentenceTranslation;

    /** 创建时间 */
    @TableField(fill = FieldFill.INSERT)
    @JsonIgnore
    private LocalDateTime createTime;

    /** 更新时间 */
    @TableField(fill = FieldFill.INSERT_UPDATE)
    @JsonIgnore
    private LocalDateTime updateTime;
}
