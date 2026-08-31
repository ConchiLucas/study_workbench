package com.robword.controller;

import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.robword.entity.GameAnswerDetail;
import com.robword.entity.GameRecord;
import com.robword.mapper.GameAnswerDetailMapper;
import com.robword.mapper.GameRecordMapper;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

@RestController
@RequestMapping("/api/game")
@RequiredArgsConstructor
public class GameController {

    private final GameRecordMapper gameRecordMapper;
    private final GameAnswerDetailMapper gameAnswerDetailMapper;

    @GetMapping("/records")
    public ResponseEntity<Map<String, Object>> getGameRecords(
            Authentication auth,
            @RequestParam(defaultValue = "1") Integer page,
            @RequestParam(defaultValue = "10") Integer size,
            @RequestParam(defaultValue = "match") String mode) {

        Long userId = (Long) auth.getPrincipal();
        String normalizedMode = normalizeRecordMode(mode);

        // 查询当前用户参与的游戏记录，按创建时间倒序
        QueryWrapper<GameRecord> wrapper = new QueryWrapper<>();
        wrapper.and(w -> w.eq("player1_id", userId).or().eq("player2_id", userId));
        if (!"all".equals(normalizedMode)) {
            wrapper.eq("mode", normalizedMode);
        }
        wrapper.orderByDesc("create_time");

        Page<GameRecord> pageParam = new Page<>(page, size);
        Page<GameRecord> recordPage = gameRecordMapper.selectPage(pageParam, wrapper);

        Map<String, Object> result = new HashMap<>();
        result.put("records", recordPage.getRecords());
        result.put("total", recordPage.getTotal());
        result.put("current", recordPage.getCurrent());
        result.put("pages", recordPage.getPages());

        return ResponseEntity.ok(result);
    }

    @GetMapping("/records/{recordId}")
    public ResponseEntity<?> getGameRecord(
            Authentication auth,
            @PathVariable Long recordId) {

        Long userId = (Long) auth.getPrincipal();
        GameRecord record = gameRecordMapper.selectById(recordId);
        if (record == null) {
            return ResponseEntity.notFound().build();
        }
        if (!userId.equals(record.getPlayer1Id()) && !userId.equals(record.getPlayer2Id())) {
            return ResponseEntity.status(403).body("无权查看该记录");
        }
        return ResponseEntity.ok(record);
    }

    private String normalizeRecordMode(String mode) {
        if ("solo_training".equals(mode) || "all".equals(mode)) {
            return mode;
        }
        return "match";
    }

    @GetMapping("/answer-detail")
    public ResponseEntity<?> getAnswerDetail(
            Authentication auth,
            @RequestParam Long recordId,
            @RequestParam(required = false) Long targetUserId) {

        Long currentUserId = (Long) auth.getPrincipal();

        // 1. 验证当前用户是否有权限查看该记录
        GameRecord record = gameRecordMapper.selectById(recordId);
        if (record == null) {
            return ResponseEntity.notFound().build();
        }

        // 验证当前用户是否是该对战的参与者
        if (!currentUserId.equals(record.getPlayer1Id()) && !currentUserId.equals(record.getPlayer2Id())) {
            return ResponseEntity.status(403).body("无权查看该对战记录");
        }

        // 2. 确定要查询的用户ID
        Long queryUserId = targetUserId != null ? targetUserId : currentUserId;

        // 3. 查询该用户的答题详情
        QueryWrapper<GameAnswerDetail> wrapper = new QueryWrapper<>();
        wrapper.eq("record_id", recordId)
                .eq("user_id", queryUserId)
                .orderByAsc("round_no");

        List<GameAnswerDetail> details = gameAnswerDetailMapper.selectList(wrapper);

        return ResponseEntity.ok(details);
    }
}
