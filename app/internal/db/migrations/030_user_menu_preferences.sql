-- Migration 030: per-user menu (nav) preferences — lets a user hide pages
-- from their own Sidebar/Navbar without affecting their role-based page
-- access. One row per (user, page) they've explicitly hidden or shown.
-- Grants included up front (lesson from migration 019).

create table user_menu_preferences (
    user_id     uuid not null references auth.users(id) on delete cascade,
    page_key    text not null,
    hidden      boolean not null default false,
    updated_at  timestamptz not null default now(),
    primary key (user_id, page_key)
);

alter table user_menu_preferences enable row level security;

create policy "user_menu_preferences: user owns row"
on user_menu_preferences for all
using     (user_id = auth.uid())
with check (user_id = auth.uid());

grant all on table user_menu_preferences to anon, authenticated, service_role;
