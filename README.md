# MLAKP Backend

Go backend for the MLAKP shared expense API. The current implementation exposes health checks, OpenAPI documentation in local/test mode, rate-limited user registration/login/refresh, strict JSON request decoding, session-backed logout, authenticated current-user/profile update endpoints, username search, authenticated group creation, listing, details, member management by username, expense creation/detail/listing, debtor-only debt acceptance/rejection, owner review/resend for rejected debts, current-user debt listing, payment listing/marking/review, and dashboard snapshots.

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
CORS_ALLOWED_ORIGINS=http://localhost:8081,http://localhost:19006
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=720h
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
- `CORS_ALLOWED_ORIGINS` is a comma-separated list of exact browser origins allowed to call the API. Include scheme, host, and optional port only, for example `http://localhost:8081`; do not include paths.
- Token TTL and timeout values must be valid Go durations such as `15m`, `5s`, or `1h`.

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

The second migration creates:

- `auth_sessions` table
- refresh token hash uniqueness constraint
- active-session lookup index

The third migration creates:

- `groups` table
- `group_members` table
- owner/member role constraint
- group membership uniqueness constraint
- group membership lookup index

The fourth migration creates:

- `expenses` table
- `expense_participants` table
- `debts` table
- expense, participant, and debt integrity constraints
- indexes for group expense lists and debt lookups

The fifth migration creates:

- `payments` table
- pending/confirmed/rejected payment status constraint
- payment amount and payer/receiver integrity constraints
- indexes for debt, payer, and receiver payment lookups

The sixth migration adds:

- `users.username` column
- lowercase username and format constraints
- username uniqueness constraint
- username prefix-search index

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
{"success":true,"data":{"status":"ok"}}
```

Check readiness:

```sh
curl -i http://localhost:8080/readyz
```

Expected response when PostgreSQL is reachable:

```json
{"success":true,"data":{"status":"ready"}}
```

Open Swagger UI when `APP_ENV=local` or `APP_ENV=test`:

```text
http://localhost:8080/docs
```

Open the raw OpenAPI spec when `APP_ENV=local` or `APP_ENV=test`:

```text
http://localhost:8080/docs/openapi.yaml
```

When `APP_ENV=production`, the Swagger UI and raw OpenAPI route are not registered.

## Auth Smoke Test

Register a user:

```sh
curl -s -X POST http://localhost:8080/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Thomas",
    "username": "thomas",
    "email": "thomas@example.com",
    "password": "password123"
  }'
