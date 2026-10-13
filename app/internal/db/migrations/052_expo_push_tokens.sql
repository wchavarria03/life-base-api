-- Migration 052: Expo push tokens — one row per mobile device (ledger-mobile)
-- a user has granted notification permission on. Separate from
-- push_subscriptions (web push/VAPID) since Expo's push service uses a
-- single opaque token delivered over its own HTTPS API, not VAPID keys.

create table expo_push_tokens (
    id         uuid primary key default gen_random_uuid(),
    user_id    uuid not null references auth.users(id) on delete cascade,
    token      text not null unique,
    platform   text not null check (platform in ('ios', 'android')),
    created_at timestamptz not null default now()
);

create index on expo_push_tokens (user_id);

alter table expo_push_tokens enable row level security;

create policy "expo_push_tokens: user owns rows"
on expo_push_tokens for all
using     (user_id = auth.uid())
with check (user_id = auth.uid());

grant all on table expo_push_tokens to anon, authenticated, service_role;
