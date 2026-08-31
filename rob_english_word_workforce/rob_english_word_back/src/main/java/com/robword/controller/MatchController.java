package com.robword.controller;

import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.*;

import java.util.Map;

/**
 * 匹配控制器
 * 注意：匹配逻辑已迁移到 Netty WebSocket（match_start / match_cancel 消息）
 * 此控制器保留为兼容性接口，可在前端完全切换 WebSocket 后删除
 */
@RestController
@RequestMapping("/api/match")
@RequiredArgsConstructor
public class MatchController {

    @PostMapping("/start")
    public ResponseEntity<Map<String, Object>> startMatch(Authentication auth) {
        // 匹配已迁移到 WebSocket，此接口仅返回提示
        return ResponseEntity.ok(Map.of(
                "status", "use_websocket",
                "message", "Please use WebSocket match_start message"
        ));
    }

    @PostMapping("/cancel")
    public ResponseEntity<Map<String, Object>> cancelMatch(Authentication auth) {
        // 匹配已迁移到 WebSocket，此接口仅返回提示
        return ResponseEntity.ok(Map.of(
                "status", "use_websocket",
                "message", "Please use WebSocket match_cancel message"
        ));
    }
}