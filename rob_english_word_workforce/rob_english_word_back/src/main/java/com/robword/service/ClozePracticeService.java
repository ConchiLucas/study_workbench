package com.robword.service;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.robword.dto.ClozePracticeAnswerRequest;
import com.robword.dto.ClozePracticeAnswerResponse;
import com.robword.dto.ClozePracticeDifficultyBatchRequest;
import com.robword.dto.ClozePracticeHistoryItem;
import com.robword.dto.ClozePracticePreferenceResponse;
import com.robword.dto.ClozePracticeSentenceCandidate;
import com.robword.dto.ClozePracticeStatsResponse;
import com.robword.dto.ClozePracticeTaskResponse;
import com.robword.dto.ClozeWrongSentenceAttempt;
import com.robword.dto.ClozeWrongSentenceDetail;
import com.robword.dto.ClozeWrongSentenceItem;
import com.robword.dto.ClozeWrongSentencePageResponse;
import com.robword.dto.UpdateSoloDifficultyRequest;
import com.robword.entity.SentenceClozeAnswerRecord;
import com.robword.entity.SentenceClozeItem;
import com.robword.entity.User;
import com.robword.entity.Word;
import com.robword.entity.WordCleanTts;
import com.robword.entity.WrongWordReviewProgress;
import com.robword.mapper.ClozeWrongSentenceQueryMapper;
import com.robword.mapper.SentenceClozeAnswerRecordMapper;
import com.robword.mapper.SentenceClozeItemMapper;
import com.robword.mapper.SentenceClozeReviewScheduleMapper;
import com.robword.mapper.UserMapper;
import com.robword.mapper.WordMapper;
import com.robword.mapper.WordCleanTtsMapper;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.text.Normalizer;
import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Set;
import java.util.stream.Collectors;

@Service
@RequiredArgsConstructor
public class ClozePracticeService {

    private static final int DEFAULT_HISTORY_LIMIT = 20;
    private static final int MAX_HISTORY_LIMIT = 100;
    private static final int DEFAULT_BATCH_LIMIT = 10;
    private static final int MAX_BATCH_LIMIT = 20;
    private static final String DEFAULT_SOLO_DIFFICULTY_GROUP = "junior";
    private static final String DEFAULT_SOLO_DIFFICULTY_LEVEL = "junior";
    private static final Map<String, Integer> DIFFICULTY_SOURCE_MAP = buildDifficultySourceMap();
    private static final Map<String, List<Integer>> DIFFICULTY_GROUP_SOURCE_MAP = buildDifficultyGroupSourceMap();
    private static final Map<String, String> DIFFICULTY_GROUP_LABELS = Map.of(
            "rank", "段位难度",
            "primary", "小学英语",
            "junior", "初中英语",
            "senior", "高中英语",
            "college", "大学英语",
            "entrance", "升学考试英语",
            "business_abroad", "商务与出国英语",
            "professional", "专业英语",
            "advanced_exam", "高阶考试英语"
    );
    private static final Map<String, String> DIFFICULTY_LEVEL_LABELS = buildDifficultyLevelLabels();

    private final SentenceClozeItemMapper sentenceClozeItemMapper;
    private final SentenceClozeAnswerRecordMapper answerRecordMapper;
    private final SentenceClozeReviewScheduleMapper reviewScheduleMapper;
    private final UserMapper userMapper;
    private final WordMapper wordMapper;
    private final WordCleanTtsMapper wordCleanTtsMapper;
    private final ObjectMapper objectMapper;
    private final WrongWordAgentNotificationService wrongWordAgentNotificationService;
    private final UserWordMasteryService userWordMasteryService;
    private final WrongWordReviewProgressService wrongWordReviewProgressService;
    private final ClozeWrongSentenceQueryMapper wrongSentenceQueryMapper;

    public ClozePracticeTaskResponse getNextTask(Long userId) {
        requireUserId(userId);
        SentenceClozeItem item = sentenceClozeItemMapper.selectNextPracticeItem(userId);
        return item == null ? null : toTaskResponse(userId, item);
    }

    public ClozePracticePreferenceResponse getPreferences(Long userId) {
        User user = requireUser(userId);
        String group = normalizeString(user.getSoloDifficultyGroup());
        String level = normalizeString(user.getSoloDifficultyLevel());
        if (!isValidSoloDifficulty(group, level)) {
            group = DEFAULT_SOLO_DIFFICULTY_GROUP;
            level = DEFAULT_SOLO_DIFFICULTY_LEVEL;
            persistSoloDifficulty(user.getId(), group, level);
        }
        return new ClozePracticePreferenceResponse(group, level);
    }

    public ClozePracticePreferenceResponse updateSoloDifficulty(Long userId, UpdateSoloDifficultyRequest request) {
        String group = request == null ? "" : normalizeString(request.getDifficultyGroup());
        String level = request == null ? "" : normalizeString(request.getDifficultyLevel());
        if (!isValidSoloDifficulty(group, level)) {
            throw new IllegalArgumentException("不支持的单独训练难度");
        }
        User user = requireUser(userId);
        persistSoloDifficulty(user.getId(), group, level);
        return new ClozePracticePreferenceResponse(group, level);
    }

    private void persistSoloDifficulty(Long userId, String group, String level) {
        User update = new User();
        update.setId(userId);
        update.setSoloDifficultyGroup(group);
        update.setSoloDifficultyLevel(level);
        userMapper.updateById(update);
    }

