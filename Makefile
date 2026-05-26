.PHONY: test build run migrate

DB_URL ?=

test:
	go test ./...

build:
	go build ./...

run:
	go run ./...

migrate:
	@test -n "$(DB_URL)" || (echo "DB_URL is required (example: postgres URL to your QuizArena database)" && exit 1)
	for f in database/migrations/*.sql; do \
		echo "Applying $$f"; \
		psql "$(DB_URL)" -v ON_ERROR_STOP=1 -f "$$f"; \
	done
