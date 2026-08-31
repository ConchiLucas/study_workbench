package com.robword.dto;

import lombok.Data;

import java.util.List;

@Data
public class ClozeWrongSentencePageResponse {

    private List<ClozeWrongSentenceItem> items;
    private Long total;
    private Integer current;
    private Integer pages;
    private Summary summary;

    @Data
    public static class Summary {
        private Long activeCount;
        private Long dueCount;
        private Long stage1Count;
        private Long stage2Count;
        private Long completedCount;
    }
}
