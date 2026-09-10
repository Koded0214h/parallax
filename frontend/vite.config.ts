import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// Production backend is live at https://parallax-n4it.onrender.com
// VITE_API_URL is set in .env so API_BASE uses direct cross-origin calls (CORS: *)
// The proxy below is a safety net for any relative-path calls that slip through.
const BACKEND = 'https://parallax-n4it.onrender.com'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/v1': {
        target: BACKEND,
        changeOrigin: true,
      },
      '/health': {
        target: BACKEND,
        changeOrigin: true,
      },
      '/healthz': {
        target: BACKEND,
        changeOrigin: true,
      },
    },
  },
})
