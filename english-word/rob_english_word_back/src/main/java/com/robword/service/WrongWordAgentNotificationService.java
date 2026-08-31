package com.robword.service;

import com.robword.entity.GameAnswerDetail;
import com.robword.entity.SentenceClozeAnswerRecord;
import com.robword.entity.SentenceClozeItem;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.HttpStatusCode;
import org.springframework.stereotype.Service;
import org.springframework.web.client.RestClient;

import java.util.Arrays;
import java.util.List;
import java.util.concurrent.CompletableFuture;

@Service
@RequiredArgsConstructor
@Slf4j
public class WrongWordAgentNotificationService {

    private final RestClient.Builder restClientBuilder;

    @Value("${word-agent.base-url:http://127.0.0.1:8010}")
    private String wordAgentBaseUrl;

    public void notifyWrongAnswer(GameAnswerDetail detail) {
        if (detail == null || detail.getIsCorrect() == null || detail.getIsCorrect() != 0) {
            return;
        }
        if (detail.getUserId() == null || detail.getUserId() < 0) {
            return;
        }
        if (detail.getId() == null || detail.getWordContent() == null || detail.getWordContent().isBlank()) {
            return;
        }

        CompletableFuture.runAsync(() -> sendWrongAnswer(detail))
                .exceptionally(error -> {
                    log.warn("Failed to notify word-agent wrong answer: detailId={}, error={}",
                            detail.getId(), error.getMessage());
                    return null;
                });
    }

    public void notifyClozeWrongAnswer(
            SentenceClozeAnswerRecord record,
            SentenceClozeItem item,
            List<String> expectedWords,
            List<String> answers,
            List<Integer> wrongIndexes
    ) {
        if (record == null || item == null || expectedWords == null || expectedWords.isEmpty()
                || wrongIndexes == null || wrongIndexes.isEmpty()) {
            return;
        }
        if (!Boolean.FALSE.equals(record.getIsCorrect())) {
            return;
        }
        if (record.getId() == null || record.getUserId() == null || record.getUserId() < 0) {
            return;
        }

        List<Integer> wrongIndexSnapshot = List.copyOf(wrongIndexes);
        CompletableFuture.runAsync(() -> sendClozeWrongAnswer(
                        record, item, expectedWords, answers, wrongIndexSnapshot))
                .exceptionally(error -> {
                    log.warn("Failed to notify word-agent cloze wrong answer: recordId={}, error={}",
                            record.getId(), error.getMessage());
                    return null;
                });
    }

    private void sendWrongAnswer(GameAnswerDetail detail) {
        WrongWordEventRequest request = new WrongWordEventRequest(
                "rob_english_word_back",
                detail.getId(),
                detail.getRecordId(),
                detail.getUserId(),
                detail.getUserName(),
                detail.getWordId(),
                detail.getWordContent(),
                detail.getWordDifficulty(),
                Arrays.asList(detail.getOption1(), detail.getOption2(), detail.getOption3(), detail.getOption4()),
                detail.getCorrectAnswerIndex(),
                detail.getSelectedAnswerIndex(),
                resolveOption(detail, detail.getCorrectAnswerIndex()),
                resolveOption(detail, detail.getSelectedAnswerIndex())
        );

        RestClient restClient = restClientBuilder.baseUrl(stripTrailingSlash(wordAgentBaseUrl)).build();
        restClient.post()
                .uri("/v1/wrong-words/events")
                .body(request)
                .retrieve()
                .onStatus(HttpStatusCode::isError, (httpRequest, response) -> {
                    throw new IllegalStateException(
                            "word-agent wrong event failed: HTTP " + response.getStatusCode().value());
                })
                .toBodilessEntity();
    }

    private void sendClozeWrongAnswer(
            SentenceClozeAnswerRecord record,
            SentenceClozeItem item,
            List<String> expectedWords,
            List<String> answers,
            List<Integer> wrongIndexes
    ) {
        RestClient restClient = restClientBuilder.baseUrl(stripTrailingSlash(wordAgentBaseUrl)).build();
        for (Integer wrongIndex : wrongIndexes) {
            String word = expectedWords.get(wrongIndex);
            if (word == null || word.isBlank()) {
                continue;
            }

            WrongWordEventRequest request = new WrongWordEventRequest(
                    "sentence_cloze_practice",
                    buildClozeAnswerDetailId(record.getId(), wrongIndex),
                    item.getId(),
                    record.getUserId(),
                    record.getUserName(),
                    null,
                    word.trim(),
                    null,
                    buildClozeContextOptions(item, record),
                    wrongIndex + 1,
                    null,
                    item.getTranslationZh(),
                    resolveAnswerAt(answers, wrongIndex)
            );

            restClient.post()
                    .uri("/v1/wrong-words/events")
                    .body(request)
                    .retrieve()
                    .onStatus(HttpStatusCode::isError, (httpRequest, response) -> {
                        throw new IllegalStateException(
                                "word-agent cloze wrong event failed: HTTP " + response.getStatusCode().value());
                    })
                    .toBodilessEntity();
        }
    }

    private Long buildClozeAnswerDetailId(Long recordId, int wrongIndex) {
        return recordId * 1000 + wrongIndex + 1;
    }

    private List<String> buildClozeContextOptions(SentenceClozeItem item, SentenceClozeAnswerRecord record) {
        return Arrays.asList(
                item.getTranslationZh(),
                item.getClozeSentence(),
                record.getAnswerText(),
                item.getSentence()
        );
    }

    private String resolveAnswerAt(List<String> answers, int index) {
        if (answers == null || index < 0 || index >= answers.size()) {
            return null;
        }
        return answers.get(index);
    }

    private String resolveOption(GameAnswerDetail detail, Integer index) {
        if (index == null) {
            return null;
        }
        return switch (index) {
            case 1 -> detail.getOption1();
            case 2 -> detail.getOption2();
            case 3 -> detail.getOption3();
            case 4 -> detail.getOption4();
            default -> null;
        };
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

    private record WrongWordEventRequest(
            String source,
            Long answerDetailId,
            Long recordId,
            Long userId,
            String userName,
            Long wordId,
            String word,
            Integer wordDifficulty,
            List<String> options,
            Integer correctAnswerIndex,
            Integer selectedAnswerIndex,
            String correctMeaning,
            String selectedMeaning
    ) {
    }
}
