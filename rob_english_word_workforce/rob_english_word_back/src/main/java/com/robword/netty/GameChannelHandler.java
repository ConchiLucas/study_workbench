package com.robword.netty;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.robword.service.AnswerService;
import com.robword.service.GameService;
import com.robword.service.MatchService;
import com.robword.service.TrainingDifficultyCatalog;
import com.robword.state.PlayerState;
import com.robword.state.PlayerStateManager;
import com.robword.util.JwtUtil;
import io.netty.channel.ChannelHandler;
import io.netty.channel.ChannelHandlerContext;
import io.netty.channel.SimpleChannelInboundHandler;
import io.netty.handler.codec.http.websocketx.TextWebSocketFrame;
import io.netty.handler.timeout.IdleStateEvent;
import lombok.extern.slf4j.Slf4j;
import org.springframework.context.annotation.Lazy;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.stereotype.Component;

import java.net.URI;
import java.util.Map;
import java.util.Optional;

/**
 * Netty WebSocket 消息处理器
 * 替代 Spring WebSocket 的 GameWebSocketHandler
 * 所有状态决策由后端控制，前端只做渲染
 */
@Component
@ChannelHandler.Sharable
@Slf4j
public class GameChannelHandler extends SimpleChannelInboundHandler<TextWebSocketFrame> {

    private final ChannelManager channelManager;
    private final PlayerStateManager stateManager;
    private final MatchService matchService;
    private final GameService gameService;
    private final AnswerService answerService;
    private final JwtUtil jwtUtil;
    private final ObjectMapper objectMapper;
    private final RedisTemplate<String, Object> redisTemplate;
    private final TrainingDifficultyCatalog difficultyCatalog;

    public GameChannelHandler(ChannelManager channelManager,
                              PlayerStateManager stateManager,
                              @Lazy MatchService matchService,
                              @Lazy GameService gameService,
                              @Lazy AnswerService answerService,
                              JwtUtil jwtUtil,
                              ObjectMapper objectMapper,
                              RedisTemplate<String, Object> redisTemplate,
                              TrainingDifficultyCatalog difficultyCatalog) {
        this.channelManager = channelManager;
        this.stateManager = stateManager;
        this.matchService = matchService;
        this.gameService = gameService;
        this.answerService = answerService;
        this.jwtUtil = jwtUtil;
        this.objectMapper = objectMapper;
        this.redisTemplate = redisTemplate;
        this.difficultyCatalog = difficultyCatalog;
    }

    @Override
    public void userEventTriggered(ChannelHandlerContext ctx, Object evt) throws Exception {
        if (evt instanceof io.netty.handler.codec.http.websocketx.WebSocketServerProtocolHandler.HandshakeComplete handshake) {
            // WebSocket 握手完成，从 URI 提取 token 鉴权
            String uri = handshake.requestUri();
            String token = extractToken(uri);

            if (token == null || !jwtUtil.validateToken(token)) {
                log.warn("WebSocket auth failed, closing connection");
                ctx.close();
                return;
            }

            Long userId = jwtUtil.getUserIdFromToken(token);
            channelManager.register(userId, ctx.channel());
            PlayerState currentState = resolveStateOnConnect(userId);
            pushStateSnapshot(userId, currentState);
            log.info("User {} connected and authenticated", userId);

        } else if (evt instanceof IdleStateEvent) {
            // 心跳超时，关闭连接
            Long userId = channelManager.getUserId(ctx.channel());
            log.info("Idle timeout for user {}, closing", userId);
            ctx.close();
        }

        super.userEventTriggered(ctx, evt);
    }

    @Override
    protected void channelRead0(ChannelHandlerContext ctx, TextWebSocketFrame frame) {
        Long userId = channelManager.getUserId(ctx.channel());
        if (userId == null) {
            log.warn("Received message from unauthenticated channel, closing");
            ctx.close();
            return;
        }

        String payload = frame.text();

        try {
            @SuppressWarnings("unchecked")
            Map<String, Object> msg = objectMapper.readValue(payload, Map.class);
            String type = (String) msg.get("type");
            @SuppressWarnings("unchecked")
            Map<String, Object> data = (Map<String, Object>) msg.get("data");

            log.debug("Received from user {}: type={}", userId, type);

            switch (type) {
                case "match_start" -> handleMatchStart(userId, data);
                case "solo_training_start" -> handleSoloTrainingStart(userId, data);
                case "match_cancel" -> handleMatchCancel(userId);
                case "ping" -> handlePing(userId);
                case "sync_state" -> handleSyncState(userId);
                case "grab_word" -> handleGrabWord(userId, data);
                case "submit_answer" -> handleSubmitAnswer(userId, data);
                case "go_home" -> handleGoHome(userId);
                default -> {
                    log.warn("Unknown message type from user {}: {}", userId, type);
                    channelManager.sendToPlayer(userId, "error", Map.of("message", "Unknown type: " + type));
                }
            }
        } catch (Exception e) {
            log.error("Error handling message from user {}: {}", userId, e.getMessage());
            channelManager.sendToPlayer(userId, "error", Map.of("message", "Invalid message: " + e.getMessage()));
        }
    }

