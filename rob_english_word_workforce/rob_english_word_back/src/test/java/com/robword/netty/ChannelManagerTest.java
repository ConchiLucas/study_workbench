package com.robword.netty;

import com.fasterxml.jackson.databind.ObjectMapper;
import io.netty.channel.embedded.EmbeddedChannel;
import io.netty.handler.codec.http.websocketx.TextWebSocketFrame;
import org.junit.jupiter.api.Test;

import java.util.Map;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertNotEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertTrue;

class ChannelManagerTest {

    @Test
    void shouldCloseOldSessionWithDuplicateLoginMessageWhenSameUserReconnects() throws Exception {
        ChannelManager channelManager = new ChannelManager(new ObjectMapper());
        EmbeddedChannel oldChannel = new EmbeddedChannel();
        EmbeddedChannel newChannel = new EmbeddedChannel();

        channelManager.register(1L, oldChannel);
        String oldSessionId = channelManager.getSessionId(oldChannel);

        channelManager.register(1L, newChannel);

        TextWebSocketFrame frame = oldChannel.readOutbound();
        assertNotNull(frame);

        @SuppressWarnings("unchecked")
        Map<String, Object> payload = new ObjectMapper().readValue(frame.text(), Map.class);
        assertEquals("duplicate_login", payload.get("type"));
        assertEquals(newChannel, channelManager.getChannel(1L));
        assertFalse(oldChannel.isActive());
        assertTrue(newChannel.isActive());
        assertNotEquals(oldSessionId, channelManager.getSessionId(newChannel));
    }

    @Test
    void shouldTreatOnlyLatestSessionAsCurrent() {
        ChannelManager channelManager = new ChannelManager(new ObjectMapper());
        EmbeddedChannel oldChannel = new EmbeddedChannel();
        EmbeddedChannel newChannel = new EmbeddedChannel();

        channelManager.register(1L, oldChannel);
        channelManager.register(1L, newChannel);

        assertFalse(channelManager.isCurrentChannel(1L, oldChannel));
        assertTrue(channelManager.isCurrentChannel(1L, newChannel));
    }
}
