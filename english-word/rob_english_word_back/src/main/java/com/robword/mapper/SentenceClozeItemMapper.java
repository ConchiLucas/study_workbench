package com.robword.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.robword.dto.ClozePracticeSentenceCandidate;
import com.robword.entity.SentenceClozeItem;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;

import java.util.List;

@Mapper
public interface SentenceClozeItemMapper extends BaseMapper<SentenceClozeItem> {

    @Select("""
            SELECT *
            FROM sentence_cloze_item
            WHERE id = #{id}
              AND user_id = #{userId}
            FOR UPDATE
            """)
    SentenceClozeItem selectOwnedByIdForUpdate(
            @Param("id") Long id,
            @Param("userId") Long userId
    );

    @Select("""
            SELECT *
            FROM sentence_cloze_item
            WHERE generation_key = #{generationKey}
            LIMIT 1
            """)
    SentenceClozeItem selectByGenerationKey(@Param("generationKey") String generationKey);

    @Select("""
            SELECT
                i.*,
                latest.is_correct AS latest_answer_correct,
                latest.create_time AS latest_answer_time,
                progress.next_review_time AS next_review_time
            FROM sentence_cloze_item i
            JOIN LATERAL (
                SELECT MIN(p.next_review_time) AS next_review_time
                FROM wrong_word_review_progress p
                WHERE p.user_id = #{userId}
                  AND p.active_cloze_item_id = i.id
                  AND p.status <> 'completed'
                  AND p.next_review_time <= NOW()
            ) progress ON progress.next_review_time IS NOT NULL
            LEFT JOIN LATERAL (
                SELECT r.is_correct, r.create_time
                FROM sentence_cloze_answer_record r
                WHERE r.user_id = #{userId}
                  AND r.cloze_item_id = i.id
                ORDER BY r.create_time DESC, r.id DESC
                LIMIT 1
            ) latest ON true
            WHERE i.user_id = #{userId}
              AND i.source = 'word-agent'
            ORDER BY progress.next_review_time ASC, i.id ASC
            LIMIT 1
            """)
    SentenceClozeItem selectNextPracticeItem(@Param("userId") Long userId);

    @Select("""
            SELECT
                i.*,
                latest.is_correct AS latest_answer_correct,
                latest.create_time AS latest_answer_time,
                progress.next_review_time AS next_review_time
            FROM sentence_cloze_item i
            JOIN LATERAL (
                SELECT MIN(p.next_review_time) AS next_review_time
                FROM wrong_word_review_progress p
                WHERE p.user_id = #{userId}
                  AND p.active_cloze_item_id = i.id
                  AND p.status <> 'completed'
                  AND p.next_review_time <= NOW()
            ) progress ON progress.next_review_time IS NOT NULL
            LEFT JOIN LATERAL (
                SELECT r.is_correct, r.create_time
                FROM sentence_cloze_answer_record r
                WHERE r.user_id = #{userId}
                  AND r.cloze_item_id = i.id
                ORDER BY r.create_time DESC, r.id DESC
                LIMIT 1
            ) latest ON true
            WHERE i.user_id = #{userId}
              AND i.source = 'word-agent'
            ORDER BY progress.next_review_time ASC, i.id ASC
            LIMIT #{limit}
            """)
    List<SentenceClozeItem> selectNextPracticeItems(@Param("userId") Long userId, @Param("limit") Integer limit);

    @Select("""
            SELECT
                i.*,
                latest.is_correct AS latest_answer_correct,
                latest.create_time AS latest_answer_time,
                progress.next_review_time AS next_review_time
            FROM sentence_cloze_item i
            JOIN LATERAL (
                SELECT MIN(p.next_review_time) AS next_review_time
                FROM wrong_word_review_progress p
                WHERE p.user_id = #{userId}
                  AND p.active_cloze_item_id = i.id
                  AND p.status <> 'completed'
                  AND p.next_review_time <= NOW()
            ) progress ON progress.next_review_time IS NOT NULL
            LEFT JOIN LATERAL (
                SELECT r.is_correct, r.create_time
                FROM sentence_cloze_answer_record r
                WHERE r.user_id = #{userId}
                  AND r.cloze_item_id = i.id
                ORDER BY r.create_time DESC, r.id DESC
                LIMIT 1
            ) latest ON true
            WHERE i.user_id = #{userId}
              AND i.source = 'word-agent'
            ORDER BY progress.next_review_time ASC, i.id ASC
            LIMIT #{limit}
            """)
    List<SentenceClozeItem> selectPendingPracticeItems(@Param("userId") Long userId, @Param("limit") Integer limit);

