# Pending: final testing per domain migration

Consolidated checklist of what's left before each newly-migrated/built domain can be
considered actually working end-to-end — not just "builds and vets clean." Do these
together at the end rather than one at a time.

## Social posting (Facebook + Instagram)

- [ ] Set real env vars on deploy (Render): `META_ACCESS_TOKEN`, `META_FACEBOOK_PAGE_ID`,
      `META_INSTAGRAM_USER_ID`.
- [ ] Manual smoke test: `POST /v1/social/posts` with a real image — confirm it actually
      posts to both Facebook and Instagram, not just that the endpoint returns 201.
- [ ] Confirm `facebook_permalink`/`instagram_permalink`/`facebook_photo_url` come back
      populated (best-effort calls — worth checking they don't silently no-op).
- [ ] Exercise the edge cases once real posting works: duplicate-filename `409` +
      `force=true` override, `retry-instagram` on a post where Facebook succeeded but
      Instagram was made to fail, `DELETE` a history row, `?status=failed` filter,
      `?limit=&offset=` pagination past one page.
- [ ] Frontend page already shipped (`life-base-fe` — "social posting page for
      Facebook/Instagram") — verify it actually against the live endpoints, not just that
      it renders.

Migration `018_social_posts.sql` — already applied (confirmed earlier this session).

## Bikes & gear maintenance

- [ ] Apply `app/internal/db/migrations/bikes/001_initial_schema.sql` to Supabase — **not
      applied yet**.
- [ ] Supabase dashboard: expose the `bikes` schema under Settings → API → Exposed
      schemas. Every `/v1/bikes/...`, `/v1/gear`, `/v1/bottles`, `/v1/supplies` endpoint
      fails until this is done.
- [ ] Local dev via `docker compose up`: `docker-compose.yml`'s `rest` service currently
      sets `PGRST_DB_SCHEMA: public` only — needs `public,bikes` (or however PostgREST's
      multi-schema env var is comma-separated in the version pinned) to test Bikes against
      the local stack at all.
- [ ] Register a real Strava API app (developers.strava.com) — the old track-life-v2 one
      may be tied to its old Supabase project/redirect URI, don't assume it's reusable as-is.
- [ ] Set env vars: `STRAVA_CLIENT_ID`, `STRAVA_CLIENT_SECRET`, and a strong random
      `STRAVA_TOKEN_ENCRYPTION_KEY` (this one you generate yourself, not from Strava).
- [ ] Smoke test plain CRUD first (no Strava needed): create a bike, add a component with
      `replacement_interval_km`, log an activity against the bike, confirm
      `GET .../components` shows `wear_percentage`/`is_due` updated and the bike's
      `mileage` incremented. Same for a maintenance task via `.../complete`.
- [ ] Smoke test the `replace` component flow and confirm `accumulated_km` actually
      resets (not just that the endpoint returns 200).
- [ ] Smoke test Strava end-to-end once credentials exist: `authorize` → real Strava
      login/consent → `callback` → `status` shows connected → `activities` preview
      returns real rides → confirming one via `POST .../activities` accumulates wear
      correctly.
- [ ] `[fe]` Bikes pages don't exist yet in `life-base-fe` — spec is
      `docs/bikes-frontend-handoff.md`. Nothing to test on the frontend side until those
      are built.

## General

- [ ] Both domains' new env vars need to land in whatever secrets store the Render
      deploy actually reads from, not just `.env.example`/local `.env`.
- [ ] After both are verified, decommission steps from `ledger-api-fe/docs/HUB_PRD.md`
      §6 apply to `track-life-v2` (archive repo + Supabase project, reconsider the Strava
      app registration tied to it) — only once Bikes is confirmed live and working here.