```

The response includes access and refresh tokens:

```json
{
  "success": true,
  "data": {
    "access_token": "...",
    "refresh_token": "...",
    "token_type": "Bearer",
    "expires_at": "2026-05-02T12:15:00Z",
    "user": {
      "id": "...",
      "name": "Thomas",
      "email": "thomas@example.com"
    }
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
REFRESH_TOKEN='paste-refresh-token-here'
```

Call the authenticated current-user endpoint:

```sh
curl -s http://localhost:8080/v1/users/me \
  -H "Authorization: Bearer $TOKEN"
```

Create a group:

```sh
curl -s -X POST http://localhost:8080/v1/groups \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Home"}'
```

List your groups:

```sh
curl -s http://localhost:8080/v1/groups \
  -H "Authorization: Bearer $TOKEN"
```

Read a group with members:

```sh
curl -s http://localhost:8080/v1/groups/$GROUP_ID \
  -H "Authorization: Bearer $TOKEN"
```

The group detail response includes each member's `user_id`, role, join time, and `user` profile summary including username. Mobile clients use that member list to select expense participants.

Search users by username prefix:

```sh
curl -s "http://localhost:8080/v1/users/search?username=ali" \
  -H "Authorization: Bearer $TOKEN"
```

Add a member as the group owner:

```sh
curl -s -X POST http://localhost:8080/v1/groups/$GROUP_ID/members \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice"}'
```

Create an expense:

```sh
curl -s -X POST http://localhost:8080/v1/expenses \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "group_id": "'"$GROUP_ID"'",
    "title": "Dinner",
    "total_amount": "100.00",
    "currency": "THB",
    "paid_by": "'"$PAYER_ID"'",
    "split_type": "equal",
    "participants": [
      {"user_id": "'"$PAYER_ID"'"},
      {"user_id": "'"$MEMBER_ID"'"}
    ]
  }'
```

Read expense, debt, and dashboard data:

```sh
curl -s http://localhost:8080/v1/expenses/$EXPENSE_ID \
  -H "Authorization: Bearer $TOKEN"

curl -s http://localhost:8080/v1/groups/$GROUP_ID/expenses \
  -H "Authorization: Bearer $TOKEN"

curl -s http://localhost:8080/v1/debts \
  -H "Authorization: Bearer $TOKEN"

curl -s 'http://localhost:8080/v1/payments?type=received&status=pending_confirmation' \
  -H "Authorization: Bearer $TOKEN"

curl -s http://localhost:8080/v1/dashboard \
  -H "Authorization: Bearer $TOKEN"
```

The payments list returns records where the current user is either payer or receiver. Use `type=received` for the creditor review inbox, `type=sent` for submitted payments, and optional `status=pending_confirmation|confirmed|rejected` for state-specific views.

The dashboard response includes `you_owe`, `you_get`, and an `unsettled_balances` preview with up to five pending or active balances that still have remaining amount. Each preview item includes the source expense title, counterparty user, remaining amount, status, and whether the current user sees it as `owed` or `receivable`.

Refresh the access token:

```sh
curl -s -X POST http://localhost:8080/v1/auth/refresh \
  -H 'Content-Type: application/json' \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}"
```

Logout revokes the current server-side session. The access token and refresh token for that session are rejected after logout:

```sh
curl -i -X POST http://localhost:8080/v1/auth/logout \
  -H "Authorization: Bearer $TOKEN"
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

Regenerate OpenAPI, format, vet, and run tests:

```sh
make check
```

Run PostgreSQL integration/concurrency tests:

```sh
MLAKP_TEST_DATABASE_URL='postgres://mlakp:mlakp@localhost:5432/mlakp_test?sslmode=disable' make test-integration
```

The integration tests apply migrations into a temporary schema and drop it after the run. If `MLAKP_TEST_DATABASE_URL` is not set, `make test-integration` falls back to `DATABASE_URL` from the environment or `.env`.

Regenerate SQL code after changing files in `queries/` or `migrations/`:

```sh
make sqlc
```

Generated SQL code is written to:

```text
internal/postgres/sqlc/
```

Regenerate the served OpenAPI document after changing files in `api/openapi/`:

```sh
make openapi
```

Edit the split OpenAPI source files by feature:

```text
api/openapi/root.yaml
api/openapi/paths/
api/openapi/components/
```

The generated document served by the app is:

```text
api/openapi.yaml
```

## Project Layout

```text
cmd/api/main.go                 Application entrypoint and HTTP server startup
internal/app/routes.go          Route registration, health/readiness, docs, middleware wiring
internal/config/config.go       Environment loading and validation
internal/auth/                  Password hashing and access token handling
internal/sessions/              Server-side session and refresh token handling
internal/httpapi/handlers/      HTTP request handlers
internal/httpapi/middleware/    Authentication and rate limiting middleware
internal/httpapi/response/      JSON response helpers
internal/users/                 User domain service and repository
internal/groups/                Group domain service and repository
internal/money/                 Minor-unit money parsing, formatting, and splitting
internal/expenses/              Expense creation and debt generation
internal/debts/                 Debt state transitions and rejected-debt review
internal/payments/              Payment listing, marking, and creditor review
internal/dashboard/             Current-user financial summaries
internal/postgres/              PostgreSQL pool setup
internal/postgres/sqlc/         Generated sqlc database code
queries/                        SQL queries consumed by sqlc
migrations/                     PostgreSQL schema migrations
api/openapi/                    Split OpenAPI source files
api/openapi.yaml                Generated OpenAPI contract served by the app
scripts/openapi/                OpenAPI bundler
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
