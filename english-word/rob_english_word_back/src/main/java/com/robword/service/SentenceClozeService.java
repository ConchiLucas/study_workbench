package com.robword.service;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.robword.dto.SentenceClozeGenerateRequest;
import com.robword.dto.SentenceClozeGenerateResponse;
import com.robword.entity.SentenceClozeItem;
import com.robword.mapper.SentenceClozeItemMapper;
import lombok.Data;
import lombok.RequiredArgsConstructor;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.dao.DuplicateKeyException;
import org.springframework.http.HttpStatusCode;
import org.springframework.stereotype.Service;
import org.springframework.web.client.RestClient;

import java.util.ArrayList;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Locale;
import java.util.Set;
import java.time.LocalDateTime;
import java.util.regex.Pattern;

@Service
@RequiredArgsConstructor
public class SentenceClozeService {

    private static final int MAX_WORD_COUNT = 12;
    private static final String BLANK_TEXT = "____";

    private final SentenceClozeItemMapper sentenceClozeItemMapper;
    private final ObjectMapper objectMapper;
    private final RestClient.Builder restClientBuilder;
    private final WrongWordReviewProgressService wrongWordReviewProgressService;

    @Value("${word-agent.base-url:http://127.0.0.1:8010}")
    private String wordAgentBaseUrl;

    public SentenceClozeGenerateResponse generateAndSave(SentenceClozeGenerateRequest request) {
        List<String> words = normalizeWords(request);
        String generationKey = normalizeGenerationKey(request.getGenerationKey());
        if (generationKey != null) {
            SentenceClozeItem existing = sentenceClozeItemMapper.selectByGenerationKey(generationKey);
            if (existing != null) {
                linkReviewProgress(existing, parseList(existing.getWordsJson(), String.class));
                return toStoredResponse(existing);
            }
        }
        List<Long> sourceEventIds = normalizeLongs(request.getSourceEventIds());
        List<Long> sourceAnswerDetailIds = normalizeLongs(request.getSourceAnswerDetailIds());
        List<Long> sourceRecordIds = normalizeLongs(request.getSourceRecordIds());
        List<Long> sourceWordIds = normalizeLongs(request.getSourceWordIds());
        WordAgentSentenceResponse agentResponse = callWordAgent(words);
        String sentenceAudioUrl = requireSentenceAudioUrl(agentResponse.getSentenceAudioUrl());
        String clozeSentence = buildClozeSentence(agentResponse.getSentence(), words);

        SentenceClozeItem item = new SentenceClozeItem();
        item.setUserId(request.getUserId());
        item.setUserName(normalizeUserName(request.getUserName()));
        item.setWord(words.get(0));
        item.setWordsJson(toJson(words));
        item.setBlankWordsJson(toJson(words));
        item.setSentence(agentResponse.getSentence());
        item.setSentenceAudioUrl(sentenceAudioUrl);
        item.setTranslationZh(agentResponse.getTranslationZh());
        item.setExplanationZh(agentResponse.getExplanationZh());
        item.setClozeSentence(clozeSentence);
        item.setProviderId(agentResponse.getProviderId());
        item.setProviderLabel(agentResponse.getProviderLabel());
        item.setModel(agentResponse.getModel());
        item.setSource("word-agent");
        item.setSourceEventIdsJson(toJson(sourceEventIds));
        item.setSourceAnswerDetailIdsJson(toJson(sourceAnswerDetailIds));
        item.setSourceRecordIdsJson(toJson(sourceRecordIds));
        item.setSourceWordIdsJson(toJson(sourceWordIds));
        item.setGenerationKey(generationKey);
        try {
            sentenceClozeItemMapper.insert(item);
        } catch (DuplicateKeyException exception) {
            if (generationKey == null) {
                throw exception;
            }
            SentenceClozeItem existing = sentenceClozeItemMapper.selectByGenerationKey(generationKey);
            if (existing == null) {
                throw exception;
            }
            linkReviewProgress(existing, parseList(existing.getWordsJson(), String.class));
            return toStoredResponse(existing);
        }

        linkReviewProgress(item, words);
        return toResponse(item, words, sourceEventIds, sourceAnswerDetailIds, sourceRecordIds, sourceWordIds);
    }

    private void linkReviewProgress(SentenceClozeItem item, List<String> words) {
        if (item == null) {
            return;
        }
        wrongWordReviewProgressService.linkGeneratedSentence(
                item.getUserId(),
                item.getId(),
                words,
                LocalDateTime.now()
        );
    }

    private List<String> normalizeWords(SentenceClozeGenerateRequest request) {
        if (request == null) {
            throw new IllegalArgumentException("请求体不能为空");
        }

        Set<String> seen = new LinkedHashSet<>();
        List<String> words = new ArrayList<>();
        addWord(words, seen, request.getWord());
        if (request.getWords() != null) {
            for (String word : request.getWords()) {
                addWord(words, seen, word);
            }
        }

        if (words.isEmpty()) {
            throw new IllegalArgumentException("请至少传入一个单词");
        }
        if (words.size() > MAX_WORD_COUNT) {
            throw new IllegalArgumentException("一次最多支持 12 个单词");
        }
        return words;
    }

    private void addWord(List<String> words, Set<String> seen, String rawWord) {
        if (rawWord == null) {
            return;
        }
        String word = rawWord.trim();
        if (word.isEmpty()) {
            return;
        }
        String key = word.toLowerCase(Locale.ROOT);
        if (seen.add(key)) {
            words.add(word);
        }
    }

