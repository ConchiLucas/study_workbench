package com.robword.service;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.robword.dto.AuthResponse;
import com.robword.dto.RegisterRequest;
import com.robword.entity.User;
import com.robword.mapper.UserMapper;
import com.robword.util.JwtUtil;
import lombok.RequiredArgsConstructor;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;

@Service
@RequiredArgsConstructor
public class AuthService {

    private final UserMapper userMapper;
    private final PasswordEncoder passwordEncoder;
    private final JwtUtil jwtUtil;

    public AuthResponse register(RegisterRequest request) {
        LambdaQueryWrapper<User> wrapper = new LambdaQueryWrapper<>();
        wrapper.eq(User::getUsername, request.getUsername());
        if (userMapper.selectCount(wrapper) > 0) {
            throw new RuntimeException("用户名已存在");
        }

        User user = new User();
        user.setUsername(request.getUsername());
        user.setPassword(passwordEncoder.encode(request.getPassword()));
        user.setNickname(request.getNickname());
        user.setRank(1);
        user.setExp(0);
        user.setTotalWins(0);
        user.setTotalGames(0);
        user.setCurrentWinStreak(0);
        user.setTrainingRank(1);
        user.setTrainingExp(0);
        user.setTrainingTotalWins(0);
        user.setTrainingTotalGames(0);

        userMapper.insert(user);

        String token = jwtUtil.generateToken(user.getId(), user.getUsername());
        return AuthResponse.of(token, user);
    }

    public AuthResponse login(String username, String password) {
        LambdaQueryWrapper<User> wrapper = new LambdaQueryWrapper<>();
        wrapper.eq(User::getUsername, username);
        User user = userMapper.selectOne(wrapper);

        if (user == null || !passwordEncoder.matches(password, user.getPassword())) {
            throw new RuntimeException("用户名或密码错误");
        }

        String token = jwtUtil.generateToken(user.getId(), user.getUsername());
        return AuthResponse.of(token, user);
    }

    public User getUserById(Long userId) {
        return userMapper.selectById(userId);
    }

    public void updateUser(User user) {
        userMapper.updateById(user);
    }
}
