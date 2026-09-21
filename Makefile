BINARY   := life-base-api
INPUT    := data/input
OUTPUT   := data/output

.DEFAULT_GOAL := help

.PHONY: help build clean tidy lint vuln extract extract-dry dump setup db-up db-down db-reset

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

# ── Build ─────────────────────────────────────────────────────────────────────

build: ## Build the binary
	go build -o $(BINARY) .

clean: ## Remove binary and output files
	rm -f $(BINARY)
	rm -f $(OUTPUT)/*.transactions $(OUTPUT)/*.txt

tidy: ## Tidy go modules
	go mod tidy

lint: ## Run golangci-lint
	golangci-lint run ./...

vuln: ## Scan for vulnerabilities
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# ── Run ───────────────────────────────────────────────────────────────────────

extract: build ## Extract and import transactions (requires SUPABASE_URL + SUPABASE_KEY)
	./$(BINARY) -v -i $(INPUT) -o $(OUTPUT) extract

extract-dry: build ## Extract transactions and write to files (no DB writes)
	./$(BINARY) -v --dry-run -i $(INPUT) -o $(OUTPUT) extract

dump: build ## Dump raw extracted text from PDFs (for debugging parsers)
	./$(BINARY) -v -i $(INPUT) -o $(OUTPUT) dump

# ── Local dev ─────────────────────────────────────────────────────────────────

setup: ## Copy .env.example to .env if .env does not exist
	@if [ ! -f .env ]; then cp .env.example .env && echo "Created .env from .env.example"; \
	else echo ".env already exists, skipping"; fi

db-up: ## Start local PostgreSQL + PostgREST (applies migrations automatically)
	docker compose up -d
	@echo "PostgREST available at http://localhost:3000"

db-down: ## Stop local database services
	docker compose down

db-reset: ## Wipe local database volume and restart (re-applies all migrations)
	docker compose down -v
	docker compose up -d
