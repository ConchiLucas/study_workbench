package com.robword.dto;

import lombok.Data;

import java.time.LocalDateTime;

@Data
public class WrongWordSummary {
    private Long wordId;
    private String wordContent;
    private String correctMeaning;
    private Integer wrongCount;
    private LocalDateTime lastWrongTime;
}
