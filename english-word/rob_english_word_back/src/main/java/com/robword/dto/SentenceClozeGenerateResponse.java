package com.robword.dto;

import lombok.Data;

import java.time.LocalDateTime;
import java.util.List;

@Data
public class SentenceClozeGenerateResponse {

    private Long id;
    private Long userId;
    private String userName;
    private String word;
    private List<String> words;
    private String sentence;
    private String sentenceAudioUrl;
    private String translationZh;
    private String explanationZh;
    private String clozeSentence;
    private List<String> blankWords;
    private List<Long> sourceEventIds;
    private List<Long> sourceAnswerDetailIds;
    private List<Long> sourceRecordIds;
    private List<Long> sourceWordIds;
    private String providerId;
    private String providerLabel;
    private String model;
    private LocalDateTime createTime;
}
