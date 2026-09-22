# Notes — enhancements

Concrete, scoped improvements to the existing Notes feature — things we know how to
build, just haven't yet. Speculative/exploratory stuff belongs in `ideas.md` instead.

## No search or filtering

`GET /v1/notes` returns everything, unfiltered. Fine at low volume (same reasoning as
Social posting's original list endpoint before it needed pagination) but there's no
way to find a specific note by title/content once there are more than a screenful.

## No pagination

Same shape as Social posting's original gap: `GET /v1/notes` has no `limit`/`offset`.
Not urgent at current volume, but the same fix (default+max limit, offset param) is a
known, cheap pattern to reuse from `social.go` if the list grows.

## No edit history

`updated_at` tracks the last edit, but a `PATCH` overwrites `content` with no way to
see or revert a previous version. Fine for a personal scratch-note app; worth
reconsidering only if notes start being used for anything you'd regret losing an
earlier draft of.
