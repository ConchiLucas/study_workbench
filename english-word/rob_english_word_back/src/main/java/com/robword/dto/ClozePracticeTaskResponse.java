package com.robword.dto;

import lombok.Data;

import java.time.LocalDateTime;
import java.util.List;

@Data
public class ClozePracticeTaskResponse {

    private Long id;
    private String word;
    private String wordAudioUrl;
    private String sentence;
    private String sentenceAudioUrl;
    private String clozeSentence;
    private String translationZh;
    private Integer blankCount;
    private List<Integer> blankLengths;
    private Integer attemptCount;
    private Integer wrongCount;
    private String difficultyGroup;
    private String difficultyLevel;
    private String difficultyLabel;
    private String source;
    private String model;
    private Boolean latestAnswerCorrect;
    private LocalDateTime latestAnswerTime;
    private LocalDateTime nextReviewTime;
    private LocalDateTime createTime;
}
