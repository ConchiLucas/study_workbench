package com.robword.controller;

import com.robword.dto.SentenceClozeGenerateRequest;
import com.robword.dto.SentenceClozeGenerateResponse;
import com.robword.service.SentenceClozeService;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/external/sentence-cloze")
@RequiredArgsConstructor
public class ExternalSentenceClozeController {

    private final SentenceClozeService sentenceClozeService;

    @PostMapping("/generate")
    public ResponseEntity<SentenceClozeGenerateResponse> generate(@RequestBody SentenceClozeGenerateRequest request) {
        return ResponseEntity.ok(sentenceClozeService.generateAndSave(request));
    }
}
