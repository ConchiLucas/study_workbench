package com.robword.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.robword.dto.MasteredWordSummary;
import com.robword.entity.UserWordMasteryProgress;
import com.robword.entity.Word;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;

import java.util.List;

@Mapper
public interface UserWordMasteryProgressMapper extends BaseMapper<UserWordMasteryProgress> {

    @Select("""
            <script>
            SELECT
                p.word_id,
                COALESCE(NULLIF(p.word_content, ''), w.word) AS word_content,
                COALESCE(NULLIF(p.correct_meaning, ''), w.meaning) AS correct_meaning,
                p.status,
                p.stage,
                p.correct_count,
                p.first_correct_time,
                p.day1_correct_time,
                p.day7_correct_time,
                p.next_review_time,
                p.last_correct_time,
                p.mastered_time
            FROM user_word_mastery_progress p
            LEFT JOIN word w ON w.id = p.word_id
            WHERE p.user_id = #{userId}
              <if test="status != null">
              AND p.status = #{status}
              </if>
              <if test="keyword != null">
              AND (
                COALESCE(NULLIF(p.word_content, ''), w.word, '') ILIKE CONCAT('%', #{keyword}, '%')
                OR COALESCE(NULLIF(p.correct_meaning, ''), w.meaning, '') ILIKE CONCAT('%', #{keyword}, '%')
              )
              </if>
            <choose>
                <when test="sort == 'due'">
                    ORDER BY p.next_review_time ASC NULLS LAST, p.last_correct_time DESC NULLS LAST
                </when>
                <when test="sort == 'mastered'">
                    ORDER BY p.mastered_time DESC NULLS LAST, p.last_correct_time DESC NULLS LAST
                </when>
                <otherwise>
                    ORDER BY p.last_correct_time DESC NULLS LAST, p.update_time DESC
                </otherwise>
            </choose>
            LIMIT #{size} OFFSET #{offset}
            </script>
            """)
    List<MasteredWordSummary> selectMasterySummaries(@Param("userId") Long userId,
                                                     @Param("keyword") String keyword,
                                                     @Param("status") String status,
                                                     @Param("sort") String sort,
                                                     @Param("size") Integer size,
                                                     @Param("offset") Long offset);

    @Select("""
            <script>
            SELECT COUNT(*)
            FROM user_word_mastery_progress p
            LEFT JOIN word w ON w.id = p.word_id
            WHERE p.user_id = #{userId}
              <if test="status != null">
              AND p.status = #{status}
              </if>
              <if test="keyword != null">
              AND (
                COALESCE(NULLIF(p.word_content, ''), w.word, '') ILIKE CONCAT('%', #{keyword}, '%')
                OR COALESCE(NULLIF(p.correct_meaning, ''), w.meaning, '') ILIKE CONCAT('%', #{keyword}, '%')
              )
              </if>
            </script>
            """)
    Long countMasterySummaries(@Param("userId") Long userId,
                               @Param("keyword") String keyword,
                               @Param("status") String status);

    @Select("""
            <script>
            SELECT w.*
            FROM user_word_mastery_progress p
            JOIN word w ON w.id = p.word_id
            <if test="libraryNames != null and libraryNames.size() > 0">
            JOIN word_library wl ON wl.id = w.library_id
            </if>
            WHERE p.user_id = #{userId}
              AND p.status = 'learning'
              AND p.stage IN (1, 2)
              AND p.next_review_time IS NOT NULL
              AND p.next_review_time &lt;= CURRENT_TIMESTAMP
              AND w.status = 1
              AND NOT EXISTS (
                SELECT 1 FROM user_word_mastery_progress mastered
                WHERE mastered.user_id = p.user_id
                  AND mastered.status = 'mastered'
                  AND (
                    mastered.word_id = p.word_id
                    OR (
                      mastered.word_content IS NOT NULL
                      AND COALESCE(NULLIF(p.word_content, ''), w.word) IS NOT NULL
                      AND lower(btrim(mastered.word_content)) = lower(btrim(COALESCE(NULLIF(p.word_content, ''), w.word)))
                    )
                  )
              )
              <if test="libraryNames != null and libraryNames.size() > 0">
              AND wl.library_name IN
              <foreach collection="libraryNames" item="name" open="(" separator="," close=")">
                #{name}
              </foreach>
              </if>
            ORDER BY p.next_review_time ASC, p.last_correct_time ASC NULLS LAST
            LIMIT #{limit}
            </script>
            """)
    List<Word> selectDueReviewWords(@Param("userId") Long userId,
                                    @Param("libraryNames") List<String> libraryNames,
                                    @Param("limit") Integer limit);
}
