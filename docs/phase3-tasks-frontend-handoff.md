# Tasks module (Household + House) — frontend handoff

## Context

One generic `tasks` domain backs Household chores, House maintenance, and a TODO list —
not three separate backends. A `category` field is the only thing that distinguishes
them; Household, House, and TODO pages should each just call the same API with a
different `category` filter/value. Fresh design (Household/House were hardcoded UI
mockups in track-life-v2, nothing real to port; TODO didn't exist there at all).

**TODO is unblocked now too** — same endpoints below, `category: "todo"`. It'll most
often be one-off tasks (`due_date`, no `interval_days`) but recurring TODOs are
supported the same way as Household/House if that's ever useful (a weekly recurring
reminder-style TODO, say).

## Before this can be tested

Supabase dashboard: expose the `tasks` schema under Settings → API → Exposed schemas
(same manual step as `bikes` — see `docs/pending-domain-testing.md`).

## Auth

Same as everywhere: `Authorization: Bearer <supabase-jwt>`.

## The recurring/one-off split

Every task is one of two shapes, picked by `is_recurring`:

- **Recurring** (household chores, house maintenance — "clean bathroom every 14 days"):
  set `is_recurring: true` and `interval_days`. Due-ness is derived from
  `last_completed_date + interval_days` vs today (never completed = always due). Don't
  set `due_date` on these — it's ignored.
- **One-off** (a single repair, most TODO items — "fix the leaky faucet by Friday"):
  leave `is_recurring` false (or omit it) and set `due_date` instead. Status derives from
  `due_date` vs today, same overdue/due_today/upcoming logic as Finance's Reminders.
  Don't set `interval_days` on these.

Both shapes share one `status` field in the response so the frontend doesn't need two
different status-badge components — see the enum below.

## Endpoints — `/v1/tasks`

- `GET /v1/tasks?category=household` (or `house`, or omit `category` for everything) —
  returns `TaskWithStatus[]`.
- `POST /v1/tasks` — body: `{category, title, description?, priority?, is_recurring?,
  interval_days?, due_date?, notes?}`. `category`: `household|house|todo` (`todo` is
  accepted by the schema now even though there's no TODO page yet — don't build one,
  that's Phase 4). `priority` defaults to `medium` if omitted (`low|medium|high`).
- `PATCH /v1/tasks/:id` — raw partial merge, same convention as Bikes: whatever keys you
  send are the columns updated, no fixed request shape.
- `POST /v1/tasks/:id/complete` — no body. For a recurring task, sets
  `last_completed_date` to today (due-ness recomputes from there — the row isn't
  recreated, unlike Finance's Reminders). For a one-off task, sets `completed_at` to now.
  **Use this instead of PATCH to mark done** — it needs to know which field to touch
  based on `is_recurring`.
- `DELETE /v1/tasks/:id`

## Response shape

```json
{
  "id": "uuid",
  "category": "household",
  "title": "Clean bathroom",
  "description": null,
  "priority": "medium",
  "is_recurring": true,
  "interval_days": 14,
  "last_completed_date": "2026-09-10",
  "due_date": null,
  "completed_at": null,
  "notes": null,
  "status": "overdue",
  "created_at": "2026-09-01T12:00:00Z"
}
```

`status` is one of: `overdue`, `due_today`, `upcoming`, `completed` (one-off only —
recurring tasks never show `completed`, they just go back to `upcoming`/`overdue` after
`complete`), `no_due_date` (one-off task with no `due_date` set — not urgency-orderable,
still shown).

## Suggested page shape

Household, House, and TODO are the same page component pointed at different
`?category=` values — a task list grouped/sorted by `status` (overdue first), a form
with a recurring/one-off toggle that swaps between an interval-days input and a
due-date picker, and a "Mark done" button calling `complete`. TODO will likely want a
simpler default (one-off toggle pre-selected, since that's the common case) but it's
the same component underneath. Structural reference: `life-base-fe`'s existing
`Loans.tsx`/`Categories.tsx` pages, not `track-life-v2`'s mockups (that app's being
retired).

## Out of scope for this pass

Notes (separate handoff doc: `docs/phase4-notes-frontend-handoff.md`), Hub shell nav
(Phase 5).
