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
      '/v1': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        configure: (proxy) => {
          proxy.on('error', (_err, _req, res) => {
            // Silently handle proxy errors when the Go backend is offline
            if ('writeHead' in res && !res.headersSent) {
              res.writeHead(503, { 'Content-Type': 'application/json' })
              res.end(JSON.stringify({ error: 'Gateway offline' }))
            }
          })
        },
      },
      '/healthz': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        configure: (proxy) => {
          proxy.on('error', (_err, _req, res) => {
            // Silently handle proxy errors when the Go backend is offline
            if ('writeHead' in res && !res.headersSent) {
              res.writeHead(503, { 'Content-Type': 'application/json' })
              res.end(JSON.stringify({ status: 'unreachable' }))
            }
          })
        },
      },
    },
  },
})
