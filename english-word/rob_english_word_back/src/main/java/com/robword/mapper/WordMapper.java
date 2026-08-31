package com.robword.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.robword.entity.Word;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;

import java.util.List;

@Mapper
public interface WordMapper extends BaseMapper<Word> {

    @Select("SELECT * FROM word WHERE library_id = #{libraryId} AND status = 1")
    List<Word> findByLibraryId(@Param("libraryId") Long libraryId);

    @Select("SELECT * FROM word w WHERE w.status = 1 AND w.difficulty >= #{minDiff} AND w.difficulty <= #{maxDiff} ORDER BY RANDOM() LIMIT #{limit}")
    List<Word> findRandomWords(@Param("minDiff") Integer minDiff, @Param("maxDiff") Integer maxDiff, @Param("limit") Integer limit);

    @Select("""
            SELECT *
            FROM word
            WHERE status = 1
              AND word IS NOT NULL
              AND lower(btrim(word)) = lower(btrim(#{wordContent}))
            ORDER BY COALESCE(difficulty, 999999) ASC, COALESCE(frequency, 999999) ASC, id ASC
            LIMIT 1
            """)
    Word findActiveWordByContent(@Param("wordContent") String wordContent);

    @Select({
            "<script>",
            "SELECT * FROM word w",
            "WHERE w.status = 1",
            "  AND w.difficulty &gt;= #{minDiff}",
            "  AND w.difficulty &lt;= #{maxDiff}",
            "  AND NOT EXISTS (",
            "    SELECT 1 FROM user_word_mastery_progress p",
            "    WHERE p.user_id = #{userId}",
            "      AND p.status = 'mastered'",
            "      AND (",
            "        p.word_id = w.id",
            "        OR (p.word_content IS NOT NULL",
            "        AND w.word IS NOT NULL",
            "        AND lower(btrim(p.word_content)) = lower(btrim(w.word)))",
            "      )",
            "  )",
            "  <if test='excludeWordIds != null and excludeWordIds.size() > 0'>",
            "  AND w.id NOT IN",
            "  <foreach collection='excludeWordIds' item='wordId' open='(' separator=',' close=')'>",
            "    #{wordId}",
            "  </foreach>",
            "  </if>",
            "ORDER BY RANDOM()",
            "LIMIT #{limit}",
            "</script>"
    })
    List<Word> findRandomWordsExcludingMastered(@Param("userId") Long userId,
                                                @Param("minDiff") Integer minDiff,
                                                @Param("maxDiff") Integer maxDiff,
                                                @Param("limit") Integer limit,
                                                @Param("excludeWordIds") List<Long> excludeWordIds);

    @Select({
            "<script>",
            "SELECT * FROM (",
            "  SELECT DISTINCT ON (lower(btrim(w.word))) w.*",
            "  FROM word w",
            "  JOIN word_library wl ON wl.id = w.library_id",
            "  WHERE w.status = 1",
            "    AND w.word IS NOT NULL",
            "    AND btrim(w.word) &lt;&gt; ''",
            "    AND wl.library_name IN",
            "    <foreach collection='libraryNames' item='name' open='(' separator=',' close=')'>",
            "      #{name}",
            "    </foreach>",
            "  ORDER BY lower(btrim(w.word)), RANDOM()",
            ") picked",
            "ORDER BY RANDOM()",
            "LIMIT #{limit}",
            "</script>"
    })
    List<Word> findRandomWordsByLibraryNames(@Param("libraryNames") List<String> libraryNames,
                                             @Param("limit") Integer limit);

    @Select({
            "<script>",
            "SELECT * FROM (",
            "  SELECT DISTINCT ON (lower(btrim(w.word))) w.*",
            "  FROM word w",
            "  JOIN word_library wl ON wl.id = w.library_id",
            "  WHERE w.status = 1",
            "    AND w.word IS NOT NULL",
            "    AND btrim(w.word) &lt;&gt; ''",
            "    AND wl.library_name IN",
            "    <foreach collection='libraryNames' item='name' open='(' separator=',' close=')'>",
            "      #{name}",
            "    </foreach>",
            "    AND NOT EXISTS (",
            "      SELECT 1 FROM user_word_mastery_progress p",
            "      WHERE p.user_id = #{userId}",
            "        AND p.status = 'mastered'",
            "        AND (",
            "          p.word_id = w.id",
            "          OR (p.word_content IS NOT NULL",
            "          AND w.word IS NOT NULL",
            "          AND lower(btrim(p.word_content)) = lower(btrim(w.word)))",
            "        )",
            "    )",
            "    <if test='excludeWordIds != null and excludeWordIds.size() > 0'>",
            "    AND w.id NOT IN",
            "    <foreach collection='excludeWordIds' item='wordId' open='(' separator=',' close=')'>",
            "      #{wordId}",
            "    </foreach>",
            "    </if>",
            "  ORDER BY lower(btrim(w.word)), RANDOM()",
            ") picked",
            "ORDER BY RANDOM()",
            "LIMIT #{limit}",
            "</script>"
    })
    List<Word> findRandomWordsByLibraryNamesExcludingMastered(@Param("userId") Long userId,
                                                              @Param("libraryNames") List<String> libraryNames,
                                                              @Param("limit") Integer limit,
                                                              @Param("excludeWordIds") List<Long> excludeWordIds);
}
