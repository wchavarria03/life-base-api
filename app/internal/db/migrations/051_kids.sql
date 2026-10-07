-- Migration 051: kids — child profiles, a chore-to-coin wallet, and a shop.
--
-- A child_profiles row is either Google-linked (email set, matched live
-- against the caller's own JWT email claim — same no-invite-flow pattern
-- already used for medical_profile_access) or PIN-mode (pin_hash set,
-- no email) for a shared-device "Kid Mode" that rides on the parent's own
-- authenticated session instead of a separate login.
--
-- A linked child's JWT has a *different* auth.uid() than the parent who
-- owns the row, so every kids.* table needs an explicit, narrow SELECT
-- policy for the child in addition to the normal owner policy. Deliberately
-- no write policy for the child anywhere in this schema: every mutation
-- (completing an assigned task, a shop purchase, a penalty) goes through
-- the Go backend using auth.WithServiceRole, which does the authorization
-- check in code. Postgres RLS can't express column-level restriction (e.g.
-- "the child may toggle completed_at but not coin_value"), so this is the
-- one place in the app where the backend — not RLS — is the authorization
-- boundary. See auth/context.go's WithServiceRole doc comment.

create schema if not exists kids;

create table kids.child_profiles (
    id           uuid primary key default gen_random_uuid(),
    owner_user_id uuid not null references auth.users(id) on delete cascade,
    name         text not null,
    avatar_emoji text,
    email        text,
    pin_hash     text,
    penalty_fee  integer not null default 5,
    created_at   timestamptz not null default now(),
    unique (owner_user_id, email)
);

create table kids.shop_items (
    id           uuid primary key default gen_random_uuid(),
    owner_user_id uuid not null references auth.users(id) on delete cascade,
    name         text not null,
    description  text,
    coin_cost    integer not null,
    emoji        text,
    active       boolean not null default true,
    created_at   timestamptz not null default now()
);

create table kids.shop_orders (
    id                     uuid primary key default gen_random_uuid(),
    child_id               uuid not null references kids.child_profiles(id) on delete cascade,
    shop_item_id           uuid not null references kids.shop_items(id) on delete restrict,
    coin_cost_at_purchase  integer not null,
    status                 text not null default 'pending' check (status in ('pending','fulfilled','cancelled')),
    created_at             timestamptz not null default now(),
    fulfilled_at           timestamptz
);

create table kids.wallet_transactions (
    id            uuid primary key default gen_random_uuid(),
    child_id      uuid not null references kids.child_profiles(id) on delete cascade,
    amount        integer not null,
    reason        text not null check (reason in ('task_complete','task_penalty','shop_purchase','adjustment')),
    task_id       uuid references tasks.tasks(id) on delete set null,
    shop_order_id uuid references kids.shop_orders(id) on delete set null,
    created_at    timestamptz not null default now()
);

create index on kids.child_profiles (owner_user_id);
create index on kids.shop_items (owner_user_id);
create index on kids.shop_orders (child_id);
create index on kids.wallet_transactions (child_id);
create index on kids.wallet_transactions (task_id);

-- Task assignment: a task can be handed to a child with a coin value.
-- penalty_applied_at is the idempotency marker for the daily penalty job —
-- without it, re-running the job (or a retry) would double-penalize.
alter table tasks.tasks add column assigned_child_id uuid references kids.child_profiles(id) on delete set null;
alter table tasks.tasks add column coin_value integer;
alter table tasks.tasks add column penalty_applied_at timestamptz;

create index on tasks.tasks (assigned_child_id);

-- ── Access helper ────────────────────────────────────────────────────────
-- security definer so RLS policies on other tables can look up "is the
-- caller a linked child, and which one" without being filtered by
-- child_profiles' own RLS first (same recursion-avoidance pattern as
-- medical_profile_access_role, migration 043).

create or replace function kids.caller_child_id()
returns uuid
language sql
security definer
stable
as $$
    select id from kids.child_profiles where email = (auth.jwt() ->> 'email')
$$;

-- ── RLS ──────────────────────────────────────────────────────────────────

alter table kids.child_profiles enable row level security;

create policy "child_profiles: owner manages own kids"
on kids.child_profiles for all
using     (owner_user_id = auth.uid())
with check (owner_user_id = auth.uid());

create policy "child_profiles: linked child can view own row"
on kids.child_profiles for select
using (email = (auth.jwt() ->> 'email'));

alter table kids.shop_items enable row level security;

create policy "shop_items: owner manages own items"
on kids.shop_items for all
using     (owner_user_id = auth.uid())
with check (owner_user_id = auth.uid());

create policy "shop_items: linked child can view own parent's active items"
on kids.shop_items for select
using (
    active
    and owner_user_id = (select owner_user_id from kids.child_profiles where id = kids.caller_child_id())
);

alter table kids.shop_orders enable row level security;

create policy "shop_orders: owner manages own kids' orders"
on kids.shop_orders for all
using (
    exists (select 1 from kids.child_profiles cp where cp.id = shop_orders.child_id and cp.owner_user_id = auth.uid())
)
with check (
    exists (select 1 from kids.child_profiles cp where cp.id = shop_orders.child_id and cp.owner_user_id = auth.uid())
);

create policy "shop_orders: linked child can view own orders"
on kids.shop_orders for select
using (child_id = kids.caller_child_id());

alter table kids.wallet_transactions enable row level security;

create policy "wallet_transactions: owner views own kids' ledger"
on kids.wallet_transactions for select
using (
    exists (select 1 from kids.child_profiles cp where cp.id = wallet_transactions.child_id and cp.owner_user_id = auth.uid())
);

create policy "wallet_transactions: linked child can view own ledger"
on kids.wallet_transactions for select
using (child_id = kids.caller_child_id());

-- Deliberately no insert/update/delete policy for anyone, parent included —
-- every wallet mutation goes through the Go backend under WithServiceRole,
-- which is the only writer. This keeps the ledger append-only and tamper-
-- proof even against a parent's own forged PostgREST call.

grant usage on schema kids to anon, authenticated, service_role;
grant all on all tables in schema kids to anon, authenticated, service_role;

-- A linked child needs to see tasks assigned to them, additive to tasks.tasks'
-- existing owner-only policy (migration tasks/001) — still no write policy
-- for the child: completing an assigned task goes through the Go backend
-- under WithServiceRole, same reasoning as the rest of this migration.
create policy "tasks: assigned child can view own tasks"
on tasks.tasks for select
using (assigned_child_id = kids.caller_child_id());

insert into page_access (role, page_key, allowed)
values ('admin', 'kids', true), ('member', 'kids', true)
on conflict (role, page_key) do nothing;
