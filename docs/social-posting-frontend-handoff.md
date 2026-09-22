# Social posting — frontend handoff

## Context

`life-base-api` now has endpoints to post an image to Facebook + Instagram (the
WallyRides CR page/account), browse posting history, retry a failed Instagram leg, and
delete a history row. This replaces a manual Python script that used to be run by hand.
We need a page/section so posting + history management can happen from the app instead
of the terminal.

## Auth

Same as every other endpoint in this app: `Authorization: Bearer <supabase-jwt>` header,
the logged-in user's Supabase session token. No separate auth needed for this feature.

## Endpoints

Base path: `/v1/social/posts`

### `POST /v1/social/posts` — create a post

Request: `multipart/form-data`
| field     | type | required | notes |
|-----------|------|----------|-------|
| `image`   | file | yes      | the image to post (PNG/JPG) |
| `caption` | text | no       | if omitted, the backend picks a caption from a built-in rotating pool |
| `force`   | text | no       | `"true"` to bypass the duplicate-filename warning (see below) |

Response: `201 Created`, body is the created post record (see shape below).

Important: a Facebook or Instagram posting failure does **not** produce an HTTP error —
you still get `201` back with the created record, because the record itself carries the
per-network outcome. **Always render based on `facebook_status` / `instagram_status` in
the response body, not just the HTTP status code.**

Error responses:
- `400` — `image` field missing.
- `409 Conflict` — **duplicate filename**: this exact filename was already posted within
  the last 24 hours. `error` has a human-readable message including when it was posted.
  Show this as a confirm dialog ("Already posted at X — post again?") and resubmit with
  `force=true` if the user confirms.
- `422` — the attempt couldn't even be saved (not authenticated, or the image itself
  couldn't be read). Show the `error` field as a generic failure message.

### `GET /v1/social/posts?limit=&offset=&status=` — list history

All query params optional.
- `limit` — default `50`, max `200`.
- `offset` — default `0`, for pagination (page 2 = `offset=50` with the default limit).
- `status` — one of `success` / `failed` / `skipped`. Filters to posts where **either**
  network has that status (e.g. `status=failed` surfaces anything needing attention).

Response: `200 OK`, array of post records, most recent first. There's no `total count` in
the response — to build a "page N of M" UI, keep requesting with increasing `offset`
until a page comes back shorter than `limit`.

### `POST /v1/social/posts/:id/retry-instagram` — retry the Instagram leg

For a post where Facebook succeeded but Instagram failed. Reuses the already-hosted
Facebook photo, no re-upload. Response: `200 OK` with the updated post record.

Error responses:
- `409 Conflict` — not retryable: either Facebook didn't succeed on this post (resubmit
  the whole image instead — there's no Facebook retry, only Instagram) or Instagram
  already succeeded. `error` explains which.
- `422` — the retry attempt itself failed for another reason (auth, post not found, or a
  new Instagram error) — check the returned post's `instagram_error` if you also re-fetch
  it, or the top-level `error` if the call itself failed rather than the Graph API call.

**UI implication**: only show a "Retry Instagram" button when `facebook_status ===
"success"` and `instagram_status !== "success"` (i.e. `"failed"` or `"skipped"`).

### `DELETE /v1/social/posts/:id` — remove a history row

Response: `204 No Content`. This only deletes our record of the attempt — it does **not**
delete the live Facebook post or Instagram media. Use it for cleaning up
duplicates/mistakes in the history list, not as an "unpublish" action. Worth a confirm
dialog before calling it, and maybe a tooltip clarifying it doesn't touch the live posts.

## Response shape (a "post record")

```json
{
  "id": "uuid",
  "filename": "2026-09-21-week-39.png",
  "caption": "New week, new goals 💪 ...",
  "facebook_status": "success",
  "facebook_post_id": "1129280473610868_123456789",
  "facebook_photo_url": "https://scontent.xx.fbcdn.net/...",
  "facebook_permalink": "https://www.facebook.com/1129280473610868/photos/...",
  "facebook_error": null,
  "instagram_status": "success",
  "instagram_media_id": "17987654321",
  "instagram_permalink": "https://www.instagram.com/p/Cxxxxxxxxx/",
  "instagram_error": null,
  "created_at": "2026-09-21T12:00:00Z"
}
```

`facebook_status` / `instagram_status` are one of:
- `"success"` — posted, id + permalink fields are populated (permalink can occasionally
  be missing even on success — it's a best-effort second API call, not guaranteed).
- `"failed"` — attempted and failed, `*_error` has the reason (human-readable message
  from Meta's API, safe to show directly).
- `"skipped"` — **only applies to `instagram_status`**. Instagram publishes from the
  photo Facebook just hosted, so if Facebook fails, Instagram is never attempted and is
  marked `skipped` rather than `failed`. There is no case where Facebook is `skipped`.

All the `facebook_*`/`instagram_*` id/url/error fields are nullable.

## Suggested UI

1. **Upload form**: image picker + optional caption textarea (placeholder text like
   "Leave blank to use an auto-generated caption") + Post button. On a `409` duplicate
   response, show a confirm dialog and resubmit with `force=true` on confirm.
2. **After posting**: show the result inline — two rows/badges, one per network, each
   linking out via `facebook_permalink` / `instagram_permalink` when present:
   - success (green, link to the permalink)
   - failed (red, show `*_error` text, plus a "Retry" button for Instagram — see above)
   - skipped (gray, "skipped — Facebook failed")
3. **History list**: paginated table/card list from `GET /v1/social/posts` — a thumbnail
   isn't available (we don't store the original upload, only the Meta-hosted copy — link
   out via `facebook_permalink` instead of trying to embed an image), filename, caption
   (truncated), the two status badges with permalinks, created_at, a "Retry Instagram"
   button where applicable, and a delete (with confirm) action.
4. **Failed-only view**: a filter/tab using `status=failed` to see what needs attention
   without scrolling full history.

## Explicitly not supported yet (don't build UI for these)

- No scheduling / "post later" — posting happens immediately on upload.
- No Google Drive / Dropbox picker — file must be uploaded directly from the browser.
- No video/Reels support — images only.
- No networks beyond Facebook + Instagram.
- No Facebook retry — only Instagram can be retried (Facebook failing means nothing was
  ever uploaded to Meta, so "retry" there is just resubmitting the whole post).
- No stored copy of the original image — history relies on Meta's hosted copy and
  permalinks; if a post is ever deleted on Facebook/Instagram directly, the permalink in
  our history will break. (Flagged to us as a possible future improvement — not in scope
  now.)

## Reference

Implementation plan: `.claude/plans/social-posting-endpoint.md` in `life-base-api`. Code:
`app/internal/handlers/social.go` / `app/internal/services/social.go` if you need to
double-check a behavior.
