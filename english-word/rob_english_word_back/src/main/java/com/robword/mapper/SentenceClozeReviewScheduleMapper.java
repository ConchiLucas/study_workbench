package com.robword.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.robword.entity.SentenceClozeReviewSchedule;
import org.apache.ibatis.annotations.Delete;
import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Update;

import java.time.LocalDateTime;

@Mapper
public interface SentenceClozeReviewScheduleMapper extends BaseMapper<SentenceClozeReviewSchedule> {

    @Insert("""
            INSERT INTO sentence_cloze_review_schedule (
                user_id,
                cloze_item_id,
                correct_streak,
                review_stage,
                status,
                next_review_time,
                wrong_count,
                first_wrong_time,
                last_answer_record_id,
                last_wrong_answer_record_id,
                last_wrong_time
            ) SELECT
                #{userId},
                #{clozeItemId},
                0,
                0,
                'active',
                CURRENT_TIMESTAMP,
                1,
                CURRENT_TIMESTAMP,
                #{recordId},
                #{recordId},
                CURRENT_TIMESTAMP
            WHERE NOT EXISTS (
                SELECT 1
                FROM sentence_cloze_answer_record newer
                WHERE newer.user_id = #{userId}
                  AND newer.cloze_item_id = #{clozeItemId}
                  AND newer.id > #{recordId}
            )
            ON CONFLICT (user_id, cloze_item_id)
            DO UPDATE SET
                correct_streak = 0,
                review_stage = 0,
                status = 'active',
                next_review_time = EXCLUDED.next_review_time,
                wrong_count = sentence_cloze_review_schedule.wrong_count + 1,
                first_wrong_time = COALESCE(
                    sentence_cloze_review_schedule.first_wrong_time,
                    EXCLUDED.first_wrong_time
                ),
                last_answer_record_id = EXCLUDED.last_answer_record_id,
                last_wrong_answer_record_id = EXCLUDED.last_wrong_answer_record_id,
                last_wrong_time = EXCLUDED.last_wrong_time,
                last_correct_time = NULL,
                completed_time = NULL
            WHERE (
                sentence_cloze_review_schedule.last_answer_record_id IS NULL
                OR EXCLUDED.last_answer_record_id > sentence_cloze_review_schedule.last_answer_record_id
            )
              AND NOT EXISTS (
                  SELECT 1
                  FROM sentence_cloze_answer_record newer
                  WHERE newer.user_id = #{userId}
                    AND newer.cloze_item_id = #{clozeItemId}
                    AND newer.id > #{recordId}
              )
            """)
    void upsertWrongSchedule(
            @Param("userId") Long userId,
            @Param("clozeItemId") Long clozeItemId,
            @Param("recordId") Long recordId,
            @Param("answeredAt") LocalDateTime answeredAt
    );

    @Update("""
            UPDATE sentence_cloze_review_schedule
            SET correct_streak = CASE
                    WHEN review_stage >= 2 THEN 3
                    ELSE review_stage + 1
                END,
                review_stage = CASE
                    WHEN review_stage >= 2 THEN 3
                    ELSE review_stage + 1
                END,
                status = CASE
                    WHEN review_stage >= 2 THEN 'completed'
                    ELSE 'active'
                END,
                next_review_time = CASE
                    WHEN review_stage = 0 THEN CURRENT_TIMESTAMP + INTERVAL '7 days'
                    WHEN review_stage = 1 THEN CURRENT_TIMESTAMP + INTERVAL '15 days'
                    ELSE NULL
                END,
                last_answer_record_id = #{recordId},
                last_correct_time = CURRENT_TIMESTAMP,
                completed_time = CASE
                    WHEN review_stage >= 2 THEN CURRENT_TIMESTAMP
                    ELSE NULL
                END
            WHERE user_id = #{userId}
              AND cloze_item_id = #{clozeItemId}
              AND status = 'active'
              AND next_review_time <= CURRENT_TIMESTAMP
              AND (
                  last_answer_record_id IS NULL
                  OR last_answer_record_id <> #{recordId}
              )
            """)
    int advanceDueCorrectSchedule(
            @Param("userId") Long userId,
            @Param("clozeItemId") Long clozeItemId,
            @Param("recordId") Long recordId,
            @Param("answeredAt") LocalDateTime answeredAt,
            @Param("sevenDaysAt") LocalDateTime sevenDaysAt,
            @Param("fifteenDaysAt") LocalDateTime fifteenDaysAt
    );

    @Update("""
            UPDATE sentence_cloze_review_schedule
            SET correct_streak = #{correctStreak},
                review_stage = #{reviewStage},
                next_review_time = #{nextReviewTime},
                last_answer_record_id = #{recordId},
                last_correct_time = NOW()
            WHERE user_id = #{userId}
              AND cloze_item_id = #{clozeItemId}
            """)
    void advanceCorrectSchedule(
            @Param("userId") Long userId,
            @Param("clozeItemId") Long clozeItemId,
            @Param("recordId") Long recordId,
            @Param("correctStreak") Integer correctStreak,
            @Param("reviewStage") Integer reviewStage,
            @Param("nextReviewTime") LocalDateTime nextReviewTime
    );

    @Delete("""
            DELETE FROM sentence_cloze_review_schedule
            WHERE user_id = #{userId}
              AND cloze_item_id = #{clozeItemId}
            """)
    void deleteByUserAndItem(@Param("userId") Long userId, @Param("clozeItemId") Long clozeItemId);
}
