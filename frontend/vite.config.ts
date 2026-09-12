import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// `npm run dev` proxies relative /v1, /health, /healthz calls here (api.ts's
// API_BASE is '' in dev since VITE_API_URL only loads from .env.production).
// Defaults to the local Go gateway (`make dev-backend`, :8080). Override with
// DEV_BACKEND=https://parallax-n4it.onrender.com npm run dev to point at prod
// without touching this file. This only affects `vite dev` — production
// builds ignore it and call VITE_API_URL directly (see .env.production).
const BACKEND = process.env.DEV_BACKEND || 'http://localhost:8080'

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