    private User requireUser(Long userId) {
        requireUserId(userId);
        User user = userMapper.selectById(userId);
        if (user == null) {
            throw new IllegalArgumentException("用户不存在");
        }
        return user;
    }

    private boolean isValidSoloDifficulty(String group, String level) {
        List<Integer> groupSources = DIFFICULTY_GROUP_SOURCE_MAP.get(group);
        if (groupSources == null || group.isBlank() || level.isBlank()) {
            return false;
        }
        if (group.equals(level)) {
            return true;
        }
        Integer source = DIFFICULTY_SOURCE_MAP.get(level);
        return source != null && groupSources.contains(source);
    }

    public List<ClozePracticeTaskResponse> createDifficultyBatch(Long userId, ClozePracticeDifficultyBatchRequest request) {
        requireUserId(userId);
        int limit = normalizeBatchLimit(request == null ? null : request.getLimit());
        String difficultyLevel = request == null ? "" : normalizeString(request.getDifficultyLevel());
        String difficultyGroup = request == null ? "" : normalizeString(request.getDifficultyGroup());
        if (difficultyLevel.isBlank() || "rank_current".equals(difficultyLevel)) {
            List<SentenceClozeItem> items = sentenceClozeItemMapper.selectNextPracticeItems(userId, limit);
            return toTaskResponses(
                    userId,
                    items,
                    difficultyGroup,
                    difficultyLevel,
                    difficultyLabel(difficultyGroup, difficultyLevel)
            );
        }

        List<Integer> sourceDifficulties = sourceDifficultiesForLevel(difficultyLevel);
        if (sourceDifficulties.isEmpty()) {
            throw new IllegalArgumentException("不支持的难度：" + difficultyLevel);
        }

        List<ClozePracticeSentenceCandidate> candidates =
                sentenceClozeItemMapper.selectDifficultyCandidates(userId, sourceDifficulties, limit);
        List<SentenceClozeItem> createdItems = new ArrayList<>();
        for (ClozePracticeSentenceCandidate candidate : candidates) {
            SentenceClozeItem item = buildDifficultyPracticeItem(userId, difficultyGroup, difficultyLevel, candidate);
            sentenceClozeItemMapper.insert(item);
            createdItems.add(item);
        }

        if (!createdItems.isEmpty()) {
            String label = difficultyLabel(difficultyGroup, difficultyLevel);
            return toTaskResponses(userId, createdItems, difficultyGroup, difficultyLevel, label);
        }

        String label = difficultyLabel(difficultyGroup, difficultyLevel);
        List<SentenceClozeItem> items = sentenceClozeItemMapper.selectPendingDifficultyPracticeItems(userId, label, limit);
        return toTaskResponses(userId, items, difficultyGroup, difficultyLevel, label);
    }

    public List<ClozePracticeTaskResponse> getPendingTasks(Long userId, Integer limit) {
        requireUserId(userId);
        int normalizedLimit = limit == null ? MAX_HISTORY_LIMIT : Math.max(1, Math.min(limit, MAX_HISTORY_LIMIT));
        List<SentenceClozeItem> items = sentenceClozeItemMapper.selectPendingPracticeItems(userId, normalizedLimit);
        return toTaskResponses(userId, items, null, null, null);
    }

    public List<ClozePracticeTaskResponse> getDueReviewTasks(Long userId, Integer limit) {
        requireUserId(userId);
        int normalizedLimit = limit == null ? DEFAULT_BATCH_LIMIT : Math.max(1, Math.min(limit, MAX_BATCH_LIMIT));
        List<SentenceClozeItem> items = sentenceClozeItemMapper.selectDueWrongReviewItems(userId, normalizedLimit);
        return toTaskResponses(userId, items, null, null, null);
    }

    public List<ClozePracticeTaskResponse> getAnsweredTasks(Long userId, String status, Integer limit) {
        requireUserId(userId);
        int normalizedLimit = limit == null ? MAX_HISTORY_LIMIT : Math.max(1, Math.min(limit, MAX_HISTORY_LIMIT));
        String state = switch (normalizeString(status).toLowerCase(Locale.ROOT)) {
            case "mastered", "known", "correct" -> "mastered";
            case "wrong", "incorrect" -> "wrong";
            case "review", "scheduled", "schedule" -> "review";
            default -> throw new IllegalArgumentException("不支持的句子状态：" + status);
        };
        List<SentenceClozeItem> items = "mastered".equals(state)
                ? sentenceClozeItemMapper.selectMasteredPracticeItems(userId, state, normalizedLimit)
                : sentenceClozeItemMapper.selectWrongPracticeItems(userId, "wrong", normalizedLimit);
        return toTaskResponses(userId, items, null, null, null);
    }

    private static Map<String, Integer> buildDifficultySourceMap() {
        Map<String, Integer> map = new LinkedHashMap<>();
        map.put("primary_3_1", 1);
        map.put("primary_3_2", 2);
        map.put("primary_4_1", 3);
        map.put("primary_4_2", 4);
        map.put("primary_5_1", 5);
        map.put("primary_5_2", 6);
        map.put("primary_6_1", 7);
        map.put("primary_6_2", 8);
        map.put("junior_7_1", 9);
        map.put("junior_7_2", 10);
        map.put("junior_8_1", 11);
        map.put("junior_8_2", 12);
        map.put("junior_9_1", 13);
        for (int index = 1; index <= 11; index++) {
            map.put("senior_" + index, 13 + index);
        }
        map.put("college_cet4", 25);
        map.put("entrance_kaoyan", 26);
        map.put("business_bec", 27);
        map.put("college_cet6", 28);
        map.put("business_ielts", 29);
        map.put("professional_tem4", 30);
        map.put("professional_tem8", 31);
        map.put("business_toefl", 32);
        map.put("business_gmat", 33);
        map.put("advanced_sat", 34);
        map.put("advanced_gre", 35);
        return Map.copyOf(map);
    }

