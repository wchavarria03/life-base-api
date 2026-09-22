# Bikes module — frontend handoff

## Context

`life-base-api` now has a full Bikes & gear maintenance backend (HUB_PLAN.md Phase 2
`[api]`) — bikes, components with wear tracking, service logs, maintenance tasks, gear,
bottle-cleaning tracker, supplies, activities, and Strava OAuth + activity import preview.
This is a fresh implementation (not a port of track-life-v2's data — that app is being
retired, no data migration). Needs `[fe]` pages built against it.

## Before this can be tested end-to-end

1. **Supabase dashboard**: expose the `bikes` Postgres schema under Settings → API →
   Exposed schemas. Every endpoint below 404s/500s until this is done.
2. **Strava app registration**: `STRAVA_CLIENT_ID`, `STRAVA_CLIENT_SECRET`, and a random
   `STRAVA_TOKEN_ENCRYPTION_KEY` need to be set as real env vars on the backend before the
   Strava-related endpoints work. Not done yet — backend structure is built, credentials
   are a manual follow-up.

## Auth

Same as every other endpoint: `Authorization: Bearer <supabase-jwt>`.

## Endpoints

All under `/v1`. Path params: `:id` is always the bike id where nested under
`/bikes/:id/...`; nested resources use their own id param (`:componentId`, `:taskId`,
etc).

### Bikes — `/v1/bikes`
- `GET /v1/bikes` — list.
- `POST /v1/bikes` — body: `{name, type, model, purchase_date?, purchase_location?}`.
  `type` is one of `Road|Gravel|Mountain|Hybrid|Commuter|Other`.
