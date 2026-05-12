GOCACHE ?= /tmp/mlakp-go-build
ENV_FILE ?= .env

.PHONY: run test test-integration fmt vet check sqlc openapi migrate-up migrate-down reset-db

run:
	@if [ ! -f "$(ENV_FILE)" ]; then \
		echo "$(ENV_FILE) not found. Create it with: cp .env.example $(ENV_FILE)"; \
		exit 1; \
	fi
	@set -a; . ./$(ENV_FILE); set +a; GOCACHE=$(GOCACHE) go run ./cmd/api

test:
	GOCACHE=$(GOCACHE) go test ./...

test-integration:
	@if [ -f "$(ENV_FILE)" ]; then set -a; . ./$(ENV_FILE); set +a; fi; \
	if [ -z "$$MLAKP_TEST_DATABASE_URL" ] && [ -z "$$DATABASE_URL" ]; then \
		echo "Set MLAKP_TEST_DATABASE_URL or DATABASE_URL to run PostgreSQL integration tests"; \
		exit 1; \
	fi; \
	GOCACHE=$(GOCACHE) go test ./internal/payments -run 'TestRepository'

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './internal/postgres/sqlc/*')

vet:
	GOCACHE=$(GOCACHE) go vet ./...

check: openapi fmt vet test

sqlc:
	sqlc generate

openapi:
	GOCACHE=$(GOCACHE) go run ./scripts/openapi

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

reset-db:
	@if [ "$(CONFIRM_RESET)" != "reset" ]; then \
		echo "This drops all database objects and data. Run: make reset-db CONFIRM_RESET=reset"; \
		exit 1; \
	fi
	@if [ ! -f "$(ENV_FILE)" ]; then \
		echo "$(ENV_FILE) not found. Create it with: cp .env.example $(ENV_FILE)"; \
		exit 1; \
	fi
	@set -a; . ./$(ENV_FILE); set +a; \
	if [ -z "$$DATABASE_URL" ]; then \
		echo "DATABASE_URL is required"; \
		exit 1; \
	fi; \
	psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -c "DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public; GRANT ALL ON SCHEMA public TO public;"
