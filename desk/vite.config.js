import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: process.env.UNITS_URL || 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
