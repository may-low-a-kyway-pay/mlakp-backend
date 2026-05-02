# MLAKP Backend

Go backend for the MLAKP shared expense API. The current implementation exposes health checks, OpenAPI documentation, user registration, login, logout, and the authenticated current-user endpoint.

## Requirements

- Go `1.26.1` or newer, matching `go.mod`
- PostgreSQL
- Git
- `golang-migrate` CLI for database migrations
- `sqlc` CLI only when regenerating database query code

The project intentionally keeps Go build cache outside the home directory through the `Makefile`:

```sh
GOCACHE ?= /tmp/mlakp-go-build
```

That avoids local permission issues in restricted environments.

## Fresh Setup From Git Clone

Clone the repository:

```sh
git clone git@github.com:may-low-a-kway-pay/mlakp-backend.git
```

Enter the project:

```sh
cd mlakp-backend
```

Download Go dependencies:

```sh
go mod download
```

Install the migration CLI used by `make migrate-up`:

```sh
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Verify it is available in your shell:

```sh
command -v migrate
migrate -version
```

If `command -v migrate` prints nothing, make sure your Go binary directory is in `PATH`:

```sh
export PATH="$(go env GOPATH)/bin:$PATH"
```

Add that `PATH` export to your shell profile if needed.

Install `sqlc` only if you need to regenerate database query code:

```sh
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

Create your local environment file from the example:

```sh
cp .env.example .env
```

Open `.env` and review the values before running the app:

```env
APP_ENV=local
APP_PORT=8080
DATABASE_URL=postgres://mlakp:mlakp@localhost:5432/mlakp?sslmode=disable
TOKEN_ISSUER=mlakp-backend
TOKEN_AUDIENCE=mlakp-api
TOKEN_SECRET=change-me-local-development-secret
ACCESS_TOKEN_TTL=15m
READ_TIMEOUT=5s
WRITE_TIMEOUT=10s
IDLE_TIMEOUT=60s
SHUTDOWN_TIMEOUT=10s
```

The default `DATABASE_URL` expects this local PostgreSQL database:

```text
host: localhost
port: 5432
database: mlakp
user: mlakp
password: mlakp
```

Important environment rules enforced by `internal/config`:

- `APP_ENV` must be `local`, `test`, or `production`.
- `APP_PORT` must be a valid TCP port.
- `DATABASE_URL` must use `postgres://` or `postgresql://` and include a database name.
- `TOKEN_SECRET` is required. In production it must be at least 32 bytes.
- Timeout values must be valid Go durations such as `15m`, `5s`, or `1h`.

After `.env` is ready, continue with database setup and migrations below.

## Database Setup

Create a local PostgreSQL user and database matching `.env`.

Using `psql` as a PostgreSQL admin user:

```sh
psql postgres
```

Then run:

```sql
CREATE USER mlakp WITH PASSWORD 'mlakp';
CREATE DATABASE mlakp OWNER mlakp;
\q
```

If the user or database already exists, skip the matching command or adjust the values in `.env`.

Run migrations:

```sh
make migrate-up
```

The first migration creates:

- `pgcrypto` extension for UUID generation
- `users` table
- lowercase email constraint
- user name length constraint
- `updated_at` trigger

To roll back one migration:

```sh
make migrate-down
```

## Run The API

Start the server:

```sh
make run
```

The server listens on:

```text
http://localhost:8080
```

If you changed `APP_PORT`, use that port instead.

The app connects to PostgreSQL during startup. If PostgreSQL is down, the database does not exist, credentials are wrong, or migrations have not run, startup will fail before the HTTP server is available.

## Verify The App

In another terminal, check process health:

```sh
curl -i http://localhost:8080/healthz
```

Expected response:

```json
{"status":"ok"}
```

Check readiness:

```sh
curl -i http://localhost:8080/readyz
```

Expected response when PostgreSQL is reachable:

```json
{"status":"ready"}
```

Open Swagger UI:

```text
http://localhost:8080/docs
```

Open the raw OpenAPI spec:

```text
http://localhost:8080/docs/openapi.yaml
```

## Auth Smoke Test

Register a user:

```sh
curl -s -X POST http://localhost:8080/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Thomas",
    "email": "thomas@example.com",
    "password": "password123"
  }'
```

The response includes an `access_token`:

```json
{
  "access_token": "...",
  "token_type": "Bearer",
  "expires_at": "2026-05-02T12:15:00Z",
  "user": {
    "id": "...",
    "name": "Thomas",
    "email": "thomas@example.com"
  }
}
```

Login with the same user:

```sh
curl -s -X POST http://localhost:8080/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "thomas@example.com",
    "password": "password123"
  }'
```

Store the token from either register or login:

```sh
TOKEN='paste-access-token-here'
```

Call the authenticated current-user endpoint:

```sh
curl -s http://localhost:8080/v1/users/me \
  -H "Authorization: Bearer $TOKEN"
```

Logout is currently client-side token discard. The backend returns `204` and does not persist sessions or token denylists yet:

```sh
curl -i -X POST http://localhost:8080/v1/auth/logout
```

## Development Commands

Run all tests:

```sh
make test
```

Format hand-written Go code:

```sh
make fmt
```

Run `go vet`:

```sh
make vet
```

Run format, vet, and tests:

```sh
make check
```

Regenerate SQL code after changing files in `queries/` or `migrations/`:

```sh
make sqlc
```

Generated SQL code is written to:

```text
internal/postgres/sqlc/
```

## Project Layout

```text
cmd/api/main.go                 Application entrypoint and HTTP server startup
internal/app/routes.go          Route registration, health/readiness, docs, middleware wiring
internal/config/config.go       Environment loading and validation
internal/auth/                  Password hashing and access token handling
internal/httpapi/handlers/      HTTP request handlers
internal/httpapi/middleware/    Authentication middleware
internal/httpapi/response/      JSON response helpers
internal/users/                 User domain service and repository
internal/postgres/              PostgreSQL pool setup
internal/postgres/sqlc/         Generated sqlc database code
queries/                        SQL queries consumed by sqlc
migrations/                     PostgreSQL schema migrations
api/openapi.yaml                OpenAPI contract
docs/                           Product and implementation planning docs
```

## Troubleshooting

If `make run` says `.env not found`, create it:

```sh
cp .env.example .env
```

If startup fails with a database connection error, verify PostgreSQL is running and the `DATABASE_URL` value is correct:

```sh
psql 'postgres://mlakp:mlakp@localhost:5432/mlakp?sslmode=disable'
```

If `/readyz` returns `503`, the process is running but PostgreSQL ping failed. Check the database service, credentials, network, and migrations.

If `make migrate-up` fails with `migrate: command not found`, install the `golang-migrate` CLI and rerun the command.

If `make sqlc` fails with `sqlc: command not found`, install the `sqlc` CLI. This is only required when regenerating database access code.

If tests fail with Go cache permission errors outside the `Makefile`, run:

```sh
GOCACHE=/tmp/mlakp-go-build go test ./...
```
