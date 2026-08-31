import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 19092,
    proxy: { '/api': 'http://localhost:19091', '/healthz': 'http://localhost:19091' },
  },
  build: {
    outDir: '../backend/internal/http/dist',
    emptyOutDir: true,
  },
})
