package com.robword.dto;

import lombok.Data;

import java.time.LocalDateTime;

@Data
public class WrongWordQueueEvent {
    private String eventKey;
    private String progressKey;
    private String word;
    private LocalDateTime answeredAt;
    private String entry;
    private String mode;
    private String difficultyGroup;
    private String difficultyLevel;
    private String difficultyLabel;
    private Integer wordDifficulty;
    private Long costMs;
    private String correctAnswer;
    private String exampleSentence;
    private String exampleSource;
    private String sourceType;
    private Integer occurrenceCount;
    private String reviewStatus;
    private Integer reviewStage;
    private LocalDateTime nextReviewTime;
}
