package com.robword.entity;

import com.baomidou.mybatisplus.annotation.FieldFill;
import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Data;

import java.time.LocalDateTime;

@Data
@TableName("sentence_cloze_item")
public class SentenceClozeItem {

    @TableId(type = IdType.AUTO)
    private Long id;

    /** 该挖空练习归属用户 */
    private Long userId;

    /** 用户名快照 */
    private String userName;

    /** 主挖空单词 */
    private String word;

    /** 请求单词 JSON */
    private String wordsJson;

    /** 需要挖空的单词 JSON */
    private String blankWordsJson;

    /** 生成的英文句子 */
    private String sentence;

    /** 最佳例句记录 ID */
    private Long bestSentenceId;

    /** 最佳例句 TTS 音频地址 */
    private String sentenceAudioUrl;

    /** 句子中文翻译 */
    private String translationZh;

    /** 中文解释 */
    private String explanationZh;

    /** 将目标单词替换为空位后的句子 */
    private String clozeSentence;

    /** 模型配置 ID */
    private String providerId;

    /** 模型配置名称 */
    private String providerLabel;

    /** 模型名称 */
    private String model;

    /** 内容来源 */
    private String source;

    /** Python 错题事件 ID JSON */
    private String sourceEventIdsJson;

    /** 后端答题明细 ID JSON */
    private String sourceAnswerDetailIdsJson;

    /** 对局/答题记录 ID JSON */
    private String sourceRecordIdsJson;

    /** 单词 ID JSON */
    private String sourceWordIdsJson;

    /** 外部生成请求幂等键 */
    private String generationKey;

    @TableField(fill = FieldFill.INSERT)
    private LocalDateTime createTime;

    @TableField(fill = FieldFill.INSERT_UPDATE)
    private LocalDateTime updateTime;

    @TableField(exist = false)
    private Boolean latestAnswerCorrect;

    @TableField(exist = false)
    private LocalDateTime latestAnswerTime;

    @TableField(exist = false)
    private LocalDateTime nextReviewTime;
}
