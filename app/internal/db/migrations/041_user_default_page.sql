-- Migration 041: default landing page preference — which menu page opens
-- when the user loads the app, instead of always Hub ("/").
-- Empty/null default_page means "use Hub", so existing rows need no backfill.

alter table user_preferences add column default_page text;
