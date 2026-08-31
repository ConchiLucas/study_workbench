package com.robword.dto;

import com.fasterxml.jackson.annotation.JsonIgnore;
import lombok.Data;

import java.time.LocalDateTime;
import java.util.List;

@Data
public class ClozeWrongSentenceItem {

    private Long progressId;
    private Long clozeItemId;
    private String clozeSentence;
    private String sentence;
    private String translationZh;
    private List<String> targetWords;
    private List<Integer> wrongBlankIndexes;
    private Integer wrongBlankCount;
    private String practiceContext;
    private String contentSource;
    private String difficultyLabel;
    private String status;
    private Integer reviewStage;
    private LocalDateTime nextReviewTime;
    private Integer wrongCount;
    private LocalDateTime firstWrongTime;
    private LocalDateTime lastWrongTime;
    private Long lastCostMs;

    @JsonIgnore
    private String targetWordsJson;

    @JsonIgnore
    private String wrongBlankIndexesJson;
}
