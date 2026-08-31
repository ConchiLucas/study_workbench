package com.robword.dto;

import lombok.Data;

@Data
public class ClozePracticeSentenceCandidate {

    private Long wordCleanId;

    private Long bestSentenceId;

    private String word;

    private String meaning;

    private String sentence;

    private String sentenceTranslation;

    private String clozeSentence;

    private String clozeAnswer;

    private String modelName;

    private String ttsObjectUrl;
}
