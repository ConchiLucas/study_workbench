package com.robword.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.robword.entity.WrongWordReviewProgress;
import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;
import org.apache.ibatis.annotations.Update;

import java.time.LocalDateTime;
import java.util.List;

@Mapper
public interface WrongWordReviewProgressMapper extends BaseMapper<WrongWordReviewProgress> {

    @Insert("""
            INSERT INTO wrong_word_review_progress (
                user_id,
                word_id,
                word,
                normalized_word,
                status,
                review_stage,
                next_review_time,
                wrong_count,
                first_wrong_time,
                last_wrong_time,
                last_answer_record_id
            ) VALUES (
                #{userId},
                #{wordId},
                #{word},
                #{normalizedWord},
                'waiting_sentence',
                0,
                NULL,
                1,
                #{wrongTime},
                #{wrongTime},
                #{answerRecordId}
            )
            ON CONFLICT (user_id, normalized_word)
            DO UPDATE SET
                word_id = COALESCE(EXCLUDED.word_id, wrong_word_review_progress.word_id),
                word = EXCLUDED.word,
                status = CASE
                    WHEN wrong_word_review_progress.active_cloze_item_id IS NULL
                        THEN 'waiting_sentence'
                    ELSE 'due'
                END,
                review_stage = 0,
                next_review_time = CASE
                    WHEN wrong_word_review_progress.active_cloze_item_id IS NULL
                        THEN NULL
                    ELSE EXCLUDED.last_wrong_time
                END,
                wrong_count = wrong_word_review_progress.wrong_count + 1,
                last_wrong_time = EXCLUDED.last_wrong_time,
                last_answer_record_id = EXCLUDED.last_answer_record_id,
                completed_time = NULL
            WHERE EXCLUDED.last_wrong_time >= wrong_word_review_progress.last_wrong_time
              AND (
                  EXCLUDED.last_wrong_time > wrong_word_review_progress.last_wrong_time
                  OR EXCLUDED.last_answer_record_id IS DISTINCT FROM
                     wrong_word_review_progress.last_answer_record_id
              )
            """)
    void upsertWrong(
            @Param("userId") Long userId,
            @Param("wordId") Long wordId,
            @Param("word") String word,
            @Param("normalizedWord") String normalizedWord,
            @Param("wrongTime") LocalDateTime wrongTime,
            @Param("answerRecordId") Long answerRecordId
    );

    @Update("""
            UPDATE wrong_word_review_progress
            SET active_cloze_item_id = #{clozeItemId},
                active_blank_index = #{blankIndex},
                status = 'due',
                review_stage = 0,
                next_review_time = #{dueTime},
                completed_time = NULL
            WHERE user_id = #{userId}
              AND normalized_word = #{normalizedWord}
              AND status <> 'completed'
              AND active_cloze_item_id IS NULL
            """)
    int linkActiveSentence(
            @Param("userId") Long userId,
            @Param("normalizedWord") String normalizedWord,
            @Param("clozeItemId") Long clozeItemId,
            @Param("blankIndex") Integer blankIndex,
            @Param("dueTime") LocalDateTime dueTime
    );

    @Update("""
            UPDATE wrong_word_review_progress
            SET review_stage = CASE
                    WHEN review_stage = 2 THEN 3
                    ELSE review_stage + 1
                END,
                status = CASE
                    WHEN review_stage = 2 THEN 'completed'
                    ELSE 'waiting'
                END,
                next_review_time = CASE
                    WHEN review_stage = 0 THEN #{sevenDaysAt}
                    WHEN review_stage = 1 THEN #{fifteenDaysAt}
                    ELSE NULL
                END,
                completed_time = CASE
                    WHEN review_stage = 2 THEN #{answeredAt}
                    ELSE NULL
                END
            WHERE user_id = #{userId}
              AND active_cloze_item_id = #{clozeItemId}
              AND active_blank_index = #{blankIndex}
              AND status <> 'completed'
              AND next_review_time <= #{answeredAt}
            """)
    int advanceDueCorrect(
            @Param("userId") Long userId,
            @Param("clozeItemId") Long clozeItemId,
            @Param("blankIndex") Integer blankIndex,
            @Param("answerRecordId") Long answerRecordId,
            @Param("answeredAt") LocalDateTime answeredAt,
            @Param("sevenDaysAt") LocalDateTime sevenDaysAt,
            @Param("fifteenDaysAt") LocalDateTime fifteenDaysAt
    );

    @Select("""
            SELECT *
            FROM wrong_word_review_progress
            WHERE user_id = #{userId}
              AND active_cloze_item_id = #{clozeItemId}
              AND status <> 'completed'
            ORDER BY active_blank_index ASC, id ASC
            """)
    List<WrongWordReviewProgress> selectByActiveItem(
            @Param("userId") Long userId,
            @Param("clozeItemId") Long clozeItemId
    );

    @Select("""
            SELECT *
            FROM wrong_word_review_progress
            WHERE user_id = #{userId}
              AND normalized_word = #{normalizedWord}
            LIMIT 1
            """)
    WrongWordReviewProgress selectByUserAndNormalizedWord(
            @Param("userId") Long userId,
            @Param("normalizedWord") String normalizedWord
    );
}
