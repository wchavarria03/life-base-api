# Social posting — ideas

Speculative / exploratory directions for the posting feature — not scoped, not
committed, no design done yet. Promote an idea to `enhancements.md` once it's actually
going to be built next.

## Scheduling ("post later")

The old `post-weekly-card.py` script had a `pending/` → `posted/` folder flow with a
due-date check, so posting could be queued ahead of time and a cron job would pick it up
when due. The current endpoint dropped that entirely (posts immediately on upload).
Could bring it back as a `scheduled_for` field + a cron/worker that calls the same
posting logic life-base-api already has.

## Google Drive / Dropbox source

Pull the image from a Drive/Dropbox link instead of only direct upload — this was in
scope for the original ask but deferred to keep the first cut simple.

## Video / Reels support

Images only today. Meta's Graph API supports video posts and Reels on both Facebook and
Instagram, but it's a materially different upload/processing flow (resumable upload,
processing status polling) — a bigger lift than the image path.

## Multi-image carousel posts

Instagram/Facebook both support posting multiple images as one carousel post. Would need
the create endpoint to accept multiple `image` fields and a different Graph API flow
(create N containers, then a carousel container referencing them).

## Instagram Stories

Separate Graph API flow from the feed-post one currently implemented.

## More networks

Twitter/X, LinkedIn, Threads, TikTok — WallyRides CR isn't necessarily only Meta-owned
platforms long-term. Would lean on the pluggable network abstraction from
`enhancements.md` if/when this happens.

## Recurring/templated posting

Replicate the script's actual weekly cadence (a specific day/time each week) fully
automatically rather than someone remembering to open the app and upload — ties into
scheduling above, but goes further: generate or pull the week's card automatically too,
not just post it.

## Auto-generated captions from image content

Rather than (or in addition to) the rotating caption pool, use an LLM to look at the
image and write a caption — useful if this expands beyond the fixed weekly-schedule-card
format to more varied content.

## Post performance / insights

Meta's Graph API exposes insights (reach, likes, comments) for published posts/media.
Could surface these in the history view — "this post got N likes" — using the stored
`facebook_post_id`/`instagram_media_id`.

## Approval workflow

Only relevant if more than one person ends up managing the account: draft → review →
publish, instead of every upload posting immediately. Not needed for a single-user setup.

## Image editing before posting

Crop/resize in-browser before upload, since Facebook and Instagram have different ideal
aspect ratios — avoids the image looking off on one platform.
