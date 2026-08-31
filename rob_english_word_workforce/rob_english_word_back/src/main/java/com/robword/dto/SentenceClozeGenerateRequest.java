package com.robword.dto;

import lombok.Data;

import java.util.List;

@Data
public class SentenceClozeGenerateRequest {

    /** 该挖空练习归属用户 */
    private Long userId;

    /** 用户名快照，便于后续展示和排查 */
    private String userName;

    /** 主挖空单词，外部调用只传一个单词时使用 */
    private String word;

    /** 兼容批量传词；后续挖空句会按这些词生成空位 */
    private List<String> words;

    /** Python 错题事件 ID 列表 */
    private List<Long> sourceEventIds;

    /** 后端答题明细 ID 列表 */
    private List<Long> sourceAnswerDetailIds;

    /** 对局/答题记录 ID 列表 */
    private List<Long> sourceRecordIds;

    /** 单词 ID 列表 */
    private List<Long> sourceWordIds;

    /** 外部生成请求幂等键 */
    private String generationKey;
}
