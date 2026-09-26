-- Migration 024: user notification preferences — a per-user row toggling
-- push and email-digest notifications, gating features shipped in later
-- migrations (push subscriptions, the send-digest cron command).
-- Grants included up front (lesson from migration 019).

create table user_preferences (
    user_id               uuid primary key references auth.users(id) on delete cascade,
    push_enabled          boolean not null default false,
    email_digest_enabled  boolean not null default false,
    updated_at            timestamptz not null default now()
);

alter table user_preferences enable row level security;

create policy "user_preferences: user owns row"
on user_preferences for all
using     (user_id = auth.uid())
with check (user_id = auth.uid());

grant all on table user_preferences to anon, authenticated, service_role;
