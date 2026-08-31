package com.robword.dto;

import lombok.Data;

@Data
public class ClozePracticeDifficultyBatchRequest {

    private String difficultyGroup;

    private String difficultyLevel;

    private Integer limit;
}
