package com.robword.service;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.robword.dto.SentenceClozeGenerateRequest;
import com.robword.dto.SentenceClozeGenerateResponse;
import com.robword.entity.SentenceClozeItem;
import com.robword.mapper.SentenceClozeItemMapper;
import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpServer;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;
import org.springframework.dao.DuplicateKeyException;
import org.springframework.test.util.ReflectionTestUtils;
import org.springframework.web.client.RestClient;

import java.io.IOException;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.util.List;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class SentenceClozeServiceTest {

    private HttpServer server;
    private final WrongWordReviewProgressService progressService =
            mock(WrongWordReviewProgressService.class);

    @AfterEach
    void stopServer() {
        if (server != null) {
            server.stop(0);
        }
    }

    @Test
    void returnsExistingItemForRepeatedGenerationKeyWithoutCallingWordAgent() {
        SentenceClozeItemMapper mapper = mock(SentenceClozeItemMapper.class);
        SentenceClozeItem existing = storedItem("wrong-word-events:47-48-49");
        when(mapper.selectByGenerationKey("wrong-word-events:47-48-49"))
                .thenReturn(existing);
        SentenceClozeService service = createServiceWithoutServer(mapper);
        SentenceClozeGenerateRequest request = request();
        request.setGenerationKey("wrong-word-events:47-48-49");

        SentenceClozeGenerateResponse response = service.generateAndSave(request);

        assertEquals(existing.getId(), response.getId());
        verify(mapper, never()).insert(any(SentenceClozeItem.class));
        verify(progressService).linkGeneratedSentence(
                eq(1L),
                eq(99L),
                eq(List.of("brisk", "anchor", "harbor")),
                any()
        );
    }

    @Test
    void returnsExistingItemWhenGenerationKeyInsertRaces() throws IOException {
        startWordAgent("""
                {
                  "sentence": "A brisk anchor can harbor a quiet plan.",
                  "translationZh": "译文",
                  "explanationZh": "解释",
                  "providerId": "provider",
                  "providerLabel": "Provider",
                  "model": "model",
                  "sentenceAudioUrl": "/ai-file-navigation/sentence_cloze_tts/example.wav"
                }
                """);
        SentenceClozeItemMapper mapper = mock(SentenceClozeItemMapper.class);
        SentenceClozeItem existing = storedItem("wrong-word-events:47-48-49");
        when(mapper.selectByGenerationKey("wrong-word-events:47-48-49"))
                .thenReturn(null, existing);
        when(mapper.insert(any(SentenceClozeItem.class)))
                .thenThrow(new DuplicateKeyException("duplicate generation key"));
        SentenceClozeService service = createService(mapper);
        SentenceClozeGenerateRequest request = request();
        request.setGenerationKey("wrong-word-events:47-48-49");

        SentenceClozeGenerateResponse response = service.generateAndSave(request);

        assertEquals(existing.getId(), response.getId());
    }

    @Test
    void savesSentenceAudioUrlReturnedByWordAgent() throws IOException {
        String audioUrl = "/ai-file-navigation/sentence_cloze_tts/example.wav";
        startWordAgent("""
                {
                  "sentence": "A brisk anchor can harbor a quiet plan.",
                  "translationZh": "译文",
                  "explanationZh": "解释",
                  "providerId": "provider",
                  "providerLabel": "Provider",
                  "model": "model",
                  "sentenceAudioUrl": "%s"
                }
                """.formatted(audioUrl));
        SentenceClozeItemMapper mapper = mock(SentenceClozeItemMapper.class);
        when(mapper.insert(any(SentenceClozeItem.class))).thenAnswer(invocation -> {
            SentenceClozeItem item = invocation.getArgument(0);
            item.setId(101L);
            return 1;
        });
        SentenceClozeService service = createService(mapper);
        SentenceClozeGenerateRequest request = request();
        request.setGenerationKey("wrong-word-events:47-48-49");

        SentenceClozeGenerateResponse response = service.generateAndSave(request);

        ArgumentCaptor<SentenceClozeItem> captor = ArgumentCaptor.forClass(SentenceClozeItem.class);
        verify(mapper).insert(captor.capture());
        assertEquals(audioUrl, captor.getValue().getSentenceAudioUrl());
        assertEquals("wrong-word-events:47-48-49", captor.getValue().getGenerationKey());
        assertEquals(audioUrl, response.getSentenceAudioUrl());
        verify(progressService).linkGeneratedSentence(
                eq(1L),
                eq(101L),
                eq(List.of("brisk", "anchor", "harbor")),
                any()
        );
    }

    @Test
    void rejectsResponseWithoutSentenceAudioUrlBeforeInsert() throws IOException {
        startWordAgent("""
                {
                  "sentence": "A brisk anchor can harbor a quiet plan.",
                  "translationZh": "译文",
                  "explanationZh": "解释",
                  "providerId": "provider",
                  "providerLabel": "Provider",
                  "model": "model"
                }
                """);
        SentenceClozeItemMapper mapper = mock(SentenceClozeItemMapper.class);
        SentenceClozeService service = createService(mapper);

        assertThrows(IllegalStateException.class, () -> service.generateAndSave(request()));
        verify(mapper, never()).insert(any(SentenceClozeItem.class));
    }

    @Test
    void doesNotInsertWhenWordAgentRequestFails() throws IOException {
        startWordAgent("{\"detail\":\"tts unavailable\"}", 502);
        SentenceClozeItemMapper mapper = mock(SentenceClozeItemMapper.class);
        SentenceClozeService service = createService(mapper);

        assertThrows(IllegalStateException.class, () -> service.generateAndSave(request()));
        verify(mapper, never()).insert(any(SentenceClozeItem.class));
    }

    private SentenceClozeService createService(SentenceClozeItemMapper mapper) {
        SentenceClozeService service = new SentenceClozeService(
                mapper,
                new ObjectMapper(),
                RestClient.builder(),
                progressService
        );
        ReflectionTestUtils.setField(
                service,
                "wordAgentBaseUrl",
                "http://127.0.0.1:" + server.getAddress().getPort()
        );
        return service;
    }

    private SentenceClozeService createServiceWithoutServer(SentenceClozeItemMapper mapper) {
        SentenceClozeService service = new SentenceClozeService(
                mapper,
                new ObjectMapper(),
                RestClient.builder(),
                progressService
        );
        ReflectionTestUtils.setField(service, "wordAgentBaseUrl", "http://127.0.0.1:1");
        return service;
    }

    private SentenceClozeItem storedItem(String generationKey) {
        SentenceClozeItem item = new SentenceClozeItem();
        item.setId(99L);
        item.setUserId(1L);
        item.setWord("brisk");
        item.setWordsJson("[\"brisk\",\"anchor\",\"harbor\"]");
        item.setBlankWordsJson("[\"brisk\",\"anchor\",\"harbor\"]");
        item.setSentence("A brisk anchor can harbor a quiet plan.");
        item.setSentenceAudioUrl("/ai-file-navigation/sentence_cloze_tts/example.wav");
        item.setTranslationZh("译文");
        item.setExplanationZh("解释");
        item.setClozeSentence("A ____ ____ can ____ a quiet plan.");
        item.setSourceEventIdsJson("[47,48,49]");
        item.setSourceAnswerDetailIdsJson("[147,148,149]");
        item.setSourceRecordIdsJson("[10]");
        item.setSourceWordIdsJson("[247,248,249]");
        item.setGenerationKey(generationKey);
        return item;
    }

    private SentenceClozeGenerateRequest request() {
        SentenceClozeGenerateRequest request = new SentenceClozeGenerateRequest();
        request.setUserId(1L);
        request.setWords(List.of("brisk", "anchor", "harbor"));
        return request;
    }

    private void startWordAgent(String responseJson) throws IOException {
        startWordAgent(responseJson, 200);
    }

    private void startWordAgent(String responseJson, int statusCode) throws IOException {
        server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        server.createContext(
                "/v1/sentences/generate",
                exchange -> respond(exchange, responseJson, statusCode)
        );
        server.start();
    }

    private void respond(HttpExchange exchange, String body, int statusCode) throws IOException {
        byte[] bytes = body.getBytes(StandardCharsets.UTF_8);
        exchange.getResponseHeaders().set("Content-Type", "application/json");
        exchange.sendResponseHeaders(statusCode, bytes.length);
        exchange.getResponseBody().write(bytes);
        exchange.close();
    }
}
