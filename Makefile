.PHONY: dev prod build clean

# Dev
dev:
	docker compose up --build

dev-down:
	docker compose down

dev-logs:
	docker compose logs -f csupgrade-api

# Production
prod:
	docker compose up -d --build

prod-down:
	docker compose down

clean:
	docker compose down -v
	docker system prune -f
