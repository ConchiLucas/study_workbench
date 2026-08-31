package com.robword.controller;

import com.robword.dto.ClozePracticeAnswerRequest;
import com.robword.dto.ClozePracticeAnswerResponse;
import com.robword.dto.ClozePracticeDifficultyBatchRequest;
import com.robword.dto.ClozePracticeHistoryItem;
import com.robword.dto.ClozePracticePreferenceResponse;
import com.robword.dto.ClozePracticeStatsResponse;
import com.robword.dto.ClozePracticeTaskResponse;
import com.robword.dto.ClozeWrongSentenceDetail;
import com.robword.dto.ClozeWrongSentencePageResponse;
import com.robword.dto.UpdateSoloDifficultyRequest;
import com.robword.service.ClozePracticeService;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;

@RestController
@RequestMapping("/api/cloze-practice")
@RequiredArgsConstructor
public class ClozePracticeController {

    private final ClozePracticeService clozePracticeService;

    @GetMapping("/tasks/next")
    public ResponseEntity<ClozePracticeTaskResponse> getNextTask(Authentication auth) {
        ClozePracticeTaskResponse task = clozePracticeService.getNextTask(resolveUserId(auth));
        return task == null ? ResponseEntity.noContent().build() : ResponseEntity.ok(task);
    }

    @GetMapping("/preferences")
    public ResponseEntity<ClozePracticePreferenceResponse> getPreferences(Authentication auth) {
        return ResponseEntity.ok(clozePracticeService.getPreferences(resolveUserId(auth)));
    }

    @PutMapping("/preferences/solo-difficulty")
    public ResponseEntity<ClozePracticePreferenceResponse> updateSoloDifficulty(
            Authentication auth,
            @RequestBody UpdateSoloDifficultyRequest request
    ) {
        return ResponseEntity.ok(clozePracticeService.updateSoloDifficulty(resolveUserId(auth), request));
    }

    @PostMapping("/tasks/difficulty-batch")
    public ResponseEntity<List<ClozePracticeTaskResponse>> createDifficultyBatch(
            Authentication auth,
            @RequestBody ClozePracticeDifficultyBatchRequest request
    ) {
        return ResponseEntity.ok(clozePracticeService.createDifficultyBatch(resolveUserId(auth), request));
    }

    @GetMapping("/tasks/pending")
    public ResponseEntity<List<ClozePracticeTaskResponse>> getPendingTasks(
            Authentication auth,
            @RequestParam(required = false) Integer limit
    ) {
        return ResponseEntity.ok(clozePracticeService.getPendingTasks(resolveUserId(auth), limit));
    }

    @GetMapping("/tasks/review-due")
    public ResponseEntity<List<ClozePracticeTaskResponse>> getDueReviewTasks(
            Authentication auth,
            @RequestParam(required = false) Integer limit
    ) {
        return ResponseEntity.ok(clozePracticeService.getDueReviewTasks(resolveUserId(auth), limit));
    }

    @GetMapping("/tasks/answered")
    public ResponseEntity<List<ClozePracticeTaskResponse>> getAnsweredTasks(
            Authentication auth,
            @RequestParam String status,
            @RequestParam(required = false) Integer limit
    ) {
        return ResponseEntity.ok(clozePracticeService.getAnsweredTasks(resolveUserId(auth), status, limit));
    }

    @PostMapping("/answers")
    public ResponseEntity<ClozePracticeAnswerResponse> submitAnswer(
            Authentication auth,
            @RequestBody ClozePracticeAnswerRequest request
    ) {
        return ResponseEntity.ok(clozePracticeService.submitAnswer(resolveUserId(auth), request));
    }

    @GetMapping("/history")
    public ResponseEntity<List<ClozePracticeHistoryItem>> getHistory(
            Authentication auth,
            @RequestParam(required = false) Integer limit
    ) {
        return ResponseEntity.ok(clozePracticeService.getHistory(resolveUserId(auth), limit));
    }

    @GetMapping("/stats")
    public ResponseEntity<ClozePracticeStatsResponse> getStats(Authentication auth) {
        return ResponseEntity.ok(clozePracticeService.getStats(resolveUserId(auth)));
    }

    @GetMapping("/wrong-sentences")
    public ResponseEntity<ClozeWrongSentencePageResponse> getWrongSentences(
            Authentication auth,
            @RequestParam(required = false) String status,
            @RequestParam(required = false) String source,
            @RequestParam(required = false) String availability,
            @RequestParam(required = false) String keyword,
            @RequestParam(required = false) String sort,
            @RequestParam(required = false) Integer page,
            @RequestParam(required = false) Integer size
    ) {
        return ResponseEntity.ok(clozePracticeService.getWrongSentences(
                resolveUserId(auth), status, source, availability, keyword, sort, page, size));
    }

    @GetMapping("/wrong-sentences/{progressId}")
    public ResponseEntity<ClozeWrongSentenceDetail> getWrongSentenceDetail(
            Authentication auth,
            @PathVariable Long progressId
    ) {
        return ResponseEntity.ok(clozePracticeService.getWrongSentenceDetail(
                resolveUserId(auth), progressId));
    }

    private Long resolveUserId(Authentication auth) {
        if (auth == null || !(auth.getPrincipal() instanceof Long userId)) {
            throw new IllegalArgumentException("请先登录");
        }
        return userId;
    }
}
