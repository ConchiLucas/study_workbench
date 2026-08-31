package com.robword.controller;

import com.robword.dto.MasteredWordSummary;
import com.robword.mapper.UserWordMasteryProgressMapper;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

@RestController
@RequestMapping("/api/mastered-words")
@RequiredArgsConstructor
public class MasteredWordController {

    private final UserWordMasteryProgressMapper masteryProgressMapper;

    @GetMapping
    public ResponseEntity<Map<String, Object>> listMasteredWords(
            Authentication auth,
            @RequestParam(required = false) String keyword,
            @RequestParam(defaultValue = "all") String status,
            @RequestParam(defaultValue = "recent") String sort,
            @RequestParam(defaultValue = "1") Integer page,
            @RequestParam(defaultValue = "20") Integer size) {

        Long userId = (Long) auth.getPrincipal();
        String normalizedKeyword = normalizeKeyword(keyword);
        String normalizedStatus = normalizeStatus(status);
        String normalizedSort = normalizeSort(sort);
        int normalizedPage = Math.max(page != null ? page : 1, 1);
        int normalizedSize = Math.min(Math.max(size != null ? size : 20, 1), 100);
        long offset = (long) (normalizedPage - 1) * normalizedSize;

        List<MasteredWordSummary> items = masteryProgressMapper.selectMasterySummaries(
                userId,
                normalizedKeyword,
                normalizedStatus,
                normalizedSort,
                normalizedSize,
                offset
        );
        long total = masteryProgressMapper.countMasterySummaries(userId, normalizedKeyword, normalizedStatus);
        long learningTotal = masteryProgressMapper.countMasterySummaries(userId, normalizedKeyword, "learning");
        long masteredTotal = masteryProgressMapper.countMasterySummaries(userId, normalizedKeyword, "mastered");
        long pages = total == 0 ? 0 : (long) Math.ceil((double) total / normalizedSize);

        Map<String, Object> result = new HashMap<>();
        result.put("items", items);
        result.put("total", total);
        result.put("learningTotal", learningTotal);
        result.put("masteredTotal", masteredTotal);
        result.put("current", normalizedPage);
        result.put("pages", pages);
        return ResponseEntity.ok(result);
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

    private String normalizeStatus(String status) {
        if ("learning".equals(status) || "mastered".equals(status)) {
            return status;
        }
        return null;
    }

    private String normalizeSort(String sort) {
        if ("due".equals(sort) || "mastered".equals(sort)) {
            return sort;
        }
        return "recent";
    }
}
