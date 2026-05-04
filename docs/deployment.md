# Deployment Guide

This guide describes the production deployment flow for `mlakp-backend` using:

- Supabase for PostgreSQL.
- GitHub Actions for the manual release button.
- Render free web service for hosting the Go API.

The intended release flow is:

1. Merge or push code to `main`.
2. Open GitHub Actions.
3. Click the manual release workflow button.
4. GitHub Actions runs tests, builds the Go API, runs Supabase migrations, and triggers Render deployment.
5. Render builds and starts the latest `main` commit.

Render free web services do not provide the pre-deploy command feature, so database migrations are run from GitHub Actions before triggering Render.

## 1. Supabase Setup

### 1.1 Create the Supabase Project

1. Open Supabase.
2. Create a new project.
3. Save the database password in a secure password manager.
4. Wait until the project is fully provisioned.

### 1.2 Copy the Database Connection String

1. Open the Supabase project dashboard.
2. Click `Connect`.
3. Choose the `Session pooler` connection string.
4. Copy the URI connection string.

Use the Session pooler connection for this backend because:

- This API is a persistent backend service.
- Supabase Session pooler supports IPv4 and IPv6.
- Supabase Transaction pooler is designed for temporary/serverless clients and does not support prepared statements.

The connection string should look similar to this:

```text
postgres://postgres.<project-ref>:<password>@aws-0-<region>.pooler.supabase.com:5432/postgres
```

Append `sslmode=require` if it is not already present:

```text
postgres://postgres.<project-ref>:<password>@aws-0-<region>.pooler.supabase.com:5432/postgres?sslmode=require
```

If the password contains special characters, URL-encode it before using it in the connection string.

### 1.3 Do Not Commit the Supabase URL

The Supabase database URL is a production secret.

Do not put the real value in:

- `.env.example`
- README files
- GitHub workflow YAML
- Committed documentation
- Source code

The real value should only be stored in:

- GitHub Actions secrets.
- Render environment variables.
- Your local `.env` only when you intentionally need local production access.

## 2. Render Setup

### 2.1 Create the Render Web Service

1. Open Render.
2. Click `New`.
3. Choose `Web Service`.
4. Connect the public GitHub repository.
5. Select the `main` branch.
6. Set the runtime to `Go`.

### 2.2 Configure Build and Start Commands

Use this build command:

```sh
go build -tags netgo -ldflags '-s -w' -o app ./cmd/api
```

Use this start command:

```sh
./app
```

Render will build the app again during deploy. The GitHub Actions build step is still useful because it proves the commit can compile before migrations and deployment run.

### 2.3 Disable Render Auto-Deploy

Turn Render auto-deploy off.

Reason:

- Deployment should happen only when you click the GitHub Actions release button.
- If Render auto-deploy stays on, Render can deploy immediately after a push to `main`, before GitHub Actions runs migrations.

In Render:

1. Open the web service.
2. Go to `Settings`.
3. Find `Auto-Deploy`.
4. Set it to `Off`.

### 2.4 Configure Render Environment Variables

Add these environment variables in Render:

```env
APP_ENV=production
APP_PORT=10000
DATABASE_URL=postgres://postgres.<project-ref>:<password>@aws-0-<region>.pooler.supabase.com:5432/postgres?sslmode=require
TOKEN_ISSUER=mlakp-backend
TOKEN_AUDIENCE=mlakp-api
TOKEN_SECRET=<long-random-production-secret-at-least-32-bytes>
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=720h
READ_TIMEOUT=5s
WRITE_TIMEOUT=10s
IDLE_TIMEOUT=60s
SHUTDOWN_TIMEOUT=10s
```

Important repo-specific details:

- The app reads `APP_PORT`, not `PORT`.
- Render's default web port is `10000`, so set `APP_PORT=10000`.
- `APP_ENV=production` disables the `/docs` route.
- `TOKEN_SECRET` must be at least 32 bytes in production.
- `DATABASE_URL` must use `postgres://` or `postgresql://` and include a database name.

### 2.5 Configure Health Check

Set the Render health check path to:

```text
/readyz
```

