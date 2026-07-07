# FinHelper Monorepo — Go backend + Vite/React frontend

.PHONY: all build-backend build-frontend build test-backend test-frontend test \
        dev-backend dev-frontend dev docker-up docker-down clean

# ===== Default =====
all: build test

# ===== Backend (Go) =====
build-backend:
	cd backend && "C:/Program Files/Go/bin/go.exe" build ./cmd/server/

test-backend:
	cd backend && "C:/Program Files/Go/bin/go.exe" test ./...

vet-backend:
	cd backend && "C:/Program Files/Go/bin/go.exe" vet ./...

dev-backend:
	cd backend && "C:/Program Files/Go/bin/go.exe" run ./cmd/server/

# ===== Frontend (Vite/React) =====
build-frontend:
	cd frontend && npm run build

test-frontend:
	cd frontend && npx vitest run

lint-frontend:
	cd frontend && npx tsc -b

install-frontend:
	cd frontend && npm ci

dev-frontend:
	cd frontend && npm run dev

# ===== Combined =====
build: build-backend build-frontend

test: test-backend test-frontend

vet: vet-backend lint-frontend

# ===== Docker =====
docker-up:
	docker compose up -d

docker-down:
	docker compose down

# ===== Clean =====
clean:
	cd backend && "C:/Program Files/Go/bin/go.exe" clean ./...
	rm -rf frontend/dist