    private static Map<String, List<Integer>> buildDifficultyGroupSourceMap() {
        Map<String, List<Integer>> map = new LinkedHashMap<>();
        map.put("primary", List.of(1, 2, 3, 4, 5, 6, 7, 8));
        map.put("junior", List.of(9, 10, 11, 12, 13));
        map.put("senior", List.of(14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24));
        map.put("college", List.of(25, 28));
        map.put("entrance", List.of(26));
        map.put("business_abroad", List.of(27, 29, 32, 33));
        map.put("professional", List.of(30, 31));
        map.put("advanced_exam", List.of(34, 35));
        return Map.copyOf(map);
    }

    private static Map<String, String> buildDifficultyLevelLabels() {
        Map<String, String> map = new LinkedHashMap<>();
        map.put("rank_current", "段位难度");
        map.put("primary", "小学英语");
        map.put("primary_3_1", "3年级上册");
        map.put("primary_3_2", "3年级下册");
        map.put("primary_4_1", "4年级上册");
        map.put("primary_4_2", "4年级下册");
        map.put("primary_5_1", "5年级上册");
        map.put("primary_5_2", "5年级下册");
        map.put("primary_6_1", "6年级上册");
        map.put("primary_6_2", "6年级下册");
        map.put("junior", "初中英语");
        map.put("junior_7_1", "7年级上册");
        map.put("junior_7_2", "7年级下册");
        map.put("junior_8_1", "8年级上册");
        map.put("junior_8_2", "8年级下册");
        map.put("junior_9_1", "9年级全册");
        map.put("senior", "高中英语");
        for (int index = 1; index <= 11; index++) {
            map.put("senior_" + index, "高中第" + index + "册");
        }
        map.put("college", "大学英语");
        map.put("college_cet4", "四级");
        map.put("college_cet6", "六级");
        map.put("entrance", "升学考试英语");
        map.put("entrance_kaoyan", "考研英语");
        map.put("business_abroad", "商务与出国英语");
        map.put("business_bec", "商务英语 BEC");
        map.put("business_ielts", "雅思");
        map.put("business_toefl", "托福");
        map.put("business_gmat", "GMAT");
        map.put("professional", "专业英语");
        map.put("professional_tem4", "专四");
        map.put("professional_tem8", "专八");
        map.put("advanced_exam", "高阶考试英语");
        map.put("advanced_sat", "SAT");
        map.put("advanced_gre", "GRE");
        return Map.copyOf(map);
    }

    private int normalizeBatchLimit(Integer limit) {
        return limit == null ? DEFAULT_BATCH_LIMIT : Math.max(1, Math.min(limit, MAX_BATCH_LIMIT));
    }

    private String normalizeString(String value) {
        return value == null ? "" : value.trim();
    }

    private List<Integer> sourceDifficultiesForLevel(String difficultyLevel) {
        Integer sourceDifficulty = DIFFICULTY_SOURCE_MAP.get(difficultyLevel);
        if (sourceDifficulty != null) {
            return List.of(sourceDifficulty);
        }
        return DIFFICULTY_GROUP_SOURCE_MAP.getOrDefault(difficultyLevel, List.of());
    }

    private String difficultyLabel(String difficultyGroup, String difficultyLevel) {
        String groupLabel = DIFFICULTY_GROUP_LABELS.getOrDefault(difficultyGroup, "");
        String levelLabel = DIFFICULTY_LEVEL_LABELS.getOrDefault(difficultyLevel, difficultyLevel);
        if (groupLabel.isBlank() || groupLabel.equals(levelLabel)) {
            return levelLabel;
        }
        return groupLabel + " · " + levelLabel;
    }

