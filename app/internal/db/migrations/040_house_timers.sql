-- Migration 040: house timers — "start a timer from the dashboard, get
-- announced when it's done" (e.g. a wall tablet in the laundry room: tap
-- "Laundry", get spoken-aloud on that device + push-notified on your phone
-- when the wash cycle should be done). No new page_key — lives on the
-- existing Hub ('/') page.

create table house_timers (
    id         uuid primary key default gen_random_uuid(),
    user_id    uuid not null references auth.users(id) on delete cascade,
    label      text not null,
    message    text not null,
    fire_at    timestamptz not null,
    announced  boolean not null default false,
    notified   boolean not null default false,
    created_at timestamptz not null default now()
);

create index on house_timers (user_id);
create index on house_timers (fire_at);

alter table house_timers enable row level security;

create policy "house_timers: user owns rows"
on house_timers for all
using     (user_id = auth.uid())
with check (user_id = auth.uid());

grant all on table house_timers to anon, authenticated, service_role;
