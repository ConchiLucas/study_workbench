package com.robword.dto;

import lombok.Data;

import java.time.LocalDateTime;
import java.util.List;

@Data
public class WrongWordDetailResponse {
    private Long wordId;
    private String wordContent;
    private String correctMeaning;
    private Integer wrongCount;
    private LocalDateTime lastWrongTime;
    private List<WrongWordDetail> details;
}
