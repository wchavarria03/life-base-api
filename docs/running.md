# Running life-base-api

Two ways to run the project: against a **local database** (no internet required) or against a **Supabase production database**.

---

## Environment variables

All configuration is loaded from a `.env` file in the project root. Copy the example and fill in the values:

```bash
cp .env.example .env
```

| Variable | Description | Default |
|----------|-------------|---------|
| `SUPABASE_URL` | Base URL for the PostgREST API | — |
| `SUPABASE_KEY` | API key (service role for backend) | — |
| `SERVER_ADDR` | Address the HTTP server listens on | `:8080` |

The project uses [direnv](https://direnv.net) to auto-load `.env` when you enter the directory. Run `direnv allow` once after cloning or renaming the folder.

---

## Option 1 — Local database (Podman / Docker)

Runs a local Postgres + PostgREST stack that mirrors the Supabase API. No internet required.

### Prerequisites

- [Podman](https://podman.io) with `podman-compose`, or Docker with `docker-compose`

### Steps

**1. Start the local stack**

```bash
# Podman
podman-compose up -d

# Docker
docker-compose up -d
```

Migrations in `app/internal/db/migrations/` are applied automatically on first start via `docker-entrypoint-initdb.d`.

**2. Configure `.env` for local**

```
SUPABASE_URL=http://localhost:3000
SUPABASE_KEY=local-dev
SERVER_ADDR=:8080
```

**3. Run the API**

```bash
go run main.go serve
```

**4. Stop the stack**

```bash
# Podman
podman-compose down

# Docker
docker-compose down
```

---

## Option 2 — Supabase production

Connects to a real Supabase project. Requires a Supabase account.

### Prerequisites

- A Supabase project with all three migrations applied (see below)

### Run migrations

In the Supabase **SQL Editor**, run each migration in order:

1. `app/internal/db/migrations/001_initial_schema.sql`
2. `app/internal/db/migrations/002_classification_rules.sql`
3. `app/internal/db/migrations/003_user_isolation.sql`

### Get credentials

In **Project Settings → API Keys**:

- **Project URL** → `SUPABASE_URL` (format: `https://<project-id>.supabase.co`)
- **Legacy service_role key** → `SUPABASE_KEY` (use the legacy JWT key, not the new `sb_secret_...` format)

### Configure `.env` for production

```
SUPABASE_URL=https://<project-id>.supabase.co
SUPABASE_KEY=<service-role-key>
SERVER_ADDR=:8080
```

### Run the API

```bash
go run main.go serve
```

### Verify the connection

```bash
curl http://localhost:8080/v1/accounts
# Expected: [] (empty array — no accounts yet)
```

---

## Importing transactions

Drop BAC Credomatic PDF statements into `data/input/` then run:

```bash
go run main.go extract
```

Use `--dry-run` to write JSON to `data/output/` instead of inserting into the database:

```bash
go run main.go --dry-run extract
```
