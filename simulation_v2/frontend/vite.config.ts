import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5174,
    proxy: {
      '/api': 'http://127.0.0.1:8001',
      '/ws': { target: 'ws://127.0.0.1:8001', ws: true },
      '/status': 'http://127.0.0.1:8001',
      '/block': 'http://127.0.0.1:8001',
      '/traffic': {
        target: 'http://127.0.0.1:8002',
        rewrite: (path) => path.replace(/^\/traffic/, ''),
      },
    },
  },
})
