package com.robword.mapper;

import com.robword.dto.ClozeWrongSentenceAttempt;
import com.robword.dto.ClozeWrongSentenceItem;
import com.robword.dto.ClozeWrongSentencePageResponse;
import com.robword.entity.WrongWordReviewProgress;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;

import java.util.List;

@Mapper
public interface ClozeWrongSentenceQueryMapper {

    @Select("""
            <script>
            SELECT
                progress.id AS progress_id,
                progress.cloze_item_id,
                item.cloze_sentence,
                item.sentence,
                item.translation_zh,
                item.blank_words_json AS target_words_json,
                last_wrong.wrong_blank_indexes_json,
                COALESCE(
                    last_wrong.practice_context,
                    CASE WHEN item.source = 'word-agent' THEN 'review' ELSE 'solo' END
                ) AS practice_context,
                item.source AS content_source,
                CASE
                    WHEN item.source = 'best-sentence-practice' THEN COALESCE(item.provider_label, '')
                    ELSE ''
                END AS difficulty_label,
                progress.status,
                progress.review_stage,
                progress.next_review_time,
                progress.wrong_count,
                progress.first_wrong_time,
                progress.last_wrong_time,
                last_wrong.cost_ms AS last_cost_ms
            FROM sentence_cloze_review_schedule progress
            JOIN sentence_cloze_item item
              ON item.id = progress.cloze_item_id
            LEFT JOIN sentence_cloze_answer_record last_wrong
              ON last_wrong.id = progress.last_wrong_answer_record_id
             AND last_wrong.user_id = #{userId}
            WHERE progress.user_id = #{userId}
              AND item.user_id = #{userId}
              AND progress.status = #{status}
              AND (
                  #{source} = 'all'
                  OR COALESCE(
                      last_wrong.practice_context,
                      CASE WHEN item.source = 'word-agent' THEN 'review' ELSE 'solo' END
                  ) = #{source}
              )
              AND (
                  #{availability} = 'all'
                  OR (#{availability} = 'due' AND progress.next_review_time &lt;= NOW())
                  OR (#{availability} = 'waiting' AND progress.next_review_time &gt; NOW())
              )
              AND (
                  #{keyword} = ''
                  OR LOWER(COALESCE(item.sentence, '')) LIKE CONCAT('%', #{keyword}, '%')
                  OR LOWER(COALESCE(item.translation_zh, '')) LIKE CONCAT('%', #{keyword}, '%')
                  OR EXISTS (
                      SELECT 1
                      FROM jsonb_array_elements_text(
                          COALESCE(NULLIF(item.blank_words_json, ''), '[]')::jsonb
                      ) target_word(word)
                      WHERE LOWER(target_word.word) LIKE CONCAT('%', #{keyword}, '%')
                  )
              )
            <choose>
                <when test="sort == 'wrongCount'">
                    ORDER BY progress.wrong_count DESC, progress.last_wrong_time DESC, progress.id DESC
                </when>
                <when test="sort == 'recent'">
                    ORDER BY progress.last_wrong_time DESC, progress.id DESC
                </when>
                <otherwise>
                    ORDER BY progress.next_review_time ASC NULLS LAST,
                             progress.last_wrong_time DESC,
                             progress.id DESC
                </otherwise>
            </choose>
            LIMIT #{size} OFFSET #{offset}
            </script>
            """)
    List<ClozeWrongSentenceItem> selectWrongSentences(
            @Param("userId") Long userId,
            @Param("status") String status,
            @Param("source") String source,
            @Param("availability") String availability,
            @Param("keyword") String keyword,
            @Param("sort") String sort,
            @Param("offset") Integer offset,
            @Param("size") Integer size
    );

