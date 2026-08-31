import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 6014,
    proxy: {
      "/api": {
        target: "http://127.0.0.1:6012",
        changeOrigin: true,
      },
      "/ai-file-navigation": {
        target: "http://127.0.0.1:6015",
        changeOrigin: true,
      },
    },
  },
});
