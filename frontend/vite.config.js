import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react-swc'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
        ws: true,  // 关键！启用 WebSocket 代理
        // rewrite: (path) => path.replace(/^\/api/, ''), // 根据需要开启
      },
    },
  },
})
