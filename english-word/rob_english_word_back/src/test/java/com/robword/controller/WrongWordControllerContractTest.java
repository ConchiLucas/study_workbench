package com.robword.controller;

import com.robword.dto.WrongWordQueueEvent;
import com.robword.mapper.GameAnswerDetailMapper;
import com.robword.mapper.UserWordStateMapper;
import org.junit.jupiter.api.Test;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.GetMapping;

import java.lang.reflect.Method;
import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.assertArrayEquals;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class WrongWordControllerContractTest {

    @Test
    void queueEventCarriesResolvedExampleSentenceMetadata() throws Exception {
        assertEquals(String.class, WrongWordQueueEvent.class
                .getDeclaredField("exampleSentence").getType());
        assertEquals(String.class, WrongWordQueueEvent.class
                .getDeclaredField("exampleSource").getType());
    }

    @Test
    void exposesUnfinishedWrongWordsWithoutReplacingLegacyRoutes() throws Exception {
        Method method = WrongWordController.class.getMethod(
                "listWrongWordEvents",
                Authentication.class,
                String.class,
                String.class,
                Integer.class,
                Integer.class
        );

        GetMapping mapping = method.getAnnotation(GetMapping.class);
        assertArrayEquals(new String[]{"/events"}, mapping.value());
    }

    @Test
    void normalizesFiltersAndMapsEventPaginationResponse() {
        GameAnswerDetailMapper gameMapper = mock(GameAnswerDetailMapper.class);
        UserWordStateMapper stateMapper = mock(UserWordStateMapper.class);
        Authentication authentication = mock(Authentication.class);
        WrongWordQueueEvent event = new WrongWordQueueEvent();
        event.setProgressKey("progress:7");
        event.setWord("raw");

        when(authentication.getPrincipal()).thenReturn(2L);
        when(gameMapper.selectQueueEligibleWrongWordEvents(2L, "raw", "recent", 500, 0L))
                .thenReturn(List.of(event));
        when(gameMapper.countQueueEligibleWrongWordEvents(2L, "raw")).thenReturn(42L);

        WrongWordController controller = new WrongWordController(gameMapper, stateMapper);
        ResponseEntity<Map<String, Object>> response = controller.listWrongWordEvents(
                authentication, "  raw  ", "unsupported", 0, 999);

        Map<String, Object> body = response.getBody();
        assertEquals(List.of(event), body.get("items"));
        assertEquals(42L, body.get("total"));
        assertEquals(1, body.get("current"));
        assertEquals(1L, body.get("pages"));
        verify(gameMapper).selectQueueEligibleWrongWordEvents(2L, "raw", "recent", 500, 0L);
        verify(gameMapper).countQueueEligibleWrongWordEvents(2L, "raw");
    }
}