    @Override
    public void channelInactive(ChannelHandlerContext ctx) throws Exception {
        Long userId = channelManager.getUserId(ctx.channel());
        if (userId != null) {
            // 检查当前断开的 channel 是否仍然是该用户已注册的 channel
            // 如果用户已经重连（新 channel 已注册），则旧 channel 的断开不应清理状态
            if (!channelManager.isCurrentChannel(userId, ctx.channel())) {
                log.info("User {} old channel disconnected, skipping cleanup (already reconnected)", userId);
                super.channelInactive(ctx);
                return;
            }

            log.info("User {} disconnected", userId);

            // 获取断开前的状态
            PlayerState currentState = stateManager.getState(userId);

            // 如果在游戏中（GRABBING/ANSWERING），通知游戏服务处理断线
            if (currentState == PlayerState.GRABBING || currentState == PlayerState.ANSWERING) {
                gameService.handlePlayerDisconnect(userId);
            }

            // 匹配中断线保留状态和队列一小段时间，允许短暂重连后继续匹配
            if (currentState == PlayerState.MATCHING) {
                matchService.handleDisconnect(userId);
            } else {
                stateManager.clearState(userId);
            }
            channelManager.unregister(userId);
        }
        super.channelInactive(ctx);
    }

    @Override
    public void exceptionCaught(ChannelHandlerContext ctx, Throwable cause) {
        Long userId = channelManager.getUserId(ctx.channel());
        log.error("Exception for user {}: {}", userId, cause.getMessage());
        ctx.close();
    }

    // ==================== 消息处理方法 ====================

    void handleMatchStart(Long userId, Map<String, Object> data) {
        String difficultyGroup = data == null ? null : stringValue(data.get("difficultyGroup"));
        String difficultyLevel = data == null ? null : stringValue(data.get("difficultyLevel"));
        Optional<TrainingDifficultyCatalog.Difficulty> resolved =
                difficultyCatalog.resolve(difficultyGroup, difficultyLevel);
        if (resolved.isEmpty()) {
            channelManager.sendToPlayer(userId, "error", Map.of("message", "请选择有效的匹配难度"));
            return;
        }
        TrainingDifficultyCatalog.Difficulty difficulty = resolved.get();
        PlayerState currentState = stateManager.getState(userId);

        // 同难度重复请求幂等；等待中不允许静默切换到另一个队列。
        if (currentState == PlayerState.MATCHING) {
            Optional<TrainingDifficultyCatalog.Difficulty> currentPreference = matchService.getPreference(userId);
            if (currentPreference.isEmpty()) {
                stateManager.forceState(userId, PlayerState.IDLE);
                channelManager.sendToPlayer(userId, "error", Map.of("message", "匹配状态已失效，请重新匹配"));
                return;
            }
            if (!currentPreference.get().level().equals(difficulty.level())) {
                channelManager.sendToPlayer(userId, "error", Map.of("message", "请先取消当前匹配"));
                return;
            }
            matchService.handleReconnect(userId);
            sendMatchWaiting(userId, currentPreference.get());
            return;
        }

        // 状态为 null（Redis key 过期/被清理，常见于断线重连场景），先初始化为 IDLE
        if (currentState == null) {
            log.info("User {} state is null (likely reconnect race), initializing to IDLE", userId);
            stateManager.forceState(userId, PlayerState.IDLE);
            currentState = PlayerState.IDLE;
        }

        // 如果状态是 FINISHED/MATCHED 等残留状态（如游戏结束但没点返回首页），先强制复位为 IDLE
        if (currentState != PlayerState.IDLE) {
            log.info("User {} in non-IDLE state {} when trying to match, resetting to IDLE first", userId, currentState);
            // 如果在游戏中，先清理游戏资源
            if (currentState == PlayerState.GRABBING || currentState == PlayerState.ANSWERING) {
                gameService.handlePlayerDisconnect(userId);
            }
            stateManager.forceState(userId, PlayerState.IDLE);
        }

        // CAS: IDLE → MATCHING
        boolean transitioned = stateManager.transition(userId, PlayerState.IDLE, PlayerState.MATCHING);
        if (!transitioned) {
            log.error("CAS IDLE→MATCHING failed for user {} even after reset, actual state: {}",
                    userId, stateManager.getState(userId));
            channelManager.sendToPlayer(userId, "error", Map.of("message", "匹配失败，请重试"));
            return;
        }

        if (!matchService.addToMatchQueue(userId, difficulty)) {
            stateManager.forceState(userId, PlayerState.IDLE);
            return;
        }
        channelManager.sendToPlayer(userId, "state_change", Map.of("state", "MATCHING"));
        sendMatchWaiting(userId, difficulty);
    }

