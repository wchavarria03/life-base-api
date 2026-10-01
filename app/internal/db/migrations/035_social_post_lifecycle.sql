-- Migration 035: draft / manually-logged social posts, plus post-hoc editing.
--
-- post_status distinguishes three kinds of social_posts row:
--   'posted' (default) — went through our Post flow, a real Graph API call.
--   'logged'            — a post that happened on Facebook/Instagram before
--                          (or outside) this app; recorded here purely for
--                          tracking, with no API call ever made.
--   'draft'              — created here, not yet posted or scheduled.
-- post_facebook/post_instagram record *intent* (which networks a draft is
-- meant for, or which networks a logged post actually went to) — for
-- 'posted' rows the existing facebook_status/instagram_status already carry
-- this, so these two columns are only authoritative for draft/logged.
-- image_storage_path is set only when we own the image file (manual/draft
-- upload, or a post whose image was edited after creation) — Delete uses it
-- to also remove the stored object; a 'posted' row's original image lives on
-- Facebook's CDN (facebook_photo_url), not our storage, so it's left null.

alter table social_posts
    add column post_status      text not null default 'posted' check (post_status in ('draft', 'logged', 'posted')),
    add column posted_at        timestamptz,
    add column edited           boolean not null default false,
    add column image_storage_path text,
    add column post_facebook    boolean not null default true,
    add column post_instagram   boolean not null default true;

update social_posts set posted_at = created_at where posted_at is null;
