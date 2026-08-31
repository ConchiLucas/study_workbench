package com.robword.dto;

import lombok.Data;

import java.time.LocalDateTime;

@Data
public class MasteredWordSummary {
    private Long wordId;
    private String wordContent;
    private String correctMeaning;
    private String status;
    private Integer stage;
    private Integer correctCount;
    private LocalDateTime firstCorrectTime;
    private LocalDateTime day1CorrectTime;
    private LocalDateTime day7CorrectTime;
    private LocalDateTime nextReviewTime;
    private LocalDateTime lastCorrectTime;
    private LocalDateTime masteredTime;
}
