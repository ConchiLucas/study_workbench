package com.robword.dto;

import lombok.Data;

import java.time.LocalDateTime;

@Data
public class ClozePracticeHistoryItem {

    private Long id;
    private Long clozeItemId;
    private String clozeSentence;
    private String translationZh;
    private String answerText;
    private String expectedWordsJson;
    private Boolean isCorrect;
    private Integer attemptNo;
    private Long costMs;
    private LocalDateTime createTime;
}
