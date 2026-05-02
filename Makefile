GOCACHE ?= /tmp/mlakp-go-build
ENV_FILE ?= .env

.PHONY: run test fmt vet check sqlc migrate-up migrate-down

run:
	@if [ ! -f "$(ENV_FILE)" ]; then \
		echo "$(ENV_FILE) not found. Create it with: cp .env.example $(ENV_FILE)"; \
		exit 1; \
	fi
	@set -a; . ./$(ENV_FILE); set +a; GOCACHE=$(GOCACHE) go run ./cmd/api

test:
	GOCACHE=$(GOCACHE) go test ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './internal/postgres/sqlc/*')

vet:
	GOCACHE=$(GOCACHE) go vet ./...

check: fmt vet test

sqlc:
	sqlc generate

migrate-up:
	@if [ ! -f "$(ENV_FILE)" ]; then \
		echo "$(ENV_FILE) not found. Create it with: cp .env.example $(ENV_FILE)"; \
		exit 1; \
	fi
	@set -a; . ./$(ENV_FILE); set +a; migrate -path migrations -database "$$DATABASE_URL" up

migrate-down:
	@if [ ! -f "$(ENV_FILE)" ]; then \
		echo "$(ENV_FILE) not found. Create it with: cp .env.example $(ENV_FILE)"; \
		exit 1; \
	fi
	@set -a; . ./$(ENV_FILE); set +a; migrate -path migrations -database "$$DATABASE_URL" down 1
