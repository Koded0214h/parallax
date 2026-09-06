.PHONY: dev-backend dev-frontend build test tidy

# Run the Go gateway on :8080 (override with PARALLAX_ADDR).
dev-backend:
	cd backend && go run ./cmd/gateway

# Run the Vite dev server on :5173 (proxies /v1 and /healthz to :8080).
dev-frontend:
	cd frontend && npm run dev

build:
	cd backend && go build ./...
	cd frontend && npm run build

test:
	cd backend && go test ./...

tidy:
	cd backend && go mod tidy
