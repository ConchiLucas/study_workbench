package com.robword.dto;

import lombok.Data;

@Data
public class UpdateSoloDifficultyRequest {
    private String difficultyGroup;
    private String difficultyLevel;
}