    private void handleSoloTrainingStart(Long userId, Map<String, Object> data) {
        PlayerState currentState = stateManager.getState(userId);
        if (currentState == null) {
            stateManager.forceState(userId, PlayerState.IDLE);
            currentState = PlayerState.IDLE;
        }

        if (currentState == PlayerState.MATCHING) {
            matchService.removeFromMatchQueue(userId);
        } else if (currentState == PlayerState.GRABBING || currentState == PlayerState.ANSWERING) {
            gameService.handlePlayerDisconnect(userId);
        }

        if (currentState != PlayerState.IDLE) {
            stateManager.forceState(userId, PlayerState.IDLE);
        }

        String difficultyGroup = data == null ? null : stringValue(data.get("difficultyGroup"));
        String difficultyLevel = data == null ? null : stringValue(data.get("difficultyLevel"));
        gameService.startSoloTraining(userId, difficultyGroup, difficultyLevel);
    }

    private String stringValue(Object value) {
        return value instanceof String string ? string : null;
    }

    void handlePing(Long userId) {
        stateManager.refreshStateTtl(userId);
        channelManager.sendToPlayer(userId, "pong", Map.of());
    }

    void handleSyncState(Long userId) {
        PlayerState currentState = stateManager.getState(userId);
        if (currentState == null) {
            currentState = PlayerState.IDLE;
            stateManager.forceState(userId, currentState);
        } else {
            stateManager.refreshStateTtl(userId);
        }
        pushStateSnapshot(userId, currentState);
        if (currentState == PlayerState.GRABBING || currentState == PlayerState.ANSWERING) {
            gameService.resumeGame(userId);
        }
    }

    PlayerState resolveStateOnConnect(Long userId) {
        PlayerState currentState = stateManager.getState(userId);
        if (currentState == null) {
            stateManager.forceState(userId, PlayerState.IDLE);
            return PlayerState.IDLE;
        }
        if (currentState == PlayerState.MATCHING) {
            if (matchService.handleReconnect(userId).isEmpty()) {
                return PlayerState.IDLE;
            }
        }
        return currentState;
    }

    void pushStateSnapshot(Long userId, PlayerState state) {
        if (state == PlayerState.MATCHING) {
            Optional<TrainingDifficultyCatalog.Difficulty> preference = matchService.getPreference(userId);
            if (preference.isEmpty()) {
                stateManager.forceState(userId, PlayerState.IDLE);
                channelManager.sendToPlayer(userId, "state_change", Map.of("state", "IDLE"));
                return;
            }
            channelManager.sendToPlayer(userId, "state_change", Map.of("state", "MATCHING"));
            sendMatchWaiting(userId, preference.get());
            return;
        }
        channelManager.sendToPlayer(userId, "state_change", Map.of("state", state.name()));
    }

    private void sendMatchWaiting(Long userId, TrainingDifficultyCatalog.Difficulty difficulty) {
        channelManager.sendToPlayer(userId, "match_waiting", Map.of(
                "difficultyGroup", difficulty.group(),
                "difficultyLevel", difficulty.level(),
                "difficultyLabel", difficulty.label()
        ));
    }

