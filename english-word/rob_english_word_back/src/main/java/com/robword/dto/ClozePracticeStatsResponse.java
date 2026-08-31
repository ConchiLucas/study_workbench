package com.robword.dto;

import lombok.Data;

@Data
public class ClozePracticeStatsResponse {

    private Long totalTasks;
    private Long completedTasks;
    private Long pendingTasks;
    private Long totalAnswers;
    private Long correctAnswers;
    private Long wrongAnswers;
    private Long activeWrongSentences;
    private Long dueReviewTasks;
    private Double accuracy;
}