    @Select("""
            SELECT
                i.*,
                latest.is_correct AS latest_answer_correct,
                latest.create_time AS latest_answer_time,
                progress.next_review_time AS next_review_time
            FROM sentence_cloze_item i
            JOIN LATERAL (
                SELECT MIN(p.next_review_time) AS next_review_time
                FROM wrong_word_review_progress p
                WHERE p.user_id = #{userId}
                  AND p.active_cloze_item_id = i.id
                  AND p.status <> 'completed'
            ) progress ON progress.next_review_time IS NOT NULL
            LEFT JOIN LATERAL (
                SELECT r.is_correct, r.create_time
                FROM sentence_cloze_answer_record r
                WHERE r.user_id = #{userId}
                  AND r.cloze_item_id = i.id
                ORDER BY r.create_time DESC, r.id DESC
                LIMIT 1
            ) latest ON true
            WHERE i.user_id = #{userId}
              AND #{state} = 'wrong'
            ORDER BY progress.next_review_time ASC NULLS FIRST, latest.create_time DESC NULLS LAST, i.id DESC
            LIMIT #{limit}
            """)
    List<SentenceClozeItem> selectWrongPracticeItems(
            @Param("userId") Long userId,
            @Param("state") String state,
            @Param("limit") Integer limit
    );

    @Select("""
            WITH sentence_due AS (
                SELECT
                    s.cloze_item_id,
                    s.next_review_time
                FROM sentence_cloze_review_schedule s
                WHERE s.user_id = #{userId}
                  AND s.status = 'active'
                  AND s.next_review_time <= NOW()
            ),
            word_due AS (
                SELECT
                    p.active_cloze_item_id AS cloze_item_id,
                    MIN(p.next_review_time) AS next_review_time
                FROM wrong_word_review_progress p
                WHERE p.user_id = #{userId}
                  AND p.active_cloze_item_id IS NOT NULL
                  AND p.status <> 'completed'
                  AND p.next_review_time <= NOW()
                GROUP BY p.active_cloze_item_id
            ),
            combined_due AS (
                SELECT cloze_item_id, next_review_time FROM sentence_due
                UNION ALL
                SELECT cloze_item_id, next_review_time FROM word_due
            ),
            deduped_due AS (
                SELECT cloze_item_id, MIN(next_review_time) AS next_review_time
                FROM combined_due
                GROUP BY cloze_item_id
            )
            SELECT COUNT(*)
            FROM deduped_due due
            JOIN sentence_cloze_item i ON i.id = due.cloze_item_id
            WHERE i.user_id = #{userId}
            """)
    Long countDueReviewItems(@Param("userId") Long userId);

    @Select("""
            SELECT COUNT(*)
            FROM sentence_cloze_review_schedule s
            JOIN sentence_cloze_item i ON i.id = s.cloze_item_id
            WHERE s.user_id = #{userId}
              AND i.user_id = #{userId}
              AND s.status = 'active'
            """)
    Long countActiveWrongSentences(@Param("userId") Long userId);

    @Select("""
            WITH sentence_due AS (
                SELECT s.cloze_item_id, s.next_review_time
                FROM sentence_cloze_review_schedule s
                WHERE s.user_id = #{userId}
                  AND s.status = 'active'
                  AND s.next_review_time <= NOW()
            ),
            word_due AS (
                SELECT
                    p.active_cloze_item_id AS cloze_item_id,
                    MIN(p.next_review_time) AS next_review_time
                FROM wrong_word_review_progress p
                WHERE p.user_id = #{userId}
                  AND p.active_cloze_item_id IS NOT NULL
                  AND p.status <> 'completed'
                  AND p.next_review_time <= NOW()
                GROUP BY p.active_cloze_item_id
            ),
            combined_due AS (
                SELECT cloze_item_id, next_review_time FROM sentence_due
                UNION ALL
                SELECT cloze_item_id, next_review_time FROM word_due
            ),
            deduped_due AS (
                SELECT cloze_item_id, MIN(next_review_time) AS next_review_time
                FROM combined_due
                GROUP BY cloze_item_id
            )
            SELECT
                i.*,
                latest.is_correct AS latest_answer_correct,
                latest.create_time AS latest_answer_time,
                due.next_review_time AS next_review_time
            FROM deduped_due due
            JOIN sentence_cloze_item i ON i.id = due.cloze_item_id
            LEFT JOIN LATERAL (
                SELECT r.is_correct, r.create_time
                FROM sentence_cloze_answer_record r
                WHERE r.user_id = #{userId}
                  AND r.cloze_item_id = i.id
                ORDER BY r.create_time DESC, r.id DESC
                LIMIT 1
            ) latest ON true
            WHERE i.user_id = #{userId}
            ORDER BY due.next_review_time ASC, i.id ASC
            LIMIT #{limit}
            """)
    List<SentenceClozeItem> selectDueWrongReviewItems(
            @Param("userId") Long userId,
            @Param("limit") Integer limit
    );