    @Transactional
    public ClozePracticeAnswerResponse submitAnswer(Long userId, ClozePracticeAnswerRequest request) {
        requireUserId(userId);
        if (request == null || request.getClozeItemId() == null) {
            throw new IllegalArgumentException("clozeItemId 不能为空");
        }

        SentenceClozeItem item = sentenceClozeItemMapper.selectOwnedByIdForUpdate(
                request.getClozeItemId(), userId);
        if (item == null || item.getUserId() == null || !item.getUserId().equals(userId)) {
            throw new IllegalArgumentException("挖空题不存在或不属于当前用户");
        }

        String submissionKey = normalizeSubmissionKey(request.getSubmissionKey());
        String practiceContext = normalizeSubmissionMetadata(
                request.getPracticeContext(),
                "practiceContext",
                List.of("review", "solo"),
                "word-agent".equals(item.getSource()) ? "review" : "solo"
        );
        String actionType = normalizeSubmissionMetadata(
                request.getActionType(),
                "actionType",
                List.of("answer", "reveal"),
                "answer"
        );
        if (submissionKey != null) {
            SentenceClozeAnswerRecord replay = answerRecordMapper.selectBySubmissionKey(userId, submissionKey);
            if (replay != null) {
                return replaySubmission(request.getClozeItemId(), replay);
            }
        }

        List<String> expectedWords = parseWordsJson(item.getBlankWordsJson());
        if (expectedWords.isEmpty()) {
            throw new IllegalArgumentException("挖空题缺少标准答案");
        }

        List<String> answers = normalizeAnswers(request);
        ClozeAnswerComparison comparison = "reveal".equals(actionType)
                ? ClozeAnswerComparison.reveal(answers, expectedWords)
                : ClozeAnswerComparison.compare(answers, expectedWords);
        answers = comparison.answers();
        expectedWords = comparison.expectedWords();
        int attemptNo = countAttempts(userId, item.getId()) + 1;
        String answerText = buildAnswerText(request.getAnswerText(), answers);
        LocalDateTime answeredAt = LocalDateTime.now();

        SentenceClozeAnswerRecord record = new SentenceClozeAnswerRecord();
        record.setUserId(userId);
        record.setUserName(resolveUserName(userId));
        record.setClozeItemId(item.getId());
        record.setAnswerText(answerText);
        record.setAnswersJson(toJson(answers));
        record.setExpectedWordsJson(toJson(expectedWords));
        record.setSubmissionKey(submissionKey);
        record.setPracticeContext(practiceContext);
        record.setActionType(actionType);
        record.setWrongBlankIndexesJson(toJson(comparison.wrongIndexes()));
        record.setIsCorrect(comparison.correct());
        record.setAttemptNo(attemptNo);
        record.setCostMs(normalizeCostMs(request.getCostMs()));
        if (submissionKey == null) {
            answerRecordMapper.insert(record);
        } else if (answerRecordMapper.insertIdempotent(record) == 0) {
            SentenceClozeAnswerRecord replay = answerRecordMapper.selectBySubmissionKey(userId, submissionKey);
            if (replay == null) {
                throw new IllegalStateException("重复提交结果读取失败");
            }
            return replaySubmission(request.getClozeItemId(), replay);
        }

        if (!comparison.correct()) {
            reviewScheduleMapper.upsertWrongSchedule(userId, item.getId(), record.getId(), answeredAt);
        } else {
            reviewScheduleMapper.advanceDueCorrectSchedule(
                    userId,
                    item.getId(),
                    record.getId(),
                    answeredAt,
                    answeredAt.plusDays(7),
                    answeredAt.plusDays(15)
            );
        }

        if ("word-agent".equals(item.getSource())) {
            WrongWordReviewProgressService.WordReviewUpdateResult update =
                    wrongWordReviewProgressService.applyAnswer(
                            userId,
                            item,
                            record.getId(),
                            comparison,
                            answeredAt
                    );
            markCompletedReviewWords(userId, item, record, update.completedWords());
        } else if (comparison.correct()) {
            markMasteredWordsFromCorrectCloze(userId, item, record, expectedWords);
        }

        if (!comparison.correct()) {
            wrongWordAgentNotificationService.notifyClozeWrongAnswer(
                    record, item, expectedWords, answers, comparison.wrongIndexes());
        }

        return toAnswerResponse(record);
    }

    private ClozePracticeAnswerResponse replaySubmission(
            Long requestedClozeItemId,
            SentenceClozeAnswerRecord replay
    ) {
        if (!requestedClozeItemId.equals(replay.getClozeItemId())) {
            throw new IllegalArgumentException("submissionKey 已用于其他题目");
        }
        return toAnswerResponse(replay);
    }

    private ClozePracticeAnswerResponse toAnswerResponse(SentenceClozeAnswerRecord record) {
        ClozePracticeAnswerResponse response = new ClozePracticeAnswerResponse();
        response.setRecordId(record.getId());
        response.setClozeItemId(record.getClozeItemId());
        response.setCorrect(record.getIsCorrect());
        response.setAnswerText(record.getAnswerText());
        response.setAnswers(parseWordsJson(record.getAnswersJson()));
        response.setExpectedWords(parseWordsJson(record.getExpectedWordsJson()));
        response.setAttemptNo(record.getAttemptNo());
        response.setMessage(Boolean.TRUE.equals(record.getIsCorrect()) ? "答对了" : "答案不正确");
        return response;
    }

    private void markCompletedReviewWords(
            Long userId,
            SentenceClozeItem item,
            SentenceClozeAnswerRecord record,
            List<WrongWordReviewProgressService.CompletedReviewWord> completedWords
    ) {
        for (WrongWordReviewProgressService.CompletedReviewWord completed : completedWords) {
            String expectedWord = normalizeString(completed.word());
            if (expectedWord.isBlank()) {
                continue;
            }
            Word word = findMatchingWordById(completed.wordId(), expectedWord);
            if (word == null) {
                word = wordMapper.findActiveWordByContent(expectedWord);
            }
            if (word == null || word.getId() == null) {
                continue;
            }
            userWordMasteryService.markMasteredFromCloze(
                    userId,
                    word.getId(),
                    nonBlank(word.getWord(), expectedWord),
                    resolveMasteryMeaning(word, item),
                    record.getId()
            );
        }
    }

    private void markMasteredWordsFromCorrectCloze(Long userId,
                                                   SentenceClozeItem item,
                                                   SentenceClozeAnswerRecord record,
                                                   List<String> expectedWords) {
        List<Long> sourceWordIds = parseLongsJson(item.getSourceWordIdsJson());
        for (int index = 0; index < expectedWords.size(); index++) {
            String expectedWord = normalizeString(expectedWords.get(index));
            if (expectedWord.isBlank()) {
                continue;
            }
            Word word = resolveMasteryWord(expectedWord, sourceWordIds, index);
            if (word == null || word.getId() == null) {
                continue;
            }
            userWordMasteryService.markMasteredFromCloze(
                    userId,
                    word.getId(),
                    nonBlank(word.getWord(), expectedWord),
                    resolveMasteryMeaning(word, item),
                    record.getId()
            );
        }
    }

