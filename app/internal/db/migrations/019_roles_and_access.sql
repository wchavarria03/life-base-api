-- Migration 019: households, roles, and a role -> page access matrix.
--
-- Adds app-level authorization on top of the existing per-user RLS
-- boundary. Scope: who can see which pages/API routes — NOT a rework of
-- existing finance-table ownership, which stays user_id = auth.uid() as
-- before. household_members.role is surfaced in the JWT via the
-- custom_access_token_hook function below, which must additionally be
-- registered in the Supabase dashboard (Authentication -> Hooks ->
-- Custom Access Token) — that wiring can't be done from SQL alone.

create table households (
    id         uuid primary key default gen_random_uuid(),
    name       text not null,
    created_at timestamptz not null default now()
);

create table household_members (
    household_id uuid not null references households(id) on delete cascade,
    user_id      uuid not null unique references auth.users(id) on delete cascade,
    role         text not null check (role in ('admin', 'member')),
    created_at   timestamptz not null default now(),
    primary key (household_id, user_id)
);

create index on household_members (user_id);

create table page_access (
    role     text not null check (role in ('admin', 'member')),
    page_key text not null,
    allowed  boolean not null default true,
    primary key (role, page_key)
);

-- ── RLS ──────────────────────────────────────────────────────────────────────
-- Both tables are readable by any authenticated user (needed for /v1/me and
-- for the JWT hook's own lookup). Writes are restricted to admins, checked
-- against the JWT's own user_role claim so the backend doesn't need a
-- separate service-role code path for admin writes — same pattern already
-- used throughout this app of pushing the boundary into RLS.

alter table households        enable row level security;
alter table household_members enable row level security;
alter table page_access       enable row level security;

create policy "households: readable by authenticated"
on households for select
using (true);

create policy "households: writable by admins"
on households for all
using     ((auth.jwt() ->> 'user_role') = 'admin')
with check ((auth.jwt() ->> 'user_role') = 'admin');

create policy "household_members: readable by authenticated"
on household_members for select
using (true);

create policy "household_members: writable by admins"
on household_members for all
using     ((auth.jwt() ->> 'user_role') = 'admin')
with check ((auth.jwt() ->> 'user_role') = 'admin');

create policy "page_access: readable by authenticated"
on page_access for select
using (true);

create policy "page_access: writable by admins"
on page_access for all
using     ((auth.jwt() ->> 'user_role') = 'admin')
with check ((auth.jwt() ->> 'user_role') = 'admin');

-- ── Seed: bootstrap the existing account(s) as admin ────────────────────────
-- This is a personal app today (at most a handful of real auth.users rows),
-- so it's safe to make every existing user an admin of a freshly created
-- household on deploy, rather than requiring a manual SQL step.

do $$
declare
    hh_id uuid;
begin
    insert into households (name) values ('Household') returning id into hh_id;

    insert into household_members (household_id, user_id, role)
    select hh_id, id, 'admin' from auth.users
    on conflict (user_id) do nothing;
end $$;

-- ── Seed: default page access ───────────────────────────────────────────────
-- Every existing page is allowed for both roles except the new admin
-- dashboard, which is admin-only. page_key values must match the frontend's
-- ProtectedRoute pageKey props exactly.

insert into page_access (role, page_key, allowed)
select r.role, p.page_key, true
from (values ('admin'), ('member')) as r(role)
cross join (values
    ('hub'), ('finance'), ('accounts'), ('loans'), ('budgets'), ('envelopes'),
    ('reminders'), ('transfers'), ('work-hours'), ('social'), ('bikes'),
    ('gear'), ('bottles'), ('supplies'), ('household'), ('house'), ('todo'),
    ('notes'), ('import'), ('categories'), ('settings'), ('help')
) as p(page_key)
on conflict (role, page_key) do nothing;

insert into page_access (role, page_key, allowed) values
    ('admin', 'admin', true),
    ('member', 'admin', false)
on conflict (role, page_key) do nothing;

-- ── Custom access token hook ─────────────────────────────────────────────────
-- Adds a top-level "user_role" claim to every issued JWT. Runs as a
-- SECURITY DEFINER function owned by supabase_auth_admin per Supabase's
-- convention for auth hooks, so it can read household_members regardless
-- of the caller's own RLS visibility.

create or replace function public.custom_access_token_hook(event jsonb)
returns jsonb
language plpgsql
stable
as $$
declare
    claims jsonb;
    user_role text;
begin
    select role into user_role
    from household_members
    where user_id = (event ->> 'user_id')::uuid;

    claims := event -> 'claims';
    claims := jsonb_set(claims, '{user_role}', to_jsonb(coalesce(user_role, 'member')));

    event := jsonb_set(event, '{claims}', claims);
    return event;
end;
$$;

grant execute on function public.custom_access_token_hook(jsonb) to supabase_auth_admin;
revoke execute on function public.custom_access_token_hook(jsonb) from authenticated, anon, public;