    private void handleMatchCancel(Long userId) {
        PlayerState currentState = stateManager.getState(userId);

        // 如果已经是 IDLE 或 null，说明取消已经生效（可能是断线重连导致的），直接确认
        if (currentState == null || currentState == PlayerState.IDLE) {
            log.info("User {} cancel match but already in {} state, confirming IDLE", userId, currentState);
            if (currentState == null) {
                stateManager.forceState(userId, PlayerState.IDLE);
            }
            matchService.removeFromMatchQueue(userId); // 防御性清理
            channelManager.sendToPlayer(userId, "state_change", Map.of("state", "IDLE"));
            return;
        }

        // CAS: MATCHING → IDLE
        boolean transitioned = stateManager.transition(userId, PlayerState.MATCHING, PlayerState.IDLE);
        if (!transitioned) {
            log.warn("Cancel match CAS failed for user {}, current state: {}, force resetting", userId, currentState);
            // 非正常状态也允许取消，强制回到 IDLE
            matchService.removeFromMatchQueue(userId);
            stateManager.forceState(userId, PlayerState.IDLE);
            channelManager.sendToPlayer(userId, "state_change", Map.of("state", "IDLE"));
            return;
        }

        matchService.removeFromMatchQueue(userId);
        channelManager.sendToPlayer(userId, "state_change", Map.of("state", "IDLE"));
    }

    private void handleGrabWord(Long userId, Map<String, Object> data) {
        // 验证状态
        PlayerState state = stateManager.getState(userId);
        if (state != PlayerState.GRABBING) {
            channelManager.sendToPlayer(userId, "error", Map.of("message", "当前状态不允许抢词"));
            return;
        }

        if (data == null) {
            channelManager.sendToPlayer(userId, "grab_error", Map.of("error", "Missing data"));
            return;
        }

        Long wordId = parseId(data.get("wordId"));
        if (wordId == null) {
            channelManager.sendToPlayer(userId, "grab_error", Map.of("error", "Invalid wordId"));
            return;
        }

        gameService.grabWord(userId, wordId);
    }

    private void handleSubmitAnswer(Long userId, Map<String, Object> data) {
        // 验证状态
        PlayerState state = stateManager.getState(userId);
        if (state != PlayerState.ANSWERING) {
            channelManager.sendToPlayer(userId, "error", Map.of("message", "当前状态不允许答题"));
            return;
        }

        if (data == null) {
            channelManager.sendToPlayer(userId, "error", Map.of("message", "Missing data"));
            return;
        }

        Long roomId = parseId(data.get("roomId"));
        Integer selectedIndex = data.get("selectedIndex") != null
                ? ((Number) data.get("selectedIndex")).intValue()
                : null;

        if (roomId == null || selectedIndex == null) {
            channelManager.sendToPlayer(userId, "error", Map.of("message", "Invalid roomId or selectedIndex"));
            return;
        }

        answerService.submitAnswer(userId, roomId, selectedIndex);
    }

    private void handleGoHome(Long userId) {
        PlayerState state = stateManager.getState(userId);

        if (state == null) {
            // 状态已被清理（断线重连等），直接初始化为 IDLE
            stateManager.forceState(userId, PlayerState.IDLE);
        } else if (state == PlayerState.IDLE) {
            // 已经是 IDLE，无需操作
        } else if (state == PlayerState.FINISHED) {
            stateManager.transition(userId, PlayerState.FINISHED, PlayerState.IDLE);
        } else if (state == PlayerState.MATCHING) {
            // 匹配中返回首页，需要从匹配队列移除
            matchService.removeFromMatchQueue(userId);
            stateManager.forceState(userId, PlayerState.IDLE);
        } else {
            // MATCHED / GRABBING / ANSWERING — 游戏中途退出，清理游戏资源
            if (state == PlayerState.GRABBING || state == PlayerState.ANSWERING) {
                gameService.handlePlayerDisconnect(userId);
            }
            stateManager.forceState(userId, PlayerState.IDLE);
        }

        channelManager.sendToPlayer(userId, "state_change", Map.of("state", "IDLE"));
    }

    // ==================== 工具方法 ====================

    private String extractToken(String uri) {
        try {
            URI u = new URI(uri);
            String query = u.getQuery();
            if (query != null) {
                for (String param : query.split("&")) {
                    String[] kv = param.split("=", 2);
                    if (kv.length == 2 && "token".equals(kv[0])) {
                        return kv[1];
                    }
                }
            }
        } catch (Exception e) {
            log.error("Failed to parse URI: {}", uri, e);
        }
        return null;
    }

    private Long parseId(Object obj) {
        if (obj == null) return null;
        if (obj instanceof Number) return ((Number) obj).longValue();
        try {
            return Long.parseLong(obj.toString());
        } catch (NumberFormatException e) {
            return null;
        }
    }
}
