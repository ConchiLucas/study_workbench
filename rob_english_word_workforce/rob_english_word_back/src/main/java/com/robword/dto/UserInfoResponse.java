package com.robword.dto;

import lombok.Data;

@Data
public class UserInfoResponse {
    private Long userId;
    private String username;
    private String nickname;
    private String avatar;
    private Integer rank;
    private Integer exp;
    private Integer totalWins;
    private Integer totalGames;
    private Integer currentWinStreak;
    private Integer trainingRank;
    private Integer trainingExp;
    private Integer trainingTotalWins;
    private Integer trainingTotalGames;
    private Double winRate;
    private Double trainingWinRate;

    public static UserInfoResponse of(com.robword.entity.User user) {
        UserInfoResponse response = new UserInfoResponse();
        response.setUserId(user.getId());
        response.setUsername(user.getUsername());
        response.setNickname(user.getNickname());
        response.setAvatar(user.getAvatar());
        response.setRank(user.getRank());
        response.setExp(user.getExp());
        response.setTotalWins(user.getTotalWins());
        response.setTotalGames(user.getTotalGames());
        response.setCurrentWinStreak(user.getCurrentWinStreak());
        response.setTrainingRank(user.getTrainingRank() != null ? user.getTrainingRank() : 1);
        response.setTrainingExp(user.getTrainingExp() != null ? user.getTrainingExp() : 0);
        response.setTrainingTotalWins(user.getTrainingTotalWins() != null ? user.getTrainingTotalWins() : 0);
        response.setTrainingTotalGames(user.getTrainingTotalGames() != null ? user.getTrainingTotalGames() : 0);
        if (user.getTotalGames() != null && user.getTotalGames() > 0) {
            response.setWinRate(user.getTotalWins() * 100.0 / user.getTotalGames());
        } else {
            response.setWinRate(0.0);
        }
        Integer trainingGames = response.getTrainingTotalGames();
        if (trainingGames != null && trainingGames > 0) {
            response.setTrainingWinRate(response.getTrainingTotalWins() * 100.0 / trainingGames);
        } else {
            response.setTrainingWinRate(0.0);
        }
        return response;
    }
}
