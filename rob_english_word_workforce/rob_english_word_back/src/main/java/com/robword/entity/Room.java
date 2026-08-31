package com.robword.entity;

import com.fasterxml.jackson.annotation.JsonIgnore;
import com.baomidou.mybatisplus.annotation.*;
import lombok.Data;
import java.time.LocalDateTime;

@Data
@TableName("room")
public class Room {

    /** 房间ID */
    @TableId(type = IdType.AUTO)
    private Long id;

    /** 房间号 */
    private String roomCode;

    /** 房间状态：0-等待中 1-对战中 2-已结束 */
    private Integer status;

    /** 玩家1ID */
    private Long player1Id;

    /** 玩家2ID */
    private Long player2Id;

    /** 获胜者ID（NULL表示平局） */
    private Long winnerId;

    /** 创建时间 */
    @TableField(fill = FieldFill.INSERT)
    @JsonIgnore
    private LocalDateTime createTime;

    /** 结束时间 */
    private LocalDateTime endedAt;
}
