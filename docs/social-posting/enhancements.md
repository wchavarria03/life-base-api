# Social posting — enhancements

Concrete, scoped improvements to the existing feature (`app/internal/services/social.go`,
`app/internal/handlers/social.go`) — things we know how to build, just haven't yet.
Speculative/exploratory stuff belongs in `ideas.md` instead.

## Store the original uploaded image

Right now history relies entirely on Meta's hosted copy (`facebook_photo_url` +
permalinks). If a post is ever deleted on Facebook/Instagram directly, that history row
loses its image. Store the original upload in a Supabase storage bucket at post time and
serve it back from `GET /v1/social/posts` so history is self-contained. Also unlocks a
real "Retry" for the Facebook leg itself (today only Instagram is retryable, because
retrying Facebook would otherwise require re-uploading a file we no longer have).

## Async posting

`POST /v1/social/posts` blocks on two sequential Graph API calls (Facebook, then
Instagram) — a few seconds today, fine at current volume. Move to background
processing (return `202 Accepted` + poll/websocket for status) if this becomes higher
frequency or multi-user.

## Content-hash duplicate detection

The current duplicate guard (`ErrDuplicateSocialPost`) only matches on exact filename
within 24h. Doesn't catch the same image re-uploaded under a different filename. Hash
the image bytes (e.g. SHA-256) and check that instead — requires storing the hash
alongside (or instead of) relying on stored image bytes from the item above.

## Pagination metadata

`GET /v1/social/posts?limit=&offset=` has no total count or `has_more` flag — the
frontend has to detect the last page by getting back fewer rows than `limit`. Add a
`total` count (or a `has_more` boolean) to the list response.

## Failure notifications

Posting only happens when someone manually triggers it, so a failure just sits quietly
in history until someone checks. A Slack/email notification on `facebook_status` or
`instagram_status` landing as `"failed"` would close that gap.

## Configurable duplicate window

`duplicatePostWindow` (24h) is hardcoded in `social.go`. Move to `SocialConfig` /
env var if it turns out 24h is wrong for actual usage patterns.

## Pluggable network abstraction

Facebook + Instagram posting logic is hand-written against the Meta Graph API directly
in `SocialService`. If/when another network gets added (see `ideas.md`), extract a
small `Poster` interface (`Post(ctx, image, caption) (id, permalink, error)`) so adding
a network doesn't mean growing `PostImage` further.
