package com.robword.controller;

import com.robword.dto.ClozeWrongSentenceDetail;
import com.robword.dto.ClozeWrongSentencePageResponse;
import org.junit.jupiter.api.Test;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.GetMapping;

import java.lang.reflect.Method;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;

class ClozeWrongSentenceControllerContractTest {

    @Test
    void exposesPaginatedWrongSentenceList() throws Exception {
        Method method = ClozePracticeController.class.getMethod(
                "getWrongSentences",
                Authentication.class,
                String.class,
                String.class,
                String.class,
                String.class,
                String.class,
                Integer.class,
                Integer.class
        );

        assertEquals(ResponseEntity.class, method.getReturnType());
        GetMapping mapping = method.getAnnotation(GetMapping.class);
        assertNotNull(mapping);
        assertEquals("/wrong-sentences", mapping.value()[0]);
        assertNotNull(ClozeWrongSentencePageResponse.class);
    }

    @Test
    void exposesOwnedWrongSentenceDetail() throws Exception {
        Method method = ClozePracticeController.class.getMethod(
                "getWrongSentenceDetail", Authentication.class, Long.class
        );

        assertEquals(ResponseEntity.class, method.getReturnType());
        GetMapping mapping = method.getAnnotation(GetMapping.class);
        assertNotNull(mapping);
        assertEquals("/wrong-sentences/{progressId}", mapping.value()[0]);
        assertNotNull(ClozeWrongSentenceDetail.class);
    }
}
