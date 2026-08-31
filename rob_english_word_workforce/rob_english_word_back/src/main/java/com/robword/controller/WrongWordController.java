package com.robword.controller;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.robword.dto.WrongWordDetail;
import com.robword.dto.WrongWordDetailResponse;
import com.robword.dto.WrongWordQueueEvent;
import com.robword.dto.WrongWordSummary;
import com.robword.entity.GameAnswerDetail;
import com.robword.entity.UserWordState;
import com.robword.mapper.GameAnswerDetailMapper;
import com.robword.mapper.UserWordStateMapper;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

@RestController
@RequestMapping("/api/wrong-words")
@RequiredArgsConstructor
public class WrongWordController {

    private static final int DETAIL_LIMIT = 5;

    private final GameAnswerDetailMapper gameAnswerDetailMapper;
    private final UserWordStateMapper userWordStateMapper;

    @GetMapping("/events")
    public ResponseEntity<Map<String, Object>> listWrongWordEvents(
            Authentication auth,
            @RequestParam(required = false) String keyword,
            @RequestParam(defaultValue = "recent") String sort,
            @RequestParam(defaultValue = "1") Integer page,
            @RequestParam(defaultValue = "20") Integer size) {

        Long userId = (Long) auth.getPrincipal();
        String normalizedKeyword = normalizeKeyword(keyword);
        String normalizedSort = normalizeSort(sort);
        int normalizedPage = Math.max(page != null ? page : 1, 1);
        int normalizedSize = Math.min(Math.max(size != null ? size : 20, 1), 500);
        long offset = (long) (normalizedPage - 1) * normalizedSize;

        List<WrongWordQueueEvent> items = gameAnswerDetailMapper.selectQueueEligibleWrongWordEvents(
                userId, normalizedKeyword, normalizedSort, normalizedSize, offset);
        Long count = gameAnswerDetailMapper.countQueueEligibleWrongWordEvents(userId, normalizedKeyword);
        long total = count == null ? 0 : count;
        long pages = total == 0 ? 0 : (long) Math.ceil((double) total / normalizedSize);

        Map<String, Object> result = new HashMap<>();
        result.put("items", items);
        result.put("total", total);
        result.put("current", normalizedPage);
        result.put("pages", pages);
        return ResponseEntity.ok(result);
    }

    @GetMapping
    public ResponseEntity<Map<String, Object>> listWrongWords(
            Authentication auth,
            @RequestParam(required = false) String keyword,
            @RequestParam(defaultValue = "recent") String sort,
            @RequestParam(defaultValue = "1") Integer page,
            @RequestParam(defaultValue = "20") Integer size) {

        Long userId = (Long) auth.getPrincipal();
        String normalizedKeyword = normalizeKeyword(keyword);
        String normalizedSort = normalizeSort(sort);
        int normalizedPage = Math.max(page != null ? page : 1, 1);
        int normalizedSize = Math.min(Math.max(size != null ? size : 20, 1), 50);
        long offset = (long) (normalizedPage - 1) * normalizedSize;

        List<WrongWordSummary> items = gameAnswerDetailMapper.selectWrongWordSummaries(
                userId, normalizedKeyword, normalizedSort, normalizedSize, offset);
        long total = gameAnswerDetailMapper.countWrongWordSummaries(userId, normalizedKeyword);
        long pages = total == 0 ? 0 : (long) Math.ceil((double) total / normalizedSize);

        Map<String, Object> result = new HashMap<>();
        result.put("items", items);
        result.put("total", total);
        result.put("current", normalizedPage);
        result.put("pages", pages);

        return ResponseEntity.ok(result);
    }

    @GetMapping("/{wordId}/details")
    public ResponseEntity<WrongWordDetailResponse> getWrongWordDetails(
            Authentication auth,
            @PathVariable Long wordId) {

        Long userId = (Long) auth.getPrincipal();
        long wrongCount = gameAnswerDetailMapper.countWrongWordDetails(userId, wordId);
        if (wrongCount <= 0) {
            return ResponseEntity.notFound().build();
        }

        List<WrongWordDetail> details = gameAnswerDetailMapper.selectWrongWordDetails(
                userId, wordId, DETAIL_LIMIT);
        return ResponseEntity.ok(buildDetailResponse(wordId, details, wrongCount));
    }

    @PutMapping("/{wordId}/state")
    public ResponseEntity<?> updateWrongWordState(
            Authentication auth,
            @PathVariable Long wordId,
            @RequestBody Map<String, String> body) {

        Long userId = (Long) auth.getPrincipal();
        String status = normalizeState(body != null ? body.get("status") : null);
        if (status == null) {
            return ResponseEntity.badRequest().body("无效的错题状态");
        }

        long wrongCount = gameAnswerDetailMapper.selectCount(
                new LambdaQueryWrapper<GameAnswerDetail>()
                        .eq(GameAnswerDetail::getUserId, userId)
                        .eq(GameAnswerDetail::getWordId, wordId)
                        .eq(GameAnswerDetail::getIsCorrect, 0)
        );
        if (wrongCount <= 0) {
            return ResponseEntity.notFound().build();
        }

        LambdaQueryWrapper<UserWordState> wrapper = new LambdaQueryWrapper<UserWordState>()
                .eq(UserWordState::getUserId, userId)
                .eq(UserWordState::getWordId, wordId);

        if ("active".equals(status)) {
            userWordStateMapper.delete(wrapper);
        } else if (userWordStateMapper.selectCount(wrapper) == 0) {
            UserWordState state = new UserWordState();
            state.setUserId(userId);
            state.setWordId(wordId);
            userWordStateMapper.insert(state);
        }

        Map<String, Object> result = new HashMap<>();
        result.put("wordId", wordId);
        result.put("status", status);
        return ResponseEntity.ok(result);
    }

    private String normalizeSort(String sort) {
        if ("count".equals(sort)) {
            return sort;
        }
        return "recent";
    }

    private String normalizeKeyword(String keyword) {
        if (keyword == null) {
            return null;
        }
        String trimmed = keyword.trim();
        if (trimmed.isEmpty()) {
            return null;
        }
        return trimmed.length() > 64 ? trimmed.substring(0, 64) : trimmed;
    }

    private String normalizeState(String status) {
        if ("active".equals(status) || "mastered".equals(status)) {
            return status;
        }
        return null;
    }

    private WrongWordDetailResponse buildDetailResponse(
            Long wordId,
            List<WrongWordDetail> details,
            long wrongCount) {

        WrongWordDetailResponse response = new WrongWordDetailResponse();
        response.setWordId(wordId);
        response.setWrongCount((int) wrongCount);
        response.setDetails(details);

        if (!details.isEmpty()) {
            WrongWordDetail latest = details.get(0);
            response.setWordContent(latest.getWordContent());
            response.setCorrectMeaning(latest.getCorrectMeaning());
            response.setLastWrongTime(latest.getCreateTime());
        }
        return response;
    }
}
