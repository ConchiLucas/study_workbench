package com.robword.dto;

import lombok.Data;

import java.util.List;

@Data
public class ClozeWrongSentenceDetail {

    private ClozeWrongSentenceItem item;
    private List<BlankReview> blanks;
    private List<ClozeWrongSentenceAttempt> attempts;
    private List<ReviewStageStep> reviewStages;

    @Data
    public static class BlankReview {
        private Integer index;
        private String word;
        private Boolean lastCorrect;
        private String meaning;
        private Integer wordReviewStage;
        private String wordReviewStatus;
    }

    @Data
    public static class ReviewStageStep {
        private Integer stage;
        private String label;
        private String state;
    }
}
