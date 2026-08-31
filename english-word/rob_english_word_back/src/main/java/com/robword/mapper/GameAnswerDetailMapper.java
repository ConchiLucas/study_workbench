package com.robword.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.robword.dto.WrongWordDetail;
import com.robword.dto.WrongWordQueueEvent;
import com.robword.dto.WrongWordSummary;
import com.robword.entity.GameAnswerDetail;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;
import org.apache.ibatis.annotations.SelectProvider;

import java.util.List;

@Mapper
public interface GameAnswerDetailMapper extends BaseMapper<GameAnswerDetail> {

    @SelectProvider(type = WrongWordQueueEventSqlProvider.class, method = "selectEvents")
    List<WrongWordQueueEvent> selectQueueEligibleWrongWordEvents(
            @Param("userId") Long userId,
            @Param("keyword") String keyword,
            @Param("sort") String sort,
            @Param("size") Integer size,
            @Param("offset") Long offset
    );

    @SelectProvider(type = WrongWordQueueEventSqlProvider.class, method = "countEvents")
    Long countQueueEligibleWrongWordEvents(
            @Param("userId") Long userId,
            @Param("keyword") String keyword
    );

    @Select("""
            <script>
            WITH ranked_wrong_words AS (
                SELECT
                    d.word_id,
                    d.word_content,
                    d.option_1,
                    d.option_2,
                    d.option_3,
                    d.option_4,
                    d.correct_answer_index,
                    d.create_time,
                    (COUNT(*) OVER (PARTITION BY d.word_id))::int AS wrong_count,
                    ROW_NUMBER() OVER (PARTITION BY d.word_id ORDER BY d.create_time DESC, d.id DESC) AS row_no
                FROM game_answer_detail d
                WHERE d.user_id = #{userId}
                  AND d.is_correct = 0
                  AND d.word_id IS NOT NULL
                  <if test="keyword != null">
                  AND d.word_content ILIKE CONCAT('%', #{keyword}, '%')
                  </if>
            )
            SELECT
                word_id,
                word_content,
                CASE correct_answer_index
                    WHEN 1 THEN option_1
                    WHEN 2 THEN option_2
                    WHEN 3 THEN option_3
                    WHEN 4 THEN option_4
                    ELSE NULL
                END AS correct_meaning,
                wrong_count,
                create_time AS last_wrong_time
            FROM ranked_wrong_words
            WHERE row_no = 1
            <choose>
                <when test="sort == 'count'">
                    ORDER BY wrong_count DESC, last_wrong_time DESC
                </when>
                <otherwise>
                    ORDER BY last_wrong_time DESC
                </otherwise>
            </choose>
            LIMIT #{size} OFFSET #{offset}
            </script>
            """)
    List<WrongWordSummary> selectWrongWordSummaries(@Param("userId") Long userId,
                                                    @Param("keyword") String keyword,
                                                    @Param("sort") String sort,
                                                    @Param("size") Integer size,
                                                    @Param("offset") Long offset);

    @Select("""
            <script>
            WITH filtered_wrong_words AS (
                SELECT d.word_id
                FROM game_answer_detail d
                WHERE d.user_id = #{userId}
                  AND d.is_correct = 0
                  AND d.word_id IS NOT NULL
                  <if test="keyword != null">
                  AND d.word_content ILIKE CONCAT('%', #{keyword}, '%')
                  </if>
                GROUP BY d.word_id
            )
            SELECT COUNT(*)
            FROM filtered_wrong_words
            </script>
            """)
    Long countWrongWordSummaries(@Param("userId") Long userId,
                                 @Param("keyword") String keyword);

    @Select("""
            SELECT
                d.id,
                d.word_content,
                CASE d.correct_answer_index
                    WHEN 1 THEN d.option_1
                    WHEN 2 THEN d.option_2
                    WHEN 3 THEN d.option_3
                    WHEN 4 THEN d.option_4
                    ELSE NULL
                END AS correct_meaning,
                d.create_time
            FROM game_answer_detail d
            WHERE d.user_id = #{userId}
              AND d.word_id = #{wordId}
              AND d.is_correct = 0
            ORDER BY d.create_time DESC, d.id DESC
            LIMIT #{limit}
            """)
    List<WrongWordDetail> selectWrongWordDetails(@Param("userId") Long userId,
                                                 @Param("wordId") Long wordId,
                                                 @Param("limit") Integer limit);

    @Select("""
            SELECT COUNT(*)
            FROM game_answer_detail d
            WHERE d.user_id = #{userId}
              AND d.word_id = #{wordId}
              AND d.is_correct = 0
            """)
    Long countWrongWordDetails(@Param("userId") Long userId,
                               @Param("wordId") Long wordId);
}
