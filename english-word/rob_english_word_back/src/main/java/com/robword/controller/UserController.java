package com.robword.controller;

import com.robword.dto.UserInfoResponse;
import com.robword.entity.User;
import com.robword.service.AuthService;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/user")
@RequiredArgsConstructor
public class UserController {

    private final AuthService authService;

    @GetMapping("/info")
    public ResponseEntity<UserInfoResponse> getUserInfo(Authentication auth) {
        Long userId = (Long) auth.getPrincipal();
        User user = authService.getUserById(userId);
        return ResponseEntity.ok(UserInfoResponse.of(user));
    }

    @GetMapping("/{userId}")
    public ResponseEntity<UserInfoResponse> getUserById(@PathVariable Long userId) {
        User user = authService.getUserById(userId);
        if (user == null) {
            return ResponseEntity.notFound().build();
        }
        return ResponseEntity.ok(UserInfoResponse.of(user));
    }

    @PutMapping("/profile")
    public ResponseEntity<Void> updateProfile(Authentication auth, @RequestBody User updateUser) {
        Long userId = (Long) auth.getPrincipal();
        User user = authService.getUserById(userId);
        if (updateUser.getNickname() != null) {
            user.setNickname(updateUser.getNickname());
        }
        if (updateUser.getAvatar() != null) {
            user.setAvatar(updateUser.getAvatar());
        }
        authService.updateUser(user);
        return ResponseEntity.ok().build();
    }
}