    @Select("""
            <script>
            SELECT COUNT(*)
            FROM sentence_cloze_review_schedule progress
            JOIN sentence_cloze_item item ON item.id = progress.cloze_item_id
            LEFT JOIN sentence_cloze_answer_record last_wrong
              ON last_wrong.id = progress.last_wrong_answer_record_id
             AND last_wrong.user_id = #{userId}
            WHERE progress.user_id = #{userId}
              AND item.user_id = #{userId}
              AND progress.status = #{status}
              AND (
                  #{source} = 'all'
                  OR COALESCE(
                      last_wrong.practice_context,
                      CASE WHEN item.source = 'word-agent' THEN 'review' ELSE 'solo' END
                  ) = #{source}
              )
              AND (
                  #{availability} = 'all'
                  OR (#{availability} = 'due' AND progress.next_review_time &lt;= NOW())
                  OR (#{availability} = 'waiting' AND progress.next_review_time &gt; NOW())
              )
              AND (
                  #{keyword} = ''
                  OR LOWER(COALESCE(item.sentence, '')) LIKE CONCAT('%', #{keyword}, '%')
                  OR LOWER(COALESCE(item.translation_zh, '')) LIKE CONCAT('%', #{keyword}, '%')
                  OR EXISTS (
                      SELECT 1
                      FROM jsonb_array_elements_text(
                          COALESCE(NULLIF(item.blank_words_json, ''), '[]')::jsonb
                      ) target_word(word)
                      WHERE LOWER(target_word.word) LIKE CONCAT('%', #{keyword}, '%')
                  )
              )
            </script>
            """)
    Long countWrongSentences(
            @Param("userId") Long userId,
            @Param("status") String status,
            @Param("source") String source,
            @Param("availability") String availability,
            @Param("keyword") String keyword
    );

    @Select("""
            SELECT
                COUNT(*) FILTER (WHERE progress.status = 'active') AS active_count,
                COUNT(*) FILTER (
                    WHERE progress.status = 'active' AND progress.next_review_time <= NOW()
                ) AS due_count,
                COUNT(*) FILTER (
                    WHERE progress.status = 'active' AND progress.review_stage = 1
                ) AS stage1_count,
                COUNT(*) FILTER (
                    WHERE progress.status = 'active' AND progress.review_stage = 2
                ) AS stage2_count,
                COUNT(*) FILTER (WHERE progress.status = 'completed') AS completed_count
            FROM sentence_cloze_review_schedule progress
            JOIN sentence_cloze_item item ON item.id = progress.cloze_item_id
            WHERE progress.user_id = #{userId}
              AND item.user_id = #{userId}
            """)
    ClozeWrongSentencePageResponse.Summary selectSummary(@Param("userId") Long userId);

    @Select("""
            SELECT
                progress.id AS progress_id,
                progress.cloze_item_id,
                item.cloze_sentence,
                item.sentence,
                item.translation_zh,
                item.blank_words_json AS target_words_json,
                last_wrong.wrong_blank_indexes_json,
                COALESCE(
                    last_wrong.practice_context,
                    CASE WHEN item.source = 'word-agent' THEN 'review' ELSE 'solo' END
                ) AS practice_context,
                item.source AS content_source,
                CASE
                    WHEN item.source = 'best-sentence-practice' THEN COALESCE(item.provider_label, '')
                    ELSE ''
                END AS difficulty_label,
                progress.status,
                progress.review_stage,
                progress.next_review_time,
                progress.wrong_count,
                progress.first_wrong_time,
                progress.last_wrong_time,
                last_wrong.cost_ms AS last_cost_ms
            FROM sentence_cloze_review_schedule progress
            JOIN sentence_cloze_item item ON item.id = progress.cloze_item_id
            LEFT JOIN sentence_cloze_answer_record last_wrong
              ON last_wrong.id = progress.last_wrong_answer_record_id
             AND last_wrong.user_id = #{userId}
            WHERE progress.id = #{progressId}
              AND progress.user_id = #{userId}
              AND item.user_id = #{userId}
            LIMIT 1
            """)
    ClozeWrongSentenceItem selectWrongSentenceById(
            @Param("userId") Long userId,
            @Param("progressId") Long progressId
    );

    @Select("""
            SELECT
                record.id AS record_id,
                record.is_correct AS correct,
                record.cost_ms,
                record.practice_context,
                record.action_type,
                record.wrong_blank_indexes_json,
                record.create_time AS answered_at
            FROM sentence_cloze_answer_record record
            WHERE record.user_id = #{userId}
              AND record.cloze_item_id = #{clozeItemId}
            ORDER BY record.create_time DESC, record.id DESC
            LIMIT #{limit}
            """)
    List<ClozeWrongSentenceAttempt> selectRecentAttempts(
            @Param("userId") Long userId,
            @Param("clozeItemId") Long clozeItemId,
            @Param("limit") Integer limit
    );

    @Select("""
            SELECT *
            FROM wrong_word_review_progress
            WHERE user_id = #{userId}
              AND active_cloze_item_id = #{clozeItemId}
            ORDER BY active_blank_index ASC, id ASC
            """)
    List<WrongWordReviewProgress> selectWordProgresses(
            @Param("userId") Long userId,
            @Param("clozeItemId") Long clozeItemId
    );
}
