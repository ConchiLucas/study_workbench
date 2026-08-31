package com.robword.dto;

import lombok.Data;

import java.time.LocalDateTime;

@Data
public class WrongWordDetail {
    private Long id;
    private String wordContent;
    private String correctMeaning;
    private LocalDateTime createTime;
}
