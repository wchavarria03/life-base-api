# Notes vault sync (git <-> app)

Implements HUB_PLAN_V2 Phase D (D1-D5, see `ledger-api-fe/docs/HUB_PLAN_V2.md`).
**Status: written without a Go/Nix toolchain. Nothing here has been compiled,
vetted or run** except `notes-autosync.sh`, which was exercised against local
git repos (debounce, pause file, branch guard, index.lock, conflict abort).

## Principle

Git is the source of truth. `notes.notes` is a derived, rebuildable index for
vault notes (`source = 'vault'`). App edits flow back to git through the same
pipeline, so there is never a second source of truth.

## Architecture

```
 Mac1 / Mac2 (nvim, Emacs)                       iPhone / Windows (PWA)
   |  notes-autosync.sh (launchd)                       |
   v  git push over SSH, Tailscale only                 v  HTTPS + Supabase JWT
 +--------------------------- homelab ---------------+  +--------- Render ---------+
 | bare repo  /srv/data/notes/notes.git (ZFS)       |  | Life-Base API (Go/Gin)   |
 |   post-receive: touch sync-trigger               |  |  /v1/notes/*  (user)     |
 |   -> notes-github-mirror (private GitHub)        |  |  /v1/internal/notes/*    |
 | notes-sync-worker (systemd, outbound HTTPS only) |->|   (service token)        |
 |   work tree clone, upsert / pending / ack        |  |  Supabase PostgREST      |
 +--------------------------------------------------+  +--------------------------+
```

Render cannot reach Tailscale, so the worker only makes outbound calls.

### Worker cycle (`tools/notes-sync/worker`)

1. `git fetch` + hard-reset the disposable work tree to `origin/main`.
2. Diff since the last synced commit (state file); keep only valid `.org`/`.md`
   paths (no PDFs, images, dotfiles, `org-roam.db`); POST them to
   `/v1/internal/notes/upsert` (deletions as `deleted: true`).
3. GET `/v1/internal/notes/pending` (app edits with `pending_write = true`,
   not in conflict). Write each file (path re-validated, symlink-safe), one
   commit `app: update N note(s)`, plain `git push` (never forced), then
   `/ack` each note with the commit and a SHA-256 of what was written.
4. POST `/sync-done` with the HEAD commit when anything changed or the user
   pressed **Sync** (the pending response carries `sync_requested`).

The post-receive hook only touches `sync-trigger`; the daemon notices within
2 s. Otherwise it polls the API every `POLL_INTERVAL` (default 60 s).

### Conflicts (never lose an app edit, never force-push)

* Upsert for a note with `pending_write = true`:
  * identical content (echo of our own push) -> clear the flag;
  * different content -> **do not overwrite**: archive the incoming vault text
    in `notes.note_versions` with title prefix `[vault] `, set `conflict = true`.
    A vault deletion is archived as a `[vault]` version with null content.
* Conflicted notes are excluded from `/pending` until resolved.
* `POST /v1/notes/:id/resolve-conflict {"keep":"mine"|"vault"}`:
  `mine` clears the flag and keeps `pending_write` (the worker will commit the
  app version on top of the vault one - a normal commit, history preserved);
  `vault` adopts the archived vault text (the app version is archived first).
* If the worker's push is rejected (someone pushed meanwhile) it does not
  retry with force; next cycle resets to the remote and rewrites the notes that
  are still pending. If the remote changed the same file, the upsert flags a
  conflict.
* The ack carries a content hash: if the user edited again after the worker
  fetched pending, `pending_write` stays true and the newer text is written
  next cycle. The worker advances its state file past its own push so the echo
  through the bare repo is not mistaken for a vault edit.

## API summary

User (JWT, RLS applies):

