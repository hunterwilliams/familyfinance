.PHONY: test dev-api dev-web docker-up docker-down docker-prod

test:
	cd backend && go test ./...

# Run API on host (fastest hot reload)
dev-api:
	cd backend && DATABASE_URL="postgres://familyfinance:familyfinance@localhost:5432/familyfinance?sslmode=disable" API_KEY=dev-api-key JWT_SECRET=dev-jwt-secret SKIP_AUTH=true air

# Run web on host
dev-web:
	cd web && npm run dev

# Local dev — override.yml auto-merges (Air + Vite in Docker)
docker-up:
	docker compose up --build

docker-down:
	docker compose down

# Production only — no override merge
docker-prod:
	docker compose -f docker-compose.yml --project-directory . up --build -d
