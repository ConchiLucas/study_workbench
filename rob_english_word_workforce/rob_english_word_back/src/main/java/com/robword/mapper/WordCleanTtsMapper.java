package com.robword.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.robword.entity.WordCleanTts;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;

import java.util.List;

@Mapper
public interface WordCleanTtsMapper extends BaseMapper<WordCleanTts> {

    @Select("""
            <script>
            SELECT id, word_clean_id, word, status, tts_object_url
            FROM word_clean_tts
            WHERE word IN
              <foreach collection="words" item="word" open="(" separator="," close=")">
                #{word}
              </foreach>
              AND status = 'success'
              AND tts_object_url &lt;&gt; ''
            </script>
            """)
    List<WordCleanTts> selectSuccessfulByWords(@Param("words") List<String> words);
}