| Route | Purpose |
| --- | --- |
| `GET /v1/notes/:id` | note JSON; adds `source, sensitive, content_format, vault_path, vault_commit, pending_write, conflict` and, for org, `rendered_html` |
| `PATCH /v1/notes/:id` | edit; protected fields (source, vault_path, sensitive, ...) are stripped; editing a vault note sets `pending_write`, re-derives the title from content |
| `PATCH /v1/notes/:id/sensitive` `{sensitive}` | toggle; 409 `sensitive_declared_in_file` when the file itself declares it |
| `POST /v1/notes/sync-request`, `GET /v1/notes/sync-status` | Sync button; `last_commit`, `last_synced_at` |
| `GET /v1/notes/:id/backlinks` | notes linking here (by id or unresolved org id) |
| `GET /v1/notes/search?q=&limit=` | full-text (tsvector, `simple` config, `<<match>>` snippet markers) |
| `GET /v1/notes/lookup?org_id=` | resolve an `id:` link to a note |
| `POST /v1/notes/:id/resolve-conflict` | see above |
| `POST /v1/share-links` | adds `confirm_sensitive`; see `SHARING_HARDENING.md` |

Worker (static bearer token, `/v1/internal/notes/...`): `POST upsert`
(<= 100 items, <= 1 MiB each, <= 8 MiB body), `GET pending`, `POST ack`,
`POST sync-done`. Disabled (404) when `VAULT_SYNC_TOKEN` is unset.

## Sensitive flag

Stored column `notes.notes.sensitive`. A note is sensitive when:
the file declares it (org `#+filetags: :sensitive:` or `:SENSITIVE: t` in the
file-level property drawer; md front-matter `sensitive: true`) **or** its path
starts with `5-Health/` / `legacy-para/3-Resources/3-Finances/` **unless** the
user overrode it in the app (`sensitive_override`). A file-declared flag can
never be switched off from the app. In-app overrides are stored in the DB only;
they are not written back into the file. Consequences implemented: share
creation needs confirm + <= 7 days; the frontend never offers flagged notes to
the offline cache (the PWA cache itself is not built yet - D4 open).
Non-text files are never parsed or indexed (worker and API both reject them).

## Rendering

`app/internal/vault`: dependency-free org parser + renderer. Every text node
and attribute goes through `html.EscapeString`; the tag set is fixed in code;
no raw HTML pass-through (`#+begin_export`, html blocks are dropped); hrefs
limited to http/https/mailto; property drawers and keywords hidden. Embeds are
inert `<button class="lb-embed" data-kind="image|video" data-src="...">`
placeholders; the frontend creates `<img>` / players only on click.
`go-org` + `bluemonday` can replace it later (needs `go get`).

## Setup order

1. Apply migration `app/internal/db/migrations/notes/004_vault_sync.sql`
   (after notes/001-003). **Deploy order: run the migration first**, then the
   API (the note list filters on `deleted_at`).
2. Render env: `VAULT_SYNC_TOKEN` (long random, e.g. `openssl rand -hex 32`),
   `VAULT_SYNC_USER_ID` (your Supabase `auth.users.id`).
3. NixOS (DRAFT): `zfs create data/notes`, sops secrets, fill the placeholders
   in `deploy/nixos/notes-git.nix`, `nixos-rebuild build` then `switch`.
4. Create the bare repo remote on each Mac (`git remote set-url origin`), push
   the existing vault once (`git push -u origin main`), set up the GitHub
   mirror deploy key.
5. Start the worker; watch `journalctl -u notes-sync-worker -f`. The first
   cycle is a full resync (every `.org`/`.md` file is upserted).
6. Install `notes-autosync.sh` + launchd on the Macs (`tools/notes-sync/README.md`).

## Rollback

* Stop the worker (`systemctl stop notes-sync-worker`): git keeps working, the
  app just stops syncing. App edits stay `pending_write`.
* Kill switch on any Mac: `touch <repo>/.notes-autosync-ignore`.
* Remove the app index and rebuild: `delete from notes.notes where source =
  'vault'` (cascades to links/versions) and restart the worker with the state
  file deleted (`rm <work>/.git/notes-sync-last-commit`) for a full resync.
  App-only notes (`source = 'app'`) are untouched.
* Migration down (destructive): drop `notes.note_links`, `notes.vault_sync_state`,
  function `notes.search_notes`, then the added columns of `notes.notes`.
* Unset `VAULT_SYNC_TOKEN` on Render: internal routes answer 404.

## Security notes

