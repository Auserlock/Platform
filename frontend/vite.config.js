import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react-swc'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  // 在这里添加 server 配置
  server: {
    proxy: {
      // 关键在于这个 '/api' 字符串
      // 它会拦截所有以 /api 开头的请求，例如 /api/v1/tasks
      '/api': {
        // 将请求转发到你的后端服务器地址
        target: 'http://localhost:8080',
        
        // changeOrigin: true 对于虚拟主机站点是必需的，
        // 它会将请求头中的 Host 字段从开发服务器的地址修改为目标地址
        // 强烈建议加上
        changeOrigin: true,
        ws: true,
        // 如果你的后端API路径中没有/api，你可能需要重写路径
        // 但根据你的情况，后端路径是 /api/v1/tasks，所以不需要重写
        // rewrite: (path) => path.replace(/^\/api/, '') 
      }
    }
  }
})