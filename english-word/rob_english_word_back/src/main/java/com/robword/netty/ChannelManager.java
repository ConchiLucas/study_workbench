package com.robword.netty;

import io.netty.channel.Channel;
import io.netty.channel.group.ChannelGroup;
import io.netty.channel.group.DefaultChannelGroup;
import io.netty.handler.codec.http.websocketx.TextWebSocketFrame;
import io.netty.util.AttributeKey;
import io.netty.util.concurrent.GlobalEventExecutor;
import com.fasterxml.jackson.databind.ObjectMapper;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;

import java.util.Map;
import java.util.Set;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;

/**
 * Channel 管理器
 * 管理 userId ↔ Channel 映射，提供消息发送工具方法
 */
@Component
@Slf4j
public class ChannelManager {

    public static final AttributeKey<Long> USER_ID_KEY = AttributeKey.valueOf("userId");
    public static final AttributeKey<String> SESSION_ID_KEY = AttributeKey.valueOf("sessionId");

    private final Map<Long, Channel> userChannels = new ConcurrentHashMap<>();
    private final Map<Long, String> userSessions = new ConcurrentHashMap<>();
    private final ChannelGroup allChannels = new DefaultChannelGroup(GlobalEventExecutor.INSTANCE);
    private final ObjectMapper objectMapper;

    public ChannelManager(ObjectMapper objectMapper) {
        this.objectMapper = objectMapper;
    }

    /**
     * 注册用户连接
     */
    public void register(Long userId, Channel channel) {
        String sessionId = UUID.randomUUID().toString();
        channel.attr(USER_ID_KEY).set(userId);
        channel.attr(SESSION_ID_KEY).set(sessionId);

        Channel oldChannel = userChannels.put(userId, channel);
        String oldSessionId = userSessions.put(userId, sessionId);

        if (oldChannel != null && oldChannel.isActive()) {
            log.info("Replacing existing channel for user {}: oldSession={}, newSession={}", userId, oldSessionId, sessionId);
            sendToChannel(oldChannel, "duplicate_login", Map.of("message", "账号已在其他地方登录"));
            oldChannel.close();
        }

        allChannels.add(channel);
        log.info("User {} registered with session {}, total online: {}", userId, sessionId, userChannels.size());
    }

    /**
     * 注销用户连接
     */
    public void unregister(Long userId) {
        Channel removed = userChannels.remove(userId);
        userSessions.remove(userId);
        if (removed != null) {
            allChannels.remove(removed);
        }
        log.info("User {} unregistered, total online: {}", userId, userChannels.size());
    }

    /**
     * 获取用户 Channel
     */
    public Channel getChannel(Long userId) {
        return userChannels.get(userId);
    }

    /**
     * 获取 Channel 关联的 userId
     */
    public Long getUserId(Channel channel) {
        return channel.attr(USER_ID_KEY).get();
    }

    /**
     * 发送消息给单个玩家
     * @return true 发送成功，false 连接不存在或已关闭
     */
    public boolean sendToPlayer(Long userId, String type, Object data) {
        Channel channel = userChannels.get(userId);
        if (channel == null || !channel.isActive()) {
            log.warn("Cannot send to user {}: channel is null or inactive", userId);
            return false;
        }

        return sendToChannel(channel, type, data);
    }

    private boolean sendToChannel(Channel channel, String type, Object data) {
        try {
            Map<String, Object> message = Map.of("type", type, "data", data);
            String json = objectMapper.writeValueAsString(message);
            channel.writeAndFlush(new TextWebSocketFrame(json));
            return true;
        } catch (Exception e) {
            Long userId = getUserId(channel);
            log.error("Failed to send message to user {}: {}", userId, e.getMessage());
            return false;
        }
    }

    /**
     * 广播消息给房间内所有玩家
     */
    public void broadcastToRoom(Set<Long> playerIds, String type, Object data, Long excludeUserId) {
        for (Long playerId : playerIds) {
            if (excludeUserId != null && excludeUserId.equals(playerId)) {
                continue;
            }
            sendToPlayer(playerId, type, data);
        }
    }

    /**
     * 检查用户是否在线
     */
    public boolean isOnline(Long userId) {
        Channel channel = userChannels.get(userId);
        return channel != null && channel.isActive();
    }

    /**
     * 检查指定 Channel 是否仍然是该用户当前注册的 Channel
     * 用于防止旧连接断开时误清理新连接的状态
     */
    public boolean isCurrentChannel(Long userId, Channel channel) {
        Channel current = userChannels.get(userId);
        String currentSessionId = userSessions.get(userId);
        String channelSessionId = getSessionId(channel);
        return current != null && current == channel && currentSessionId != null && currentSessionId.equals(channelSessionId);
    }

    public String getSessionId(Channel channel) {
        return channel.attr(SESSION_ID_KEY).get();
    }
}
