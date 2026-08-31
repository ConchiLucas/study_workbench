package com.robword.netty;

import io.netty.channel.ChannelInitializer;
import io.netty.channel.ChannelPipeline;
import io.netty.channel.socket.SocketChannel;
import io.netty.handler.codec.http.HttpObjectAggregator;
import io.netty.handler.codec.http.HttpServerCodec;
import io.netty.handler.codec.http.websocketx.WebSocketServerProtocolHandler;
import io.netty.handler.stream.ChunkedWriteHandler;
import io.netty.handler.timeout.IdleStateHandler;
import lombok.RequiredArgsConstructor;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.util.concurrent.TimeUnit;

/**
 * Netty WebSocket Pipeline 初始化器
 */
@Component
@RequiredArgsConstructor
public class NettyWebSocketInitializer extends ChannelInitializer<SocketChannel> {

    private final GameChannelHandler gameChannelHandler;

    @Value("${netty.websocket.read-idle-seconds:300}")
    private int readIdleSeconds;

    @Override
    protected void initChannel(SocketChannel ch) {
        ChannelPipeline pipeline = ch.pipeline();

        // HTTP 编解码
        pipeline.addLast(new HttpServerCodec());
        pipeline.addLast(new ChunkedWriteHandler());
        pipeline.addLast(new HttpObjectAggregator(65536));

        // 心跳检测：超过配置时间未收到客户端消息则触发
        pipeline.addLast(new IdleStateHandler(readIdleSeconds, 0, 0, TimeUnit.SECONDS));

        // WebSocket 协议处理（不在这里做鉴权，手动处理握手前的 token 校验）
        // checkStartsWith=true（最后一个参数），使 /ws?token=xxx 能匹配 /ws 路径
        pipeline.addLast(new WebSocketServerProtocolHandler("/ws", null, true, 65536, false, true));

        // 业务 Handler（Sharable，单例）
        pipeline.addLast(gameChannelHandler);
    }
}
