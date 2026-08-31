package com.robword.dto;

import lombok.Data;

@Data
public class AuthResponse {
    private String token;
    private Long userId;
    private String username;
    private String nickname;
    private Integer rank;
    private Integer exp;
    private Integer totalWins;
    private Integer totalGames;
    private Integer currentWinStreak;
    private Integer trainingRank;
    private Integer trainingExp;
    private Integer trainingTotalWins;
    private Integer trainingTotalGames;

    public static AuthResponse of(String token, com.robword.entity.User user) {
        AuthResponse response = new AuthResponse();
        response.setToken(token);
        response.setUserId(user.getId());
        response.setUsername(user.getUsername());
        response.setNickname(user.getNickname());
        response.setRank(user.getRank());
        response.setExp(user.getExp());
        response.setTotalWins(user.getTotalWins());
        response.setTotalGames(user.getTotalGames());
        response.setCurrentWinStreak(user.getCurrentWinStreak());
        response.setTrainingRank(user.getTrainingRank() != null ? user.getTrainingRank() : 1);
        response.setTrainingExp(user.getTrainingExp() != null ? user.getTrainingExp() : 0);
        response.setTrainingTotalWins(user.getTrainingTotalWins() != null ? user.getTrainingTotalWins() : 0);
        response.setTrainingTotalGames(user.getTrainingTotalGames() != null ? user.getTrainingTotalGames() : 0);
        return response;
    }
}
