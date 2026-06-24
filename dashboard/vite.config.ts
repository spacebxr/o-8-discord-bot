import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: 'https://o-8-dashboard.up.railway.app',
        changeOrigin: true,
        secure: true,
      },
    },
  },
})