- `GET /v1/bikes/:id`
- `PATCH /v1/bikes/:id` — body is a raw partial object merged directly into the row
  (whatever keys you send are the columns updated — no fixed request shape). Valid keys:
  `name, type, model, purchase_date, purchase_location`. (`mileage` is server-managed via
  activities, don't PATCH it directly.)
- `DELETE /v1/bikes/:id` — cascades to all of that bike's components, activities, etc.

### Bike fit history — `/v1/bikes/:id/fit-history`
Append-only fit session log. `GET`/`POST` (body: `date, fitter` required, plus ~20
optional numeric/text fit measurement fields — see `models/bike.go` `BikeFitHistoryInput`
for the full field list), `DELETE /v1/bikes/:id/fit-history/:fitId`.

### Components — `/v1/bikes/:id/components`
- `GET` — returns `ComponentWithStatus[]`: each row has `wear_percentage` (0–100,
  `accumulated_km / replacement_interval_km`, only present if `replacement_interval_km`
  is set) and `is_due` (derived — wear ≥100%, or `replacement_interval_days` elapsed
  since `last_replaced_date`). **Neither is stored — always fresh from the read.**
- `POST` — body: `{name, last_replaced_date, brand?, model?, replacement_interval_km?,
  replacement_interval_days?, notes?}`.
- `PATCH /v1/bikes/:id/components/:componentId` — raw partial merge, same pattern as
  Bikes above.
- `DELETE /v1/bikes/:id/components/:componentId`
- `POST /v1/bikes/:id/components/:componentId/replace` — body:
  `{replaced_date, mileage_at_replacement?, notes?}`. Logs a `component_history` row and
  resets the component's `accumulated_km` to 0 and `last_replaced_date`. **Use this
  instead of PATCH when a part is physically replaced** — PATCH alone won't reset wear.
- `GET /v1/bikes/:id/components/:componentId/history` — past replacements.

### Service logs — `/v1/bikes/:id/service-logs`
`GET`/`POST` (body: `{date, description, cost?, component_id?, notes?}`),
`DELETE /v1/bikes/:id/service-logs/:logId`.

### Maintenance tasks — `/v1/bikes/:id/maintenance-tasks`
- `GET` — returns `MaintenanceTaskWithStatus[]`: `is_due` derived from
  `interval_days`/`last_completed_date`, or `interval_km` against the bike's current
  mileage vs `last_completed_km`. A task with no `last_completed_date` is always due.
- `POST` — body: `{name, description?, interval_days?, interval_km?, is_recurring?,
  priority?}` (`priority`: `low|medium|high`, default `medium`).
- `PATCH /v1/bikes/:id/maintenance-tasks/:taskId` — raw partial merge.
- `POST /v1/bikes/:id/maintenance-tasks/:taskId/complete` — no body. Sets
  `last_completed_date` to today and `last_completed_km` to the bike's current mileage.
  **Use this instead of PATCH to mark done** — it needs the bike's live mileage, which
  PATCH alone doesn't have.
- `DELETE /v1/bikes/:id/maintenance-tasks/:taskId`

### Activities — `/v1/bikes/:id/activities`
- `GET` — list for a bike, most recent first.
- `POST` — body: `{date, name, type, distance_km, duration_minutes?, elevation_gain?,
  source?, strava_activity_id?, gear_ids?, notes?}` (`type`:
  `Ride|Race|Commute|Training|Other`). **This is the write path that drives wear
  tracking** — on create, the backend adds `distance_km` onto the bike's `mileage` and
  onto every currently-active component's and gear item's accumulated distance for that
  bike. There's no PATCH — to correct an activity, delete and re-create.
- `DELETE /v1/bikes/:id/activities/:activityId`

### Gear — `/v1/gear` (not bike-nested; optionally linked via `bike_id`)
`GET`/`POST` (body: `{name, category, bike_id?, brand?, model?, purchase_date?,
purchase_location?, cost?, max_distance_km?, notes?}`, `category`:
`Shoes|Helmet|Clothing|Accessories|Electronics|Other`), `PATCH /v1/gear/:id` (raw
partial), `DELETE /v1/gear/:id`. `distance_km` accumulates automatically the same way
components do, for gear linked to a bike via `bike_id`, when that bike logs an activity.

### Bottles — `/v1/bottles`
- `GET` — returns `BottleWithStatus[]` (`is_due` from `cleaning_interval_days` vs
  `last_cleaned_date`).
- `POST` (body: `{name, last_cleaned_date?, cleaning_interval_days?}`, both optional —
  default to today / 7 days).
- `PATCH /v1/bottles/:id` — raw partial.
- `POST /v1/bottles/:id/clean` — no body, resets `last_cleaned_date` to today. Use this
  instead of PATCH to mark cleaned.
- `DELETE /v1/bottles/:id`

### Supplies — `/v1/supplies`
- `GET`/`POST` (body: `{name, category, purchase_location, purchase_date?, purchase_url?,
  cost?, estimated_lifespan_days?, current_quantity_percent?, notes?}`, `category`:
  `tool|grease|lubricant|cleaner|spare|other`).
- `PATCH /v1/supplies/:id` — raw partial.
- `POST /v1/supplies/:id/deplete` — no body. Logs the supply to history and **deletes
  the supply row** — this is the "used it all up" action, not a soft flag.
- `DELETE /v1/supplies/:id` — deletes without logging to history (use `deplete` instead
  when the intent is "ran out", plain delete for "entered by mistake").
- `GET /v1/supply-history` — depleted-supplies log (note: top-level path, not nested
  under `/supplies`).

### Strava — `/v1/bikes/strava/...`
Not bike-scoped — one connection per user.
- `GET /v1/bikes/strava/authorize?redirect_uri=<your-callback-page-url>` — returns
  `{"auth_url": "..."}`. Redirect the browser to `auth_url`; Strava will redirect back to
  your `redirect_uri` with `?code=&state=` query params.
- `GET /v1/bikes/strava/callback?code=&state=` — call this from your callback page with
  the `code`/`state` Strava gave you. Returns the connection (athlete name, expiry) on
  success.
- `GET /v1/bikes/strava/status` — `{"connected": false}` or
  `{"connected": true, "connection": {...}}`.
- `DELETE /v1/bikes/strava` — disconnect.
- `GET /v1/bikes/strava/activities?after=&before=&page=&per_page=` (`after`/`before` are
  unix timestamps) — returns a **preview** of Strava activities (`StravaActivityPreview[]`
  — `strava_id, name, type, date, distance_km, duration_minutes, elevation_gain, source`).
  **Nothing is saved by this call.** To actually import one, the user picks a bike (and
  optionally edits fields) and you `POST` it to that bike's
  `/v1/bikes/:id/activities` as a normal `ActivityInput`, setting `source: "Strava"` and
  `strava_activity_id` from the preview — this is a deliberate two-step flow (preview,
  then confirm-and-post), not an auto-sync.

## Suggested page shape

Mirrors `track-life-v2`'s UI for reference only (not its code — that app's being
retired): a bike list/detail view, tabs within a bike's detail page for
Components/Maintenance/Service Log/Activities/Fit History, a Strava "connect" button on
first use and an activity-import picker once connected, plus standalone pages for
Gear, Bottles, and Supplies (all user-level, not nested under a bike).

## Out of scope for this pass

Nutrition, Household, House, Notes, TODO — separate HUB_PLAN phases, not built yet.
