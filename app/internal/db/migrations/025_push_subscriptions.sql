-- Migration 025: push subscriptions — one row per browser/device a user has
-- granted push permission on. endpoint is unique so re-subscribing the same
-- device (e.g. after clearing storage) upserts rather than duplicating.

create table push_subscriptions (
    id         uuid primary key default gen_random_uuid(),
    user_id    uuid not null references auth.users(id) on delete cascade,
    endpoint   text not null unique,
    p256dh     text not null,
    auth_key   text not null,
    created_at timestamptz not null default now()
);

create index on push_subscriptions (user_id);

alter table push_subscriptions enable row level security;

create policy "push_subscriptions: user owns rows"
on push_subscriptions for all
using     (user_id = auth.uid())
with check (user_id = auth.uid());

grant all on table push_subscriptions to anon, authenticated, service_role;
