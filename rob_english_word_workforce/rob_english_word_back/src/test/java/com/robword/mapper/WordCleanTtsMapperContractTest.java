package com.robword.mapper;

import org.apache.ibatis.annotations.Select;
import org.junit.jupiter.api.Test;

import java.lang.reflect.Method;
import java.util.List;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

class WordCleanTtsMapperContractTest {

    @Test
    void successfulWordAudioLookupUsesExactWords() throws Exception {
        Method method = WordCleanTtsMapper.class.getMethod("selectSuccessfulByWords", List.class);
        String sql = String.join("\n", method.getAnnotation(Select.class).value());

        assertTrue(sql.contains("word IN"));
        assertTrue(sql.contains("status = 'success'"));
        assertTrue(sql.contains("tts_object_url &lt;&gt; ''"));
        assertFalse(sql.contains("lower("));
        assertFalse(sql.contains("btrim("));
    }
}
