# Rename plan: ledger-api → life-base-api

Execute this with Claude Code running inside this repo (`ledger-api`). Companion doc lives in
the frontend repo at `ledger-api-fe/docs/RENAME_PLAN.md` for the matching frontend rename —
these are independent, either can run first.

## Context

This repo is becoming the backend for a broader personal hub ("Life-Base") covering Finance
plus future modules (Bikes, Household, Notes, TODO). See
`../ledger-api-fe/docs/HUB_PRD.md` and `HUB_PLAN.md` (in the frontend repo) for the full
picture — this doc only covers the Phase 0 rename mechanics for this repo.

This is a pure rename: no behavior change, no new features. At the end, `go build ./...`,
`go vet ./...`, and `go test ./...` must all still pass exactly as before.

## Steps

1. **`go.mod`** — change the module line:
   ```
   module ledger-api
   ```
   to:
   ```
   module life-base-api
   ```

2. **Every Go file's import path** — all internal imports use the `ledger-api/app/...` prefix.
   Update them all in one pass:
   ```bash
   grep -rl "ledger-api/app" --include="*.go" . | xargs sed -i '' 's#ledger-api/app#life-base-api/app#g'
   ```
   (On Linux, drop the `''` after `-i`: `sed -i 's#...#...#g'`.)
   Verify nothing was missed:
   ```bash
   grep -rn "ledger-api" --include="*.go" .
   ```
   should return nothing.

3. **`Makefile`** — the binary name:
   ```
   BINARY   := ledger-api
   ```
   to:
   ```
   BINARY   := life-base-api
   ```

4. **`Dockerfile`** — four references to the binary name:
   - `-o ledger-api .` → `-o life-base-api .`
   - `COPY --from=builder /app/ledger-api .` → `COPY --from=builder /app/life-base-api .`
   - `chmod 755 /app/ledger-api` → `chmod 755 /app/life-base-api`
   - `ENTRYPOINT ["ledger-api"]` → `ENTRYPOINT ["life-base-api"]`

5. **`README.md`** — first line:
   ```
   # ledger-api
   ```
   to:
   ```
   # life-base-api
   ```
   Leave the rest of the README as-is for now (it still accurately describes the Finance/import
   functionality that exists today — don't rewrite it to describe unbuilt modules).

6. **`docs/running.md`** — first line:
   ```
   # Running ledger-api
   ```
   to:
   ```
   # Running life-base-api
   ```

## Explicitly do NOT touch in this pass

- **`.env.example`'s `LEDGER_USER_ID`** and its one usage in `app/cmd/root.go` (`os.Getenv("LEDGER_USER_ID")`)
  — this is a CLI-only config var for local/manual imports, not user-facing, and renaming it
  would require updating real local `.env` files outside this repo for no real benefit. Leave it.
- **The live Render service URL** (`ledger-api-o3fa.onrender.com`, referenced from the frontend
  repo) — renaming this repo on GitHub does not change the deployed Render service or its URL.
  Do not attempt to rename the Render service as part of this pass; that's a separate, riskier
  step (it would change the live API URL and require updating the frontend's `VITE_API_URL`
  everywhere it's set) and is not scoped here.
- **`docs/FUTURE_ENHANCEMENTS.md`, `app/internal/db/README.md`** — checked, neither mentions
  the old name, nothing to change.
- **`go.sum`** — only lists external dependencies, never references this module's own name.

## Verify

```bash
go build ./...
go vet ./...
go test ./...
```
All three must pass with no errors (matching whatever their state was before this change —
this rename should not fix or break anything else).

## Commit

```bash
git add -A
git commit -m "chore: rename module to life-base-api

Pure rename ahead of the GitHub repo rename (ledger-api -> life-base-api),
part of consolidating this backend into a broader personal hub. No
behavior change -- module path, binary name, and doc titles only.
See docs/RENAME_PLAN.md and ../ledger-api-fe/docs/HUB_PRD.md."
```

## After this: manual steps (not for Claude Code)

1. Rename the repo on GitHub: Settings → General → repository name → `life-base-api`
   (or via `gh repo rename life-base-api` if the `gh` CLI is authenticated locally). GitHub
   redirects the old URL automatically.
2. Update the local git remote to match (optional — the redirect means it still works, but
   cleaner):
   ```bash
   git remote set-url origin git@github.com:wchavarria03/life-base-api.git
   ```
3. Render: the deployed service can keep its current name/URL indefinitely — nothing there
   needs to change for this rename to be complete. Only revisit that if/when you decide the
   `ledger-api-o3fa.onrender.com` URL itself should change, as its own separate decision.
