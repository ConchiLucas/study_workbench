package com.robword.dto;

import lombok.Data;

import java.util.List;

@Data
public class ClozePracticeAnswerRequest {

    private Long clozeItemId;
    private String answerText;
    private List<String> answers;
    private Long costMs;
    private String submissionKey;
    private String practiceContext;
    private String actionType;
}
