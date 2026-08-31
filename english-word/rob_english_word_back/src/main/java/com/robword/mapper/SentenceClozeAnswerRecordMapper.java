package com.robword.mapper;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.robword.dto.ClozePracticeHistoryItem;
import com.robword.entity.SentenceClozeAnswerRecord;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Insert;
import org.apache.ibatis.annotations.Options;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;

import java.util.List;

@Mapper
public interface SentenceClozeAnswerRecordMapper extends BaseMapper<SentenceClozeAnswerRecord> {

    @Select("""
            SELECT *
            FROM sentence_cloze_answer_record
            WHERE user_id = #{userId}
              AND submission_key = #{submissionKey}
            LIMIT 1
            """)
    SentenceClozeAnswerRecord selectBySubmissionKey(
            @Param("userId") Long userId,
            @Param("submissionKey") String submissionKey
    );

    @Insert("""
            INSERT INTO sentence_cloze_answer_record (
                user_id,
                user_name,
                cloze_item_id,
                answer_text,
                answers_json,
                expected_words_json,
                submission_key,
                practice_context,
                action_type,
                wrong_blank_indexes_json,
                is_correct,
                attempt_no,
                cost_ms
            ) VALUES (
                #{userId},
                #{userName},
                #{clozeItemId},
                #{answerText},
                #{answersJson},
                #{expectedWordsJson},
                #{submissionKey},
                #{practiceContext},
                #{actionType},
                #{wrongBlankIndexesJson},
                #{isCorrect},
                #{attemptNo},
                #{costMs}
            )
            ON CONFLICT (user_id, submission_key)
                WHERE submission_key IS NOT NULL
            DO NOTHING
            """)
    @Options(useGeneratedKeys = true, keyProperty = "id", keyColumn = "id")
    int insertIdempotent(SentenceClozeAnswerRecord record);

    @Select("""
            SELECT COUNT(*)
            FROM sentence_cloze_item i
            JOIN LATERAL (
                SELECT r.is_correct
                FROM sentence_cloze_answer_record r
                WHERE r.user_id = #{userId}
                  AND r.cloze_item_id = i.id
                ORDER BY r.create_time DESC, r.id DESC
                LIMIT 1
            ) latest ON true
            LEFT JOIN sentence_cloze_review_schedule review
              ON review.user_id = #{userId}
             AND review.cloze_item_id = i.id
            WHERE i.user_id = #{userId}
              AND latest.is_correct = true
              AND (review.id IS NULL OR review.status = 'completed')
            """)
    Long countCompletedItems(@Param("userId") Long userId);

    @Select("""
            SELECT
                r.id,
                r.cloze_item_id,
                i.cloze_sentence,
                i.translation_zh,
                r.answer_text,
                r.expected_words_json,
                r.is_correct,
                r.attempt_no,
                r.cost_ms,
                r.create_time
            FROM sentence_cloze_answer_record r
            LEFT JOIN sentence_cloze_item i ON i.id = r.cloze_item_id
            WHERE r.user_id = #{userId}
            ORDER BY r.create_time DESC, r.id DESC
            LIMIT #{limit}
            """)
    List<ClozePracticeHistoryItem> selectHistory(@Param("userId") Long userId, @Param("limit") Integer limit);
}