    private Word resolveMasteryWord(String expectedWord, List<Long> sourceWordIds, int index) {
        Long preferredWordId = index < sourceWordIds.size() ? sourceWordIds.get(index) : null;
        Word preferredWord = findMatchingWordById(preferredWordId, expectedWord);
        if (preferredWord != null) {
            return preferredWord;
        }

        for (Long sourceWordId : sourceWordIds) {
            Word candidate = findMatchingWordById(sourceWordId, expectedWord);
            if (candidate != null) {
                return candidate;
            }
        }
        return wordMapper.findActiveWordByContent(expectedWord);
    }

    private Word findMatchingWordById(Long wordId, String expectedWord) {
        if (wordId == null || wordId <= 0) {
            return null;
        }
        Word word = wordMapper.selectById(wordId);
        if (word == null || word.getWord() == null) {
            return null;
        }
        return normalizeForCompare(word.getWord()).equals(normalizeForCompare(expectedWord)) ? word : null;
    }

    private String resolveMasteryMeaning(Word word, SentenceClozeItem item) {
        String wordMeaning = word == null ? "" : normalizeString(word.getMeaning());
        if (!wordMeaning.isBlank()) {
            return wordMeaning;
        }
        String explanation = item == null ? "" : normalizeString(item.getExplanationZh());
        if (!explanation.isBlank()) {
            return explanation;
        }
        return item == null ? "" : normalizeString(item.getTranslationZh());
    }

    public List<ClozePracticeHistoryItem> getHistory(Long userId, Integer limit) {
        requireUserId(userId);
        int normalizedLimit = limit == null ? DEFAULT_HISTORY_LIMIT : Math.max(1, Math.min(limit, MAX_HISTORY_LIMIT));
        return answerRecordMapper.selectHistory(userId, normalizedLimit);
    }

    public ClozePracticeStatsResponse getStats(Long userId) {
        requireUserId(userId);
        long totalTasks = countTasks(userId);
        long completedTasks = nullToZero(answerRecordMapper.countCompletedItems(userId));
        long totalAnswers = countAnswers(userId, null);
        long correctAnswers = countAnswers(userId, true);
        long wrongAnswers = countAnswers(userId, false);
        long activeWrongSentences = nullToZero(sentenceClozeItemMapper.countActiveWrongSentences(userId));
        long dueReviewTasks = nullToZero(sentenceClozeItemMapper.countDueReviewItems(userId));

        ClozePracticeStatsResponse response = new ClozePracticeStatsResponse();
        response.setTotalTasks(totalTasks);
        response.setCompletedTasks(completedTasks);
        response.setPendingTasks(Math.max(totalTasks - completedTasks, 0));
        response.setTotalAnswers(totalAnswers);
        response.setCorrectAnswers(correctAnswers);
        response.setWrongAnswers(wrongAnswers);
        response.setActiveWrongSentences(activeWrongSentences);
        response.setDueReviewTasks(dueReviewTasks);
        response.setAccuracy(totalAnswers == 0 ? 0.0 : Math.round(correctAnswers * 10000.0 / totalAnswers) / 100.0);
        return response;
    }

    public ClozeWrongSentencePageResponse getWrongSentences(
            Long userId,
            String status,
            String source,
            String availability,
            String keyword,
            String sort,
            Integer page,
            Integer size
    ) {
        requireUserId(userId);
        String normalizedStatus = normalizeAllowed(status, Set.of("active", "completed"), "active");
        String normalizedSource = normalizeAllowed(source, Set.of("all", "review", "solo"), "all");
        String normalizedAvailability = normalizeAllowed(
                availability, Set.of("all", "due", "waiting"), "all");
        String normalizedSort = normalizeAllowed(
                sort, Set.of("nextreview", "recent", "wrongcount"), "nextreview");
        normalizedSort = switch (normalizedSort) {
            case "wrongcount" -> "wrongCount";
            case "nextreview" -> "nextReview";
            default -> normalizedSort;
        };
        String normalizedKeyword = normalizeString(keyword).toLowerCase(Locale.ROOT);
        int normalizedPage = Math.max(page == null ? 1 : page, 1);
        int normalizedSize = Math.max(1, Math.min(size == null ? 20 : size, 100));
        int offset = (normalizedPage - 1) * normalizedSize;

        List<ClozeWrongSentenceItem> items = wrongSentenceQueryMapper.selectWrongSentences(
                userId,
                normalizedStatus,
                normalizedSource,
                normalizedAvailability,
                normalizedKeyword,
                normalizedSort,
                offset,
                normalizedSize
        );
        items.forEach(this::hydrateWrongSentenceItem);
        long total = nullToZero(wrongSentenceQueryMapper.countWrongSentences(
                userId,
                normalizedStatus,
                normalizedSource,
                normalizedAvailability,
                normalizedKeyword
        ));

        ClozeWrongSentencePageResponse response = new ClozeWrongSentencePageResponse();
        response.setItems(List.copyOf(items));
        response.setTotal(total);
        response.setCurrent(normalizedPage);
        response.setPages((int) Math.ceil(total / (double) normalizedSize));
        response.setSummary(normalizeSummary(wrongSentenceQueryMapper.selectSummary(userId)));
        return response;
    }