    private WordAgentSentenceResponse callWordAgent(List<String> words) {
        RestClient restClient = restClientBuilder
                .baseUrl(stripTrailingSlash(wordAgentBaseUrl))
                .build();

        return restClient.post()
                .uri("/v1/sentences/generate")
                .body(new WordAgentSentenceRequest(words))
                .retrieve()
                .onStatus(HttpStatusCode::isError, (request, response) -> {
                    throw new IllegalStateException("word-agent 造句失败: HTTP " + response.getStatusCode().value());
                })
                .body(WordAgentSentenceResponse.class);
    }

    private String buildClozeSentence(String sentence, List<String> words) {
        String clozeSentence = sentence == null ? "" : sentence;
        for (String word : words) {
            String quotedWord = Pattern.quote(word);
            Pattern boundaryPattern = Pattern.compile("(?i)(?<![A-Za-z])" + quotedWord + "(?![A-Za-z])");
            String nextSentence = boundaryPattern.matcher(clozeSentence).replaceAll(BLANK_TEXT);
            if (nextSentence.equals(clozeSentence)) {
                Pattern fallbackPattern = Pattern.compile(
                        "(?i)(?<![A-Za-z])[A-Za-z]*" + quotedWord + "[A-Za-z]*(?![A-Za-z])");
                nextSentence = fallbackPattern.matcher(clozeSentence).replaceAll(BLANK_TEXT);
            }
            clozeSentence = nextSentence;
        }
        return clozeSentence;
    }

    private List<Long> normalizeLongs(List<Long> values) {
        if (values == null || values.isEmpty()) {
            return List.of();
        }

        Set<Long> seen = new LinkedHashSet<>();
        for (Long value : values) {
            if (value != null && value > 0) {
                seen.add(value);
            }
        }
        return new ArrayList<>(seen);
    }

    private String normalizeUserName(String userName) {
        if (userName == null) {
            return null;
        }
        String cleanedUserName = userName.trim();
        if (cleanedUserName.isEmpty()) {
            return null;
        }
        return cleanedUserName.length() <= 100 ? cleanedUserName : cleanedUserName.substring(0, 100);
    }

    private String normalizeGenerationKey(String generationKey) {
        if (generationKey == null) {
            return null;
        }
        String cleanedKey = generationKey.trim();
        if (cleanedKey.isEmpty()) {
            return null;
        }
        return cleanedKey.length() <= 160 ? cleanedKey : cleanedKey.substring(0, 160);
    }

    private SentenceClozeGenerateResponse toStoredResponse(SentenceClozeItem item) {
        return toResponse(
                item,
                parseList(item.getWordsJson(), String.class),
                parseList(item.getSourceEventIdsJson(), Long.class),
                parseList(item.getSourceAnswerDetailIdsJson(), Long.class),
                parseList(item.getSourceRecordIdsJson(), Long.class),
                parseList(item.getSourceWordIdsJson(), Long.class)
        );
    }

    private <T> List<T> parseList(String json, Class<T> elementType) {
        if (json == null || json.isBlank()) {
            return List.of();
        }
        try {
            return objectMapper.readValue(
                    json,
                    objectMapper.getTypeFactory().constructCollectionType(List.class, elementType)
            );
        } catch (JsonProcessingException e) {
            throw new IllegalStateException("挖空题 JSON 解码失败", e);
        }
    }

    private String toJson(Object value) {
        try {
            return objectMapper.writeValueAsString(value);
        } catch (JsonProcessingException e) {
            throw new IllegalStateException("单词 JSON 编码失败", e);
        }
    }

    private SentenceClozeGenerateResponse toResponse(
            SentenceClozeItem item,
            List<String> words,
            List<Long> sourceEventIds,
            List<Long> sourceAnswerDetailIds,
            List<Long> sourceRecordIds,
            List<Long> sourceWordIds
    ) {
        SentenceClozeGenerateResponse response = new SentenceClozeGenerateResponse();
        response.setId(item.getId());
        response.setUserId(item.getUserId());
        response.setUserName(item.getUserName());
        response.setWord(item.getWord());
        response.setWords(words);
        response.setSentence(item.getSentence());
        response.setSentenceAudioUrl(item.getSentenceAudioUrl());
        response.setTranslationZh(item.getTranslationZh());
        response.setExplanationZh(item.getExplanationZh());
        response.setClozeSentence(item.getClozeSentence());
        response.setBlankWords(words);
        response.setSourceEventIds(sourceEventIds);
        response.setSourceAnswerDetailIds(sourceAnswerDetailIds);
        response.setSourceRecordIds(sourceRecordIds);
        response.setSourceWordIds(sourceWordIds);
        response.setProviderId(item.getProviderId());
        response.setProviderLabel(item.getProviderLabel());
        response.setModel(item.getModel());
        response.setCreateTime(item.getCreateTime());
        return response;
    }

    private String stripTrailingSlash(String url) {
        if (url == null || url.isBlank()) {
            return "http://127.0.0.1:8010";
        }
        String normalizedUrl = url.trim();
        while (normalizedUrl.endsWith("/")) {
            normalizedUrl = normalizedUrl.substring(0, normalizedUrl.length() - 1);
        }
        return normalizedUrl;
    }

    private String requireSentenceAudioUrl(String value) {
        if (value == null || value.isBlank()) {
            throw new IllegalStateException("word-agent 未返回挖空句子 TTS 音频地址");
        }
        return value.trim();
    }

    private record WordAgentSentenceRequest(List<String> words) {
    }

    @Data
    private static class WordAgentSentenceResponse {
        private String sentence;
        private String translationZh;
        private String explanationZh;
        private List<String> words;
        private String sentenceAudioUrl;
        private String providerId;
        private String providerLabel;
        private String model;
    }
}
