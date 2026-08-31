package com.robword.dto;

import lombok.Data;

import java.util.List;

@Data
public class ClozePracticeAnswerResponse {

    private Long recordId;
    private Long clozeItemId;
    private Boolean correct;
    private String answerText;
    private List<String> answers;
    private List<String> expectedWords;
    private Integer attemptNo;
    private String message;
}