    public ClozeWrongSentenceDetail getWrongSentenceDetail(Long userId, Long progressId) {
        requireUserId(userId);
        if (progressId == null || progressId <= 0) {
            throw new IllegalArgumentException("progressId 不能为空");
        }
        ClozeWrongSentenceItem item = wrongSentenceQueryMapper.selectWrongSentenceById(userId, progressId);
        if (item == null) {
            throw new IllegalArgumentException("错题不存在或不属于当前用户");
        }
        hydrateWrongSentenceItem(item);

        List<ClozeWrongSentenceAttempt> attempts = wrongSentenceQueryMapper.selectRecentAttempts(
                userId, item.getClozeItemId(), 5);
        for (ClozeWrongSentenceAttempt attempt : attempts) {
            if (normalizeString(attempt.getPracticeContext()).isBlank()) {
                attempt.setPracticeContext(item.getPracticeContext());
            }
            if (normalizeString(attempt.getActionType()).isBlank()) {
                attempt.setActionType("answer");
            }
        }

        Map<Integer, WrongWordReviewProgress> wordProgressByIndex = new LinkedHashMap<>();
        for (WrongWordReviewProgress progress : wrongSentenceQueryMapper.selectWordProgresses(
                userId, item.getClozeItemId())) {
            if (progress.getActiveBlankIndex() != null) {
                wordProgressByIndex.put(progress.getActiveBlankIndex(), progress);
            }
        }

        List<Integer> latestWrongIndexes = attempts.isEmpty()
                ? item.getWrongBlankIndexes()
                : parseIntegerJson(attempts.getFirst().getWrongBlankIndexesJson());
        Boolean latestCorrect = attempts.isEmpty() ? null : attempts.getFirst().getCorrect();
        List<ClozeWrongSentenceDetail.BlankReview> blanks = new ArrayList<>();
        for (int index = 0; index < item.getTargetWords().size(); index++) {
            String targetWord = item.getTargetWords().get(index);
            WrongWordReviewProgress progress = wordProgressByIndex.get(index);
            Word word = wordMapper.findActiveWordByContent(targetWord);

            ClozeWrongSentenceDetail.BlankReview blank = new ClozeWrongSentenceDetail.BlankReview();
            blank.setIndex(index);
            blank.setWord(targetWord);
            blank.setLastCorrect(latestCorrect == null
                    ? null
                    : Boolean.TRUE.equals(latestCorrect) || !latestWrongIndexes.contains(index));
            blank.setMeaning(word == null ? "" : normalizeString(word.getMeaning()));
            blank.setWordReviewStage(progress == null ? null : progress.getReviewStage());
            blank.setWordReviewStatus(progress == null ? null : progress.getStatus());
            blanks.add(blank);
        }

        ClozeWrongSentenceDetail detail = new ClozeWrongSentenceDetail();
        detail.setItem(item);
        detail.setBlanks(List.copyOf(blanks));
        detail.setAttempts(List.copyOf(attempts));
        detail.setReviewStages(buildReviewStages(item));
        return detail;
    }

    private void hydrateWrongSentenceItem(ClozeWrongSentenceItem item) {
        List<String> targetWords = parseWordsJson(item.getTargetWordsJson());
        List<Integer> wrongIndexes = parseIntegerJson(item.getWrongBlankIndexesJson());
        item.setTargetWords(targetWords);
        item.setWrongBlankIndexes(wrongIndexes);
        item.setWrongBlankCount(wrongIndexes.size());
    }

    private List<ClozeWrongSentenceDetail.ReviewStageStep> buildReviewStages(ClozeWrongSentenceItem item) {
        List<String> labels = List.of("立即", "7 天", "15 天", "完成");
        int currentStage = item.getReviewStage() == null ? 0 : item.getReviewStage();
        boolean completed = "completed".equals(item.getStatus());
        List<ClozeWrongSentenceDetail.ReviewStageStep> stages = new ArrayList<>();
        for (int stage = 0; stage < labels.size(); stage++) {
            ClozeWrongSentenceDetail.ReviewStageStep step = new ClozeWrongSentenceDetail.ReviewStageStep();
            step.setStage(stage);
            step.setLabel(labels.get(stage));
            if (completed || stage < currentStage) {
                step.setState("completed");
            } else if (stage == currentStage) {
                step.setState("current");
            } else {
                step.setState("upcoming");
            }
            stages.add(step);
        }
        return List.copyOf(stages);
    }

    private ClozeWrongSentencePageResponse.Summary normalizeSummary(
            ClozeWrongSentencePageResponse.Summary summary
    ) {
        if (summary == null) {
            summary = new ClozeWrongSentencePageResponse.Summary();
        }
        summary.setActiveCount(nullToZero(summary.getActiveCount()));
        summary.setDueCount(nullToZero(summary.getDueCount()));
        summary.setStage1Count(nullToZero(summary.getStage1Count()));
        summary.setStage2Count(nullToZero(summary.getStage2Count()));
        summary.setCompletedCount(nullToZero(summary.getCompletedCount()));
        return summary;
    }

    private String normalizeAllowed(String value, Set<String> allowed, String defaultValue) {
        String normalized = normalizeString(value).toLowerCase(Locale.ROOT);
        return allowed.contains(normalized) ? normalized : defaultValue;
    }

    private ClozePracticeTaskResponse toTaskResponse(Long userId, SentenceClozeItem item) {
        Map<String, String> wordAudioUrls = loadWordAudioUrls(List.of(item));
        return toTaskResponse(userId, item, null, null, null, wordAudioUrls);
    }

