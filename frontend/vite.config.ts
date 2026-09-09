import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// The gateway runs on :8080 by default (see backend/cmd/gateway).
// Dev requests to /v1/* and /healthz are proxied there so the frontend
// can use same-origin relative URLs.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/v1': 'http://localhost:8080',
      '/health': 'http://localhost:8080',
      '/healthz': 'http://localhost:8080',
    },
  },
})
