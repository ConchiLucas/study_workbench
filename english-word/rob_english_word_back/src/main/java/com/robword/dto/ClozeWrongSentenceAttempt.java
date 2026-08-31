package com.robword.dto;

import com.fasterxml.jackson.annotation.JsonIgnore;
import lombok.Data;

import java.time.LocalDateTime;

@Data
public class ClozeWrongSentenceAttempt {

    private Long recordId;
    private Boolean correct;
    private Long costMs;
    private String practiceContext;
    private String actionType;
    private LocalDateTime answeredAt;

    @JsonIgnore
    private String wrongBlankIndexesJson;
}