The service token can: upsert/soft-delete **vault** notes for the single
configured user, list that user's pending notes (full content, including
sensitive ones), ack, and mark sync done. It cannot read other notes, other
users' data, or any non-notes endpoint, and carries no Supabase session. It
*is* powerful for that one user's notes: store it only in sops, rotate by
changing the sops value and the Render env var together. The API enforces
path validation (no `..`, absolute, backslash, NUL, hidden segments, extension
allow-list, 240 bytes), size limits, a global 240 req/min throttle (counted
before auth, as a brute-force speed bump) and constant-time token comparison.
Because the worker runs with the service-role DB key on the server side (the
repository calls use the service key when no user JWT is in context), every
repository call in `VaultService` scopes by `user_id` explicitly - keep it that
way when editing. The `git` account is `git-shell` only, SSH is Tailscale-only,
and the bare repo refuses non-fast-forward pushes and branch deletion.

Open risks: renderer is hand-written (tests included but unrun); the worker
holds sensitive text in its work tree (root-disk, `git` user 0750); GitHub
mirror includes sensitive folders by decision (enable 2FA + secret scanning);
see `SHARING_HARDENING.md` for share-side follow-ups.

## AI worktree convention

AI-proposed vault changes never touch the Mac's main worktree (auto-sync would
commit them to `main` within 5 minutes):

```sh
cd ~/personal/notes
git worktree add ~/personal/notes-ai -b ai/chore/scratch-$(date +%F) main   # first time
# per task: cd ~/personal/notes-ai && git switch -c ai/research/<slug>-<date> main
```

The auto-sync script operates on one directory and one branch (`main`), so
branches checked out in `~/personal/notes-ai` are invisible to it. Review and
merge from the main worktree.

## Proposed vault-side changes (apply on an `ai/chore/` branch, for the owner)

These were NOT applied; the vault is read-only for this work.

`.gitignore` additions: see `tools/notes-sync/README.md` ("Proposed .gitignore
additions").

`CLAUDE.md` addition (under "Git workflow (memory approval)"):

```markdown
- AI work happens in a **separate git worktree**, never in the main checkout:
  `git worktree add ~/personal/notes-ai -b ai/chore/<slug>-<YYYY-MM-DD> main`.
  The main checkout is auto-committed to `main` by `notes-autosync` every ~5
  minutes after you stop editing, so any unreviewed AI change left there would
  land on `main`. Before writing anything, confirm the working directory is the
  `notes-ai` worktree; if you are in the main checkout, stop and ask.
- Do not create or edit `.notes-autosync-ignore` or the `sensitive` file
  declarations without being asked: `#+filetags: :sensitive:` / `:SENSITIVE: t`
  mark a note as sensitive for sharing and offline caching.
```

## Open items (not built)

Mapping to HUB_PLAN_V2 Phase D, honest status:

* **D1**: schema, links table, search, parser/renderer, sensitive flag + toggle:
  written, unrun. "Tests that flagged notes are excluded from ... offline
  cache and any future subtree/collection share": only the share-creation
  guard exists (no tests against a live DB; Go unit tests cover the pure rules).
* **D2**: internal endpoints + worker + NixOS draft written, unrun. The rate
  limit is a single global in-memory counter (240/min), not per-IP.
* **D3**: autosync script (tested against local repos), plist, README, patch
  text written. The vault-side `.gitignore`/`CLAUDE.md` changes are proposals.
  **Attachments are not built**: the worker only syncs `.org`/`.md`; nothing
  copies `attachments/` to a bucket and no endpoint serves images, so the
  frontend's image placeholder shows "image not available yet" unless
  `VITE_ATTACHMENTS_BASE_URL` points at something that serves the files.
  Uploading images from the PWA (worker writes them into `attachments/`) is
  also not built.
* **D4**: search, backlinks, source badge, conflict resolver, Sync button,
  `pending_write` on edit, on-demand embeds (org only; the markdown renderer is
  a custom line renderer with no image support, left unchanged): done.
  **Not done**: graph view, PWA offline cache (so "never prefetch sensitive
  notes" has nothing to enforce yet), in-app image upload.
* **D5**: see `SHARING_HARDENING.md`. Shared-note image serving, the snapshot /
  read-only-role resolver, token hashing and the public-route rate limit are
  open.
