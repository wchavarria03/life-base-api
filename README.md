# life-base-api

A Go API that ingests Costa Rican bank statement PDFs, extracts transactions, stores them in Supabase, and serves them via REST API to a frontend.

## Quick start

```bash
git clone https://github.com/wchavarria03/ledger-api.git
cd ledger-api
cp .env.example .env   # fill in your credentials
go run main.go serve
```

The API will be available at `http://localhost:8080`.

For full setup instructions — local development with a local database or connecting to Supabase — see [docs/running.md](docs/running.md).

## Commands

| Command | Description |
|---------|-------------|
| `serve` | Start the HTTP API server |
| `extract` | Parse PDFs from `data/input/` and import transactions into the DB |

## API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/accounts` | List all accounts |

## Project structure

```
app/
├── cmd/                        # CLI commands (serve, extract)
├── internal/
│   ├── core/                   # Dependency wiring
│   ├── databases/              # Supabase HTTP client + generic helpers
│   ├── repositories/supabase/  # PostgREST repository implementations
│   ├── services/               # Business logic
│   ├── handlers/               # HTTP handlers
│   ├── http/                   # Gin router + server
│   ├── models/                 # Domain types
│   ├── parser/                 # PDF parsing (BAC Credomatic)
│   └── db/migrations/          # Versioned SQL migrations
data/
├── input/                      # Drop PDFs here
└── output/                     # Dry-run JSON output
docs/                           # Detailed guides
```
