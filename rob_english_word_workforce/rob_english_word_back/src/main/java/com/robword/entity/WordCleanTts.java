package com.robword.entity;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Data;

@Data
@TableName("word_clean_tts")
public class WordCleanTts {

    @TableId(type = IdType.AUTO)
    private Long id;

    @TableField("word_clean_id")
    private Long wordCleanId;

    private String word;

    private String status;

    @TableField("tts_object_url")
    private String ttsObjectUrl;
}
