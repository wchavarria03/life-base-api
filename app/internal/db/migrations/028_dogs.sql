-- Migration 028: BARF dog food two-stage inventory.
--
-- Two distinct states: bulk frozen bags (raw stock, restocked when low) and
-- individually-tracked portioned recipients (containers that cycle between
-- empty and portioned, each optionally assigned to one or two dogs).
-- Portioning a batch deducts weight from bulk bags; marking one fed
-- recycles it back to empty.

create table dogs (
    id             uuid primary key default gen_random_uuid(),
    user_id        uuid not null references auth.users(id) on delete cascade,
    name           text not null,
    meals_per_day  int not null default 2,
    notes          text,
    created_at     timestamptz not null default now()
);

create table dog_recipient_types (
    id             uuid primary key default gen_random_uuid(),
    user_id        uuid not null references auth.users(id) on delete cascade,
    name           text not null,
    size_grams     int not null,
    quantity_total int not null default 0,
    created_at     timestamptz not null default now()
);

create table dog_recipients (
    id                uuid primary key default gen_random_uuid(),
    user_id           uuid not null references auth.users(id) on delete cascade,
    recipient_type_id uuid not null references dog_recipient_types(id) on delete cascade,
    status            text not null default 'empty' check (status in ('empty', 'portioned')),
    dog_id_1          uuid references dogs(id) on delete set null,
    dog_id_2          uuid references dogs(id) on delete set null,
    portioned_at      timestamptz,
    created_at        timestamptz not null default now()
);

create table dog_bulk_bags (
    id                    uuid primary key default gen_random_uuid(),
    user_id               uuid not null references auth.users(id) on delete cascade,
    label                 text not null,
    total_weight_grams    int not null,
    remaining_weight_grams int not null,
    frozen_date           date not null default current_date,
    created_at            timestamptz not null default now()
);

create table dog_settings (
    user_id                  uuid primary key references auth.users(id) on delete cascade,
    low_stock_threshold_grams int not null default 0
);

create index on dog_recipient_types (user_id);
create index on dog_recipients (user_id);
create index on dog_recipients (recipient_type_id);
create index on dog_bulk_bags (user_id);

-- ── RLS ──────────────────────────────────────────────────────────────────────

alter table dogs                enable row level security;
alter table dog_recipient_types enable row level security;
alter table dog_recipients      enable row level security;
alter table dog_bulk_bags       enable row level security;
alter table dog_settings        enable row level security;

create policy "dogs: user owns rows" on dogs for all
using (user_id = auth.uid()) with check (user_id = auth.uid());

create policy "dog_recipient_types: user owns rows" on dog_recipient_types for all
using (user_id = auth.uid()) with check (user_id = auth.uid());

create policy "dog_recipients: user owns rows" on dog_recipients for all
using (user_id = auth.uid()) with check (user_id = auth.uid());

create policy "dog_bulk_bags: user owns rows" on dog_bulk_bags for all
using (user_id = auth.uid()) with check (user_id = auth.uid());

create policy "dog_settings: user owns row" on dog_settings for all
using (user_id = auth.uid()) with check (user_id = auth.uid());

grant all on table dogs                to anon, authenticated, service_role;
grant all on table dog_recipient_types to anon, authenticated, service_role;
grant all on table dog_recipients      to anon, authenticated, service_role;
grant all on table dog_bulk_bags       to anon, authenticated, service_role;
grant all on table dog_settings        to anon, authenticated, service_role;

-- ── Page access ──────────────────────────────────────────────────────────────

insert into page_access (role, page_key, allowed)
values ('admin', 'dogs', true), ('member', 'dogs', true)
on conflict (role, page_key) do nothing;