`/readyz` checks database connectivity, so Render will only consider the service ready when it can reach Supabase.

If you need a simpler process-only health check while debugging, temporarily use:

```text
/healthz
```

For production, prefer `/readyz`.

### 2.6 Copy the Render Deploy Hook URL

1. Open the Render web service.
2. Go to `Settings`.
3. Find `Deploy Hook`.
4. Copy the deploy hook URL.

Treat this URL as a secret. Anyone with the URL can trigger a Render deploy.

## 3. GitHub Secrets

Open the GitHub repository:

1. Go to `Settings`.
2. Go to `Secrets and variables`.
3. Click `Actions`.
4. Add these repository secrets.

### 3.1 SUPABASE_DATABASE_URL

Name:

```text
SUPABASE_DATABASE_URL
```

Value:

```text
postgres://postgres.<project-ref>:<password>@aws-0-<region>.pooler.supabase.com:5432/postgres?sslmode=require
```

This is used by GitHub Actions to run migrations.

### 3.2 RENDER_DEPLOY_HOOK_URL

Name:

```text
RENDER_DEPLOY_HOOK_URL
```

Value:

```text
https://api.render.com/deploy/...
```

This is used by GitHub Actions to trigger the Render deployment.

## 4. GitHub Actions Workflow

Create this file:

```text
.github/workflows/release.yml
```

Use this workflow:

```yaml
name: Release

on:
  workflow_dispatch:

concurrency:
  group: production-release
  cancel-in-progress: false

jobs:
  release:
    name: Build, migrate, and deploy
    runs-on: ubuntu-latest

    steps:
      - name: Require main branch
        if: github.ref != 'refs/heads/main'
        run: |
          echo "Release workflow must be run from the main branch."
          echo "Selected ref: $GITHUB_REF"
          exit 1

      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Download dependencies
        run: go mod download

      - name: Test
        run: go test ./...

      - name: Build
        run: go build -tags netgo -ldflags '-s -w' -o app ./cmd/api

      - name: Validate production secrets
        run: |
          test -n "$DATABASE_URL"
          test -n "$RENDER_DEPLOY_HOOK_URL"
        env:
          DATABASE_URL: ${{ secrets.SUPABASE_DATABASE_URL }}
          RENDER_DEPLOY_HOOK_URL: ${{ secrets.RENDER_DEPLOY_HOOK_URL }}

      - name: Install golang-migrate
        run: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

      - name: Run Supabase migrations
        run: |
          "$(go env GOPATH)/bin/migrate" -path migrations -database "$DATABASE_URL" up
        env:
          DATABASE_URL: ${{ secrets.SUPABASE_DATABASE_URL }}

      - name: Trigger Render deploy
        run: curl -fsS -X POST "$RENDER_DEPLOY_HOOK_URL"
        env:
          RENDER_DEPLOY_HOOK_URL: ${{ secrets.RENDER_DEPLOY_HOOK_URL }}
```

### 4.1 Why This Workflow Is Manual

The workflow uses:

```yaml
on:
  workflow_dispatch:
```

That creates a `Run workflow` button in GitHub Actions.

The workflow does not run automatically on every push. You decide when to release.

The workflow also refuses to run unless the selected branch is `main`. This prevents production migrations from running from a feature branch while Render deploys the configured production branch.

### 4.2 Why There Is One Release Button

The workflow intentionally keeps build, migration, and deployment in one button.

Reason:

- Tests must pass before migration.
- Build must pass before migration.
- Migration must pass before Render deploy.
- Render should not deploy code before the database is ready.

If the migration fails, the Render deploy step does not run.

### 4.3 Why Render Builds Again

This workflow builds the app in GitHub Actions, then Render builds it again.

That is expected for a normal Render Go service.

The GitHub build is a release gate. It proves the commit compiles before production migration and deployment.

Render's build is the actual build used by the hosted service.

## 5. First Production Deployment

### 5.1 Verify the Render Service Is Not Auto-Deploying

Before clicking the GitHub release button:

1. Open Render.
2. Open the service.
3. Confirm `Auto-Deploy` is `Off`.