    @Select("""
            SELECT
                i.*,
                latest.is_correct AS latest_answer_correct,
                latest.create_time AS latest_answer_time,
                NULL::timestamp AS next_review_time
            FROM sentence_cloze_item i
            JOIN LATERAL (
                SELECT COUNT(*) AS word_count
                FROM wrong_word_review_progress p
                WHERE p.user_id = #{userId}
                  AND p.active_cloze_item_id = i.id
                HAVING COUNT(*) > 0
                   AND BOOL_AND(p.status = 'completed')
            ) progress ON true
            JOIN LATERAL (
                SELECT r.is_correct, r.create_time
                FROM sentence_cloze_answer_record r
                WHERE r.user_id = #{userId}
                  AND r.cloze_item_id = i.id
                ORDER BY r.create_time DESC, r.id DESC
                LIMIT 1
            ) latest ON true
            WHERE i.user_id = #{userId}
              AND #{state} = 'mastered'
              AND i.source = 'word-agent'
            ORDER BY latest.create_time DESC, i.id DESC
            LIMIT #{limit}
            """)
    List<SentenceClozeItem> selectMasteredPracticeItems(
            @Param("userId") Long userId,
            @Param("state") String state,
            @Param("limit") Integer limit
    );

    @Select("""
            SELECT
                i.*,
                latest.is_correct AS latest_answer_correct,
                latest.create_time AS latest_answer_time,
                review.next_review_time AS next_review_time
            FROM sentence_cloze_item i
            LEFT JOIN sentence_cloze_review_schedule review
              ON review.user_id = #{userId}
             AND review.cloze_item_id = i.id
            LEFT JOIN LATERAL (
                SELECT r.is_correct, r.create_time
                FROM sentence_cloze_answer_record r
                WHERE r.user_id = #{userId}
                  AND r.cloze_item_id = i.id
                ORDER BY r.create_time DESC, r.id DESC
                LIMIT 1
            ) latest ON true
            WHERE i.user_id = #{userId}
              AND i.source = 'best-sentence-practice'
              AND i.provider_label = #{providerLabel}
              AND (
                  latest.is_correct IS NULL
                  OR (review.id IS NOT NULL AND review.next_review_time <= NOW())
              )
            ORDER BY COALESCE(review.next_review_time, i.create_time) ASC, i.id ASC
            LIMIT #{limit}
            """)
    List<SentenceClozeItem> selectPendingDifficultyPracticeItems(
            @Param("userId") Long userId,
            @Param("providerLabel") String providerLabel,
            @Param("limit") Integer limit
    );

    @Select("""
            <script>
            SELECT
                wc.id AS word_clean_id,
                wcbs.id AS best_sentence_id,
                wc.word,
                COALESCE(wc.meaning, '') AS meaning,
                wcbs.sentence,
                COALESCE(wcbs.sentence_translation, '') AS sentence_translation,
                wcbs.cloze_sentence,
                wcbs.cloze_answer,
                wcbs.source_model_name AS model_name,
                wcbs.tts_object_url
            FROM word_clean wc
            JOIN word_clean_best_sentence wcbs ON wcbs.word = wc.word
            WHERE wc.source_difficulty IN
              <foreach collection="sourceDifficulties" item="sourceDifficulty" open="(" separator="," close=")">
                #{sourceDifficulty}
              </foreach>
              AND COALESCE(wc.word, '') != ''
              AND COALESCE(wcbs.sentence, '') != ''
              AND COALESCE(wcbs.cloze_sentence, '') != ''
              AND COALESCE(wcbs.cloze_answer, '') != ''
              AND NOT EXISTS (
                  SELECT 1
                  FROM sentence_cloze_item i
                  WHERE i.user_id = #{userId}
                    AND i.source = 'best-sentence-practice'
                    AND i.best_sentence_id = wcbs.id
              )
            ORDER BY RANDOM()
            LIMIT #{limit}
            </script>
            """)
    List<ClozePracticeSentenceCandidate> selectDifficultyCandidates(
            @Param("userId") Long userId,
            @Param("sourceDifficulties") List<Integer> sourceDifficulties,
            @Param("limit") Integer limit
    );
}
