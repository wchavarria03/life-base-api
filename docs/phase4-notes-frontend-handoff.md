# Notes module — frontend handoff

## Context

Standalone notes CRUD — deliberately not synced with the Obsidian vault at
`~/personal/notes`. That integration (git-based vault, markdown files, folder
structure) is a materially bigger scope than this, and was explicitly deferred rather
than built speculatively. Fresh design — Notes was a pure hardcoded mockup in
track-life-v2, nothing real to port.

## Before this can be tested

Supabase dashboard: expose the `notes` schema under Settings → API → Exposed schemas.

## Auth

Same as everywhere: `Authorization: Bearer <supabase-jwt>`.

## Endpoints — `/v1/notes`

- `GET /v1/notes` — list, most recently updated first.
- `POST /v1/notes` — body: `{title, content?}`. `title` required, `content` optional
  (treat as markdown/plain text — no rendering done server-side, that's a frontend
  concern if you want a markdown preview).
- `GET /v1/notes/:id`
- `PATCH /v1/notes/:id` — raw partial merge (send whichever of `title`/`content` you're
  changing).
- `DELETE /v1/notes/:id`

## Response shape

```json
{
  "id": "uuid",
  "title": "Grocery list",
  "content": "- milk\n- eggs",
  "created_at": "2026-09-01T12:00:00Z",
  "updated_at": "2026-09-22T09:00:00Z"
}
```

## Suggested page shape

Flat list (no folders/tags — nothing in the reference app suggested a structure worth
building speculatively), sorted by `updated_at`. A simple list+editor split view or a
list that opens a detail/edit page both work — no strong opinion here, whichever fits
the rest of `life-base-fe`'s patterns better.

## Out of scope for this pass

Obsidian vault sync, folders/tags/search/pinning, markdown rendering (frontend's call
if wanted — backend stores plain text either way).
