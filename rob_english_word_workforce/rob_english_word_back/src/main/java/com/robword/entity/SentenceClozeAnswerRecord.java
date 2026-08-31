package com.robword.entity;

import com.baomidou.mybatisplus.annotation.FieldFill;
import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Data;

import java.time.LocalDateTime;

@Data
@TableName("sentence_cloze_answer_record")
public class SentenceClozeAnswerRecord {

    @TableId(type = IdType.AUTO)
    private Long id;

    /** 答题用户ID */
    private Long userId;

    /** 答题用户名快照 */
    private String userName;

    /** 挖空题内容ID */
    private Long clozeItemId;

    /** 用户原始答案文本 */
    private String answerText;

    /** 用户答案JSON */
    private String answersJson;

    /** 期望答案JSON */
    private String expectedWordsJson;

    /** 客户端提交幂等键 */
    private String submissionKey;

    /** 实际答题入口：review或solo */
    private String practiceContext;

    /** 提交动作：answer或reveal */
    private String actionType;

    /** 错误空位下标JSON */
    private String wrongBlankIndexesJson;

    /** 是否答对 */
    private Boolean isCorrect;

    /** 该用户对该题第几次作答 */
    private Integer attemptNo;

    /** 答题耗时毫秒 */
    private Long costMs;

    @TableField(fill = FieldFill.INSERT)
    private LocalDateTime createTime;

    @TableField(fill = FieldFill.INSERT_UPDATE)
    private LocalDateTime updateTime;
}