    private List<ClozePracticeTaskResponse> toTaskResponses(
            Long userId,
            List<SentenceClozeItem> items,
            String difficultyGroup,
            String difficultyLevel,
            String difficultyLabel
    ) {
        Map<String, String> wordAudioUrls = loadWordAudioUrls(items);
        return items.stream()
                .map(item -> toTaskResponse(
                        userId,
                        item,
                        difficultyGroup,
                        difficultyLevel,
                        difficultyLabel,
                        wordAudioUrls
                ))
                .toList();
    }

    private Map<String, String> loadWordAudioUrls(List<SentenceClozeItem> items) {
        List<String> words = items.stream()
                .map(SentenceClozeItem::getWord)
                .filter(word -> word != null && !word.isBlank())
                .distinct()
                .toList();
        if (words.isEmpty()) {
            return Map.of();
        }
        return wordCleanTtsMapper.selectSuccessfulByWords(words).stream()
                .collect(Collectors.toUnmodifiableMap(
                        WordCleanTts::getWord,
                        WordCleanTts::getTtsObjectUrl,
                        (first, ignored) -> first
                ));
    }

    private ClozePracticeTaskResponse toTaskResponse(
            Long userId,
            SentenceClozeItem item,
            String difficultyGroup,
            String difficultyLevel,
            String difficultyLabel,
            Map<String, String> wordAudioUrls
    ) {
        List<String> expectedWords = parseWordsJson(item.getBlankWordsJson());
        ClozePracticeTaskResponse response = new ClozePracticeTaskResponse();
        response.setId(item.getId());
        response.setWord(item.getWord());
        response.setWordAudioUrl(wordAudioUrls.get(item.getWord()));
        response.setSentence(item.getSentence());
        response.setSentenceAudioUrl(item.getSentenceAudioUrl());
        response.setClozeSentence(item.getClozeSentence());
        response.setTranslationZh(item.getTranslationZh());
        response.setBlankCount(expectedWords.size());
        response.setBlankLengths(resolveBlankLengths(expectedWords));
        response.setAttemptCount(countAttempts(userId, item.getId()));
        response.setWrongCount(countWrongAttempts(userId, item.getId()));
        response.setDifficultyGroup(difficultyGroup);
        response.setDifficultyLevel(difficultyLevel);
        response.setDifficultyLabel(difficultyLabel == null ? item.getProviderLabel() : difficultyLabel);
        response.setSource(item.getSource());
        response.setModel(item.getModel());
        response.setLatestAnswerCorrect(item.getLatestAnswerCorrect());
        response.setLatestAnswerTime(item.getLatestAnswerTime());
        response.setNextReviewTime(item.getNextReviewTime());
        response.setCreateTime(item.getCreateTime());
        return response;
    }

    private SentenceClozeItem buildDifficultyPracticeItem(
            Long userId,
            String difficultyGroup,
            String difficultyLevel,
            ClozePracticeSentenceCandidate candidate
    ) {
        String word = normalizeString(candidate.getWord());
        SentenceClozeItem item = new SentenceClozeItem();
        item.setUserId(userId);
        item.setUserName(resolveUserName(userId));
        item.setWord(word);
        item.setWordsJson(toJson(List.of(word)));
        item.setSentence(candidate.getSentence());
        item.setBestSentenceId(candidate.getBestSentenceId());
        item.setSentenceAudioUrl(normalizeString(candidate.getTtsObjectUrl()));
        item.setTranslationZh(normalizeString(candidate.getSentenceTranslation()).isBlank()
                ? normalizeString(candidate.getMeaning())
                : normalizeString(candidate.getSentenceTranslation()));
        item.setExplanationZh(normalizeString(candidate.getMeaning()));
        String clozeAnswer = normalizeString(candidate.getClozeAnswer());
        item.setBlankWordsJson(toJson(List.of(clozeAnswer)));
        item.setClozeSentence(normalizeString(candidate.getClozeSentence()));
        item.setProviderId("word-clean-best-sentence");
        item.setProviderLabel(difficultyLabel(difficultyGroup, difficultyLevel));
        item.setModel(normalizeString(candidate.getModelName()));
        item.setSource("best-sentence-practice");
        item.setSourceEventIdsJson("[]");
        item.setSourceAnswerDetailIdsJson("[]");
        item.setSourceRecordIdsJson("[]");
        item.setSourceWordIdsJson(toJson(List.of(candidate.getWordCleanId())));
        return item;
    }

    private String buildClozeSentence(String sentence, String word) {
        if (sentence == null || sentence.isBlank() || word == null || word.isBlank()) {
            return sentence == null ? "" : sentence;
        }
        String quotedWord = java.util.regex.Pattern.quote(word);
        java.util.regex.Pattern boundaryPattern = java.util.regex.Pattern.compile("(?i)(?<![A-Za-z])" + quotedWord + "(?![A-Za-z])");
        String clozeSentence = boundaryPattern.matcher(sentence).replaceAll("____");
        if (clozeSentence.equals(sentence)) {
            java.util.regex.Pattern fallbackPattern = java.util.regex.Pattern.compile(
                    "(?i)(?<![A-Za-z])[A-Za-z]*" + quotedWord + "[A-Za-z]*(?![A-Za-z])");
            clozeSentence = fallbackPattern.matcher(sentence).replaceAll("____");
        }
        return clozeSentence;
    }