### 5.2 Commit the Workflow

After creating `.github/workflows/release.yml`, commit and push it to `main`:

```sh
git add .github/workflows/release.yml docs/deployment.md
git commit -m "Add production deployment guide and release workflow"
git push origin main
```

The manual workflow must exist on the default branch before GitHub shows the `Run workflow` button.

### 5.3 Run the Release Workflow

In GitHub:

1. Open the repository.
2. Click `Actions`.
3. Select `Release`.
4. Click `Run workflow`.
5. Select branch `main`.
6. Click `Run workflow`.

Watch the workflow logs.

Expected order:

1. Checkout succeeds.
2. Go setup succeeds.
3. Dependency download succeeds.
4. Tests pass.
5. Build passes.
6. `golang-migrate` installs.
7. Supabase migrations run.
8. Render deploy hook is triggered.

### 5.4 Watch Render Deployment

After the GitHub workflow triggers Render:

1. Open Render.
2. Open the web service.
3. Check the `Events` or `Logs` tab.
4. Confirm Render builds the latest `main` commit.
5. Confirm the service starts successfully.

### 5.5 Verify the Deployed API

Replace `<render-url>` with the real Render service URL:

```sh
curl https://<render-url>/healthz
curl https://<render-url>/readyz
```

`/healthz` verifies the API process is alive.

`/readyz` verifies the API can reach Supabase.

Then test a real API flow, such as register and login, with your API client.

## 6. Normal Release Process

For future releases:

1. Merge code into `main`.
2. Open GitHub Actions.
3. Run the `Release` workflow manually.
4. Confirm GitHub Actions finishes successfully.
5. Confirm Render deploy finishes successfully.
6. Verify `/readyz`.
7. Smoke test one authenticated API flow.

Do not run production migrations manually from local unless you are intentionally doing an emergency operation.

## 7. Failure Handling

### 7.1 Tests Fail

If `go test ./...` fails:

- The build step does not matter yet.
- Migrations do not run.
- Render deployment does not trigger.
- Fix the code, merge again, then rerun the workflow.

### 7.2 Build Fails

If the build step fails:

- Migrations do not run.
- Render deployment does not trigger.
- Fix the compile issue and rerun the workflow.

### 7.3 Migration Fails

If migration fails:

- Render deployment does not trigger.
- Check the failed migration log in GitHub Actions.
- Check the `schema_migrations` table in Supabase.
- Fix the migration safely before rerunning.

Do not edit an already-applied migration file for production unless you are certain it has not been applied anywhere important. Prefer adding a new migration.

### 7.4 Render Deploy Hook Fails

If the deploy hook step fails:

- Tests, build, and migrations may already have completed.
- Check whether the `RENDER_DEPLOY_HOOK_URL` GitHub secret is correct.
- Regenerate the deploy hook in Render if it was leaked or invalid.
- Rerun the workflow after fixing the secret.

### 7.5 Render Build Fails After Migrations Succeeded

This should be uncommon because GitHub Actions already builds the app.

If it happens:

- Check Render build logs.
- Check whether Render is using the same branch and commit.
- Check whether Render has all required environment variables.
- Fix the issue and run the release workflow again.

## 8. Security Checklist

- Keep Supabase `DATABASE_URL` out of committed files.
- Keep Render deploy hook out of committed files.
- Use GitHub repository secrets for production values.
- Use a long random `TOKEN_SECRET` in Render.
- Keep Render auto-deploy off when using this manual release workflow.
- Use Supabase Session pooler for the app connection.
- Avoid Supabase Transaction pooler unless the database client is explicitly configured to avoid prepared statements.
- Prefer `/readyz` as the Render health check path.

## 9. Reference Links

- Render deploy hooks: https://render.com/docs/deploy-hooks
- Render Go web service deploy: https://render.com/docs/deploy-go-nethttp
- Render deployment steps: https://render.com/docs/deploys
- GitHub manual workflows: https://docs.github.com/en/actions/how-tos/manage-workflow-runs/manually-run-a-workflow
- Supabase connection strings: https://supabase.com/docs/reference/postgres/connection-strings
