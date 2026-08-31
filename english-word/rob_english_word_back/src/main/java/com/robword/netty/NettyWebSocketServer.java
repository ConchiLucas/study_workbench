package com.robword.netty;

import io.netty.bootstrap.ServerBootstrap;
import io.netty.channel.ChannelFuture;
import io.netty.channel.EventLoopGroup;
import io.netty.channel.nio.NioEventLoopGroup;
import io.netty.channel.socket.nio.NioServerSocketChannel;
import jakarta.annotation.PostConstruct;
import jakarta.annotation.PreDestroy;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

/**
 * Netty WebSocket 服务端
 * 独立端口运行，Spring Boot 启动时自动拉起
 */
@Component
@RequiredArgsConstructor
@Slf4j
public class NettyWebSocketServer {

    private final NettyWebSocketInitializer initializer;

    @Value("${netty.websocket.port:9091}")
    private int port;

    private EventLoopGroup bossGroup;
    private EventLoopGroup workerGroup;
    private ChannelFuture channelFuture;

    @PostConstruct
    public void start() {
        new Thread(this::doStart, "netty-ws-server").start();
    }

    private void doStart() {
        bossGroup = new NioEventLoopGroup(1);
        workerGroup = new NioEventLoopGroup();

        try {
            ServerBootstrap bootstrap = new ServerBootstrap();
            bootstrap.group(bossGroup, workerGroup)
                    .channel(NioServerSocketChannel.class)
                    .childHandler(initializer);

            channelFuture = bootstrap.bind(port).sync();
            log.info("Netty WebSocket Server started on port {}", port);
            channelFuture.channel().closeFuture().sync();
        } catch (InterruptedException e) {
            log.error("Netty WebSocket Server interrupted", e);
            Thread.currentThread().interrupt();
        } finally {
            shutdown();
        }
    }

    @PreDestroy
    public void shutdown() {
        log.info("Shutting down Netty WebSocket Server...");
        if (bossGroup != null) {
            bossGroup.shutdownGracefully();
        }
        if (workerGroup != null) {
            workerGroup.shutdownGracefully();
        }
    }
}
