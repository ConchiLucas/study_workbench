import { defineConfig } from 'vitest/config';
import vue from '@vitejs/plugin-vue';
export default defineConfig({
    plugins: [vue()],
    test: {
        environment: 'jsdom'
    },
    server: {
        host: '0.0.0.0',
        port: 6011,
        proxy: {
            '/api': {
                target: 'http://localhost:6012',
                changeOrigin: true
            },
            '/ws': {
                target: 'ws://localhost:6013',
                ws: true
            }
        }
    }
});