    private List<Integer> resolveBlankLengths(List<String> expectedWords) {
        return expectedWords.stream()
                .map(word -> {
                    String trimmedWord = word == null ? "" : word.trim();
                    return Math.max(trimmedWord.codePointCount(0, trimmedWord.length()), 1);
                })
                .toList();
    }

    private List<String> normalizeAnswers(ClozePracticeAnswerRequest request) {
        if (request.getAnswers() != null && !request.getAnswers().isEmpty()) {
            return request.getAnswers().stream()
                    .map(answer -> answer == null ? "" : answer.trim())
                    .toList();
        }

        List<String> answers = new ArrayList<>();
        if (request.getAnswerText() != null) {
            String[] parts = request.getAnswerText().split("[,，\\s]+");
            for (String part : parts) {
                String answer = part.trim();
                if (!answer.isEmpty()) {
                    answers.add(answer);
                }
            }
        }
        return answers;
    }

    private String normalizeSubmissionKey(String submissionKey) {
        String normalized = normalizeString(submissionKey);
        if (normalized.isBlank()) {
            return null;
        }
        if (normalized.length() > 64) {
            throw new IllegalArgumentException("submissionKey 长度不能超过 64");
        }
        return normalized;
    }

    private String normalizeSubmissionMetadata(
            String value,
            String fieldName,
            List<String> allowedValues,
            String defaultValue
    ) {
        String normalized = normalizeString(value).toLowerCase(Locale.ROOT);
        if (normalized.isBlank()) {
            return defaultValue;
        }
        if (!allowedValues.contains(normalized)) {
            throw new IllegalArgumentException(fieldName + " 不合法");
        }
        return normalized;
    }

    private String normalizeForCompare(String value) {
        if (value == null) {
            return "";
        }
        String normalized = Normalizer.normalize(value.trim(), Normalizer.Form.NFKC);
        return normalized.toLowerCase(Locale.ROOT);
    }

    private String buildAnswerText(String answerText, List<String> answers) {
        if (answerText != null && !answerText.trim().isEmpty()) {
            return answerText.trim();
        }
        return String.join(", ", answers);
    }

    private Long normalizeCostMs(Long costMs) {
        if (costMs == null) {
            return null;
        }
        return Math.max(costMs, 0L);
    }

    private List<String> parseWordsJson(String json) {
        if (json == null || json.isBlank()) {
            return List.of();
        }
        try {
            return objectMapper.readValue(json, new TypeReference<List<String>>() {
            });
        } catch (JsonProcessingException e) {
            throw new IllegalStateException("挖空答案 JSON 解析失败", e);
        }
    }

    private List<Long> parseLongsJson(String json) {
        if (json == null || json.isBlank()) {
            return List.of();
        }
        try {
            return objectMapper.readValue(json, new TypeReference<List<Long>>() {
            });
        } catch (JsonProcessingException e) {
            return List.of();
        }
    }

    private List<Integer> parseIntegerJson(String json) {
        if (json == null || json.isBlank()) {
            return List.of();
        }
        try {
            return objectMapper.readValue(json, new TypeReference<List<Integer>>() {
            });
        } catch (JsonProcessingException e) {
            return List.of();
        }
    }

    private String toJson(Object value) {
        try {
            return objectMapper.writeValueAsString(value);
        } catch (JsonProcessingException e) {
            throw new IllegalStateException("答案 JSON 编码失败", e);
        }
    }

    private int countAttempts(Long userId, Long clozeItemId) {
        return Math.toIntExact(answerRecordMapper.selectCount(new LambdaQueryWrapper<SentenceClozeAnswerRecord>()
                .eq(SentenceClozeAnswerRecord::getUserId, userId)
                .eq(SentenceClozeAnswerRecord::getClozeItemId, clozeItemId)));
    }

    private int countWrongAttempts(Long userId, Long clozeItemId) {
        return Math.toIntExact(answerRecordMapper.selectCount(new LambdaQueryWrapper<SentenceClozeAnswerRecord>()
                .eq(SentenceClozeAnswerRecord::getUserId, userId)
                .eq(SentenceClozeAnswerRecord::getClozeItemId, clozeItemId)
                .eq(SentenceClozeAnswerRecord::getIsCorrect, false)));
    }

    private long countTasks(Long userId) {
        return sentenceClozeItemMapper.selectCount(new LambdaQueryWrapper<SentenceClozeItem>()
                .eq(SentenceClozeItem::getUserId, userId));
    }

    private long countAnswers(Long userId, Boolean correct) {
        LambdaQueryWrapper<SentenceClozeAnswerRecord> wrapper = new LambdaQueryWrapper<SentenceClozeAnswerRecord>()
                .eq(SentenceClozeAnswerRecord::getUserId, userId);
        if (correct != null) {
            wrapper.eq(SentenceClozeAnswerRecord::getIsCorrect, correct);
        }
        return answerRecordMapper.selectCount(wrapper);
    }

    private String resolveUserName(Long userId) {
        User user = userMapper.selectById(userId);
        if (user == null) {
            return null;
        }
        if (user.getNickname() != null && !user.getNickname().isBlank()) {
            return user.getNickname().trim();
        }
        return user.getUsername();
    }

    private String nonBlank(String next, String fallback) {
        return next != null && !next.isBlank() ? next : fallback;
    }

    private void requireUserId(Long userId) {
        if (userId == null || userId <= 0) {
            throw new IllegalArgumentException("用户信息不能为空");
        }
    }

    private long nullToZero(Long value) {
        return value == null ? 0 : value;
    }
}
