package com.robword.dto;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@NoArgsConstructor
@AllArgsConstructor
public class ClozePracticePreferenceResponse {
    private String soloDifficultyGroup;
    private String soloDifficultyLevel;
}
