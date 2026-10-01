-- Migration 037: distinguish how a 'posted' social_posts row was actually
-- sent — direct (Post button, immediate), scheduled-cron (the hourly
-- GitHub Actions job), or scheduled-manual ("Check scheduled now" button).
-- Only meaningful when post_status = 'posted'; null for draft/logged.

alter table social_posts
    add column source text check (source in ('direct', 'scheduled-cron', 'scheduled-manual'));

update social_posts set source = 'direct' where post_status = 'posted' and source is null;
