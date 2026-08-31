import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  build: {
    chunkSizeWarningLimit: 900,
    rollupOptions: {
      output: {
        manualChunks: {
          antd: ["antd", "@ant-design/icons"],
          query: ["@tanstack/react-query"],
        },
      },
    },
  },
  server: {
    port: 6016,
    proxy: {
      "/api": {
        target: "http://127.0.0.1:6015",
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ""),
      },
      "/ai-file-navigation": {
        target: "http://127.0.0.1:6015",
        changeOrigin: true,
      },
    },
  },
});
