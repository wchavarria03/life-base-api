-- Bikes domain — initial schema (separate Postgres schema, own migration sequence).
-- Reference shape: track-life-v2/sql/complete-migration.sql (trimmed: no `profiles`,
-- no nutrition tables — out of scope for this domain). No data migration — track-life-v2
-- is being retired and its Bikes history isn't carried over.

create schema if not exists bikes;

create extension if not exists pgcrypto;

-- ============================================================
-- Tables
-- ============================================================

create table bikes.bikes (
    id                uuid primary key default gen_random_uuid(),
    user_id           uuid not null references auth.users(id) on delete cascade,
    name              text not null,
    type              text not null check (type in ('Road','Gravel','Mountain','Hybrid','Commuter','Other')),
    model             text not null,
    mileage           numeric not null default 0,
    purchase_date     date,
    purchase_location text,
    created_at        timestamptz not null default now(),
    updated_at        timestamptz not null default now()
);

create table bikes.bike_fit_history (
    id                         uuid primary key default gen_random_uuid(),
    bike_id                    uuid not null references bikes.bikes(id) on delete cascade,
    date                       date not null,
    fitter                     text not null,
    location                   text,
    saddle_height              numeric,
    saddle_height_over_bars    numeric,
    saddle_to_handlebar_reach  numeric,
    saddle_angle               numeric,
    saddle_fore_aft            numeric,
    saddle_brand_model         text,
    stem_length                numeric,
    stem_angle                 numeric,
    handlebar_brand_model      text,
    handlebar_width            numeric,
    handlebar_angle            numeric,
    handlebar_extension        numeric,
    brake_lever_position       text,
    crank_length               numeric,
    chainrings                 text,
    pedal_make                 text,
    pedal_model                text,
    shoe_size                  text,
    shoe_make_model            text,
    cleat_position              text,
    notes                      text,
    created_at                 timestamptz not null default now()
);

create table bikes.components (
    id                          uuid primary key default gen_random_uuid(),
    bike_id                     uuid not null references bikes.bikes(id) on delete cascade,
    name                        text not null,
    brand                       text,
    model                       text,
    is_active                   boolean not null default true,
    last_replaced_date          date not null,
    replacement_interval_km     integer,
    replacement_interval_days   integer,
    accumulated_km              numeric not null default 0,
    notes                       text,
    created_at                  timestamptz not null default now(),
    updated_at                  timestamptz not null default now()
);

create table bikes.component_history (
    id                      uuid primary key default gen_random_uuid(),
    component_id            uuid not null references bikes.components(id) on delete cascade,
    replaced_date           date not null,
    mileage_at_replacement  numeric,
    notes                   text,
    created_at              timestamptz not null default now()
);

create table bikes.activities (
    id                uuid primary key default gen_random_uuid(),
    bike_id           uuid not null references bikes.bikes(id) on delete cascade,
    date              date not null,
    name              text not null,
    type              text not null check (type in ('Ride','Race','Commute','Training','Other')),
    distance_km       numeric not null default 0,
    duration_minutes  integer,
    elevation_gain    numeric,
    source            text check (source in ('Manual','Strava','Garmin','Wahoo','Other')),
    strava_activity_id bigint,
    gear_ids          uuid[],
    notes             text,
    created_at        timestamptz not null default now()
);

create table bikes.service_logs (
    id            uuid primary key default gen_random_uuid(),
    bike_id       uuid not null references bikes.bikes(id) on delete cascade,
    component_id  uuid references bikes.components(id) on delete set null,
    date          date not null,
    description   text not null,
    cost          numeric,
    notes         text,
    created_at    timestamptz not null default now()
);

create table bikes.maintenance_tasks (
    id                    uuid primary key default gen_random_uuid(),
    bike_id               uuid not null references bikes.bikes(id) on delete cascade,
    name                  text not null,
    description           text,
    interval_days         integer,
    interval_km           integer,
    last_completed_date   date,
    last_completed_km     numeric,
    is_recurring          boolean not null default true,
    priority              text not null default 'medium' check (priority in ('low','medium','high')),
    notes                 text,
    created_at            timestamptz not null default now(),
    updated_at            timestamptz not null default now()
);

create table bikes.gear (
    id                uuid primary key default gen_random_uuid(),
    user_id           uuid not null references auth.users(id) on delete cascade,
    bike_id           uuid references bikes.bikes(id) on delete set null,
    name              text not null,
    category          text not null check (category in ('Shoes','Helmet','Clothing','Accessories','Electronics','Other')),
    brand             text,
    model             text,
    purchase_date     date,
    purchase_location text,
    cost              numeric,
    distance_km       numeric not null default 0,
    max_distance_km   numeric,
    is_active         boolean not null default true,
    notes             text,
    created_at        timestamptz not null default now(),
    updated_at        timestamptz not null default now()
);

create table bikes.bottles (
    id                      uuid primary key default gen_random_uuid(),
    user_id                 uuid not null references auth.users(id) on delete cascade,
    name                    text not null,
    last_cleaned_date       date not null default current_date,
    cleaning_interval_days  integer not null default 7,
    created_at              timestamptz not null default now(),
    updated_at              timestamptz not null default now()
);

create table bikes.supplies (
    id                          uuid primary key default gen_random_uuid(),
    user_id                     uuid not null references auth.users(id) on delete cascade,
    name                        text not null,
    category                    text not null check (category in ('tool','grease','lubricant','cleaner','spare','other')),
    purchase_date               date not null default current_date,
    purchase_location           text not null,
    purchase_url                text,
    cost                        numeric not null default 0,
    estimated_lifespan_days     integer,
    current_quantity_percent    integer not null default 100,
    notes                       text,
    created_at                  timestamptz not null default now(),
    updated_at                  timestamptz not null default now()
);

create table bikes.supply_history (
    id                 uuid primary key default gen_random_uuid(),
    user_id            uuid not null references auth.users(id) on delete cascade,
    supply_name        text not null,
    category           text not null,
    purchase_date      date not null,
    depleted_date      date not null,
    purchase_location  text not null,
    purchase_url       text,
    cost               numeric not null default 0,
    notes              text,
    created_at         timestamptz not null default now()
);

create table bikes.strava_connections (
    id                       uuid primary key default gen_random_uuid(),
    user_id                  uuid not null references auth.users(id) on delete cascade,
    athlete_id               bigint not null,
    athlete_name             text,
    access_token_encrypted   bytea not null,
    refresh_token_encrypted  bytea not null,
    expires_at               timestamptz not null,
    scope                    text,
    created_at               timestamptz not null default now(),
    updated_at               timestamptz not null default now(),
    unique (user_id)
);

create table bikes.oauth_states (
    id            uuid primary key default gen_random_uuid(),
    user_id       uuid not null references auth.users(id) on delete cascade,
    provider      text not null default 'strava',
    state         text not null unique,
    redirect_uri  text not null,
    expires_at    timestamptz not null,
    created_at    timestamptz not null default now()
);

-- ============================================================
-- Token encryption RPCs (called from Go via PostgREST /rpc/)
-- ============================================================

create or replace function bikes.encrypt_token_pgp(token text, encryption_key text)
returns bytea
language sql
security definer
set search_path = public, pg_temp
as $$
  select pgp_sym_encrypt(token, encryption_key);
$$;

create or replace function bikes.decrypt_token_pgp(encrypted_token bytea, encryption_key text)
returns text
language sql
security definer
set search_path = public, pg_temp
as $$
  select pgp_sym_decrypt(encrypted_token, encryption_key);
$$;

grant execute on function bikes.encrypt_token_pgp(text, text) to anon, authenticated, service_role;
grant execute on function bikes.decrypt_token_pgp(bytea, text) to anon, authenticated, service_role;

-- ============================================================
-- Auto-update updated_at
-- ============================================================

create or replace function bikes.set_updated_at()
returns trigger
language plpgsql
as $$
begin
  new.updated_at = now();
  return new;
end;
$$;

create trigger set_updated_at before update on bikes.bikes
  for each row execute function bikes.set_updated_at();
create trigger set_updated_at before update on bikes.components
  for each row execute function bikes.set_updated_at();
create trigger set_updated_at before update on bikes.maintenance_tasks
  for each row execute function bikes.set_updated_at();
create trigger set_updated_at before update on bikes.gear
  for each row execute function bikes.set_updated_at();
create trigger set_updated_at before update on bikes.bottles
  for each row execute function bikes.set_updated_at();
create trigger set_updated_at before update on bikes.supplies
  for each row execute function bikes.set_updated_at();
create trigger set_updated_at before update on bikes.strava_connections
  for each row execute function bikes.set_updated_at();

-- ============================================================
-- Row level security
-- ============================================================

alter table bikes.bikes enable row level security;
alter table bikes.bike_fit_history enable row level security;
alter table bikes.components enable row level security;
alter table bikes.component_history enable row level security;
alter table bikes.activities enable row level security;
alter table bikes.service_logs enable row level security;
alter table bikes.maintenance_tasks enable row level security;
alter table bikes.gear enable row level security;
alter table bikes.bottles enable row level security;
alter table bikes.supplies enable row level security;
alter table bikes.supply_history enable row level security;
alter table bikes.strava_connections enable row level security;
alter table bikes.oauth_states enable row level security;

create policy "bikes: user owns rows" on bikes.bikes for all
  using (user_id = auth.uid()) with check (user_id = auth.uid());

create policy "bike_fit_history: via bike ownership" on bikes.bike_fit_history for all
  using (exists (select 1 from bikes.bikes b where b.id = bike_id and b.user_id = auth.uid()))
  with check (exists (select 1 from bikes.bikes b where b.id = bike_id and b.user_id = auth.uid()));

create policy "components: via bike ownership" on bikes.components for all
  using (exists (select 1 from bikes.bikes b where b.id = bike_id and b.user_id = auth.uid()))
  with check (exists (select 1 from bikes.bikes b where b.id = bike_id and b.user_id = auth.uid()));

create policy "component_history: via component->bike ownership" on bikes.component_history for all
  using (exists (
    select 1 from bikes.components c join bikes.bikes b on b.id = c.bike_id
    where c.id = component_id and b.user_id = auth.uid()
  ))
  with check (exists (
    select 1 from bikes.components c join bikes.bikes b on b.id = c.bike_id
    where c.id = component_id and b.user_id = auth.uid()
  ));

create policy "activities: via bike ownership" on bikes.activities for all
  using (exists (select 1 from bikes.bikes b where b.id = bike_id and b.user_id = auth.uid()))
  with check (exists (select 1 from bikes.bikes b where b.id = bike_id and b.user_id = auth.uid()));

create policy "service_logs: via bike ownership" on bikes.service_logs for all
  using (exists (select 1 from bikes.bikes b where b.id = bike_id and b.user_id = auth.uid()))
  with check (exists (select 1 from bikes.bikes b where b.id = bike_id and b.user_id = auth.uid()));

create policy "maintenance_tasks: via bike ownership" on bikes.maintenance_tasks for all
  using (exists (select 1 from bikes.bikes b where b.id = bike_id and b.user_id = auth.uid()))
  with check (exists (select 1 from bikes.bikes b where b.id = bike_id and b.user_id = auth.uid()));

create policy "gear: user owns rows" on bikes.gear for all
  using (user_id = auth.uid()) with check (user_id = auth.uid());

create policy "bottles: user owns rows" on bikes.bottles for all
  using (user_id = auth.uid()) with check (user_id = auth.uid());

create policy "supplies: user owns rows" on bikes.supplies for all
  using (user_id = auth.uid()) with check (user_id = auth.uid());

create policy "supply_history: user owns rows" on bikes.supply_history for all
  using (user_id = auth.uid()) with check (user_id = auth.uid());

create policy "strava_connections: user owns rows" on bikes.strava_connections for all
  using (user_id = auth.uid()) with check (user_id = auth.uid());

create policy "oauth_states: user owns rows" on bikes.oauth_states for all
  using (user_id = auth.uid()) with check (user_id = auth.uid());

-- ============================================================
-- Grants + indexes
-- ============================================================

grant usage on schema bikes to anon, authenticated, service_role;
grant all on all tables in schema bikes to anon, authenticated, service_role;

create index on bikes.bikes (user_id);
create index on bikes.bike_fit_history (bike_id);
create index on bikes.components (bike_id);
create index on bikes.component_history (component_id);
create index on bikes.activities (bike_id);
create index on bikes.activities (date);
create index on bikes.service_logs (bike_id);
create index on bikes.maintenance_tasks (bike_id);
create index on bikes.gear (user_id);
create index on bikes.gear (bike_id);
create index on bikes.bottles (user_id);
create index on bikes.supplies (user_id);
create index on bikes.supply_history (user_id);
create index on bikes.strava_connections (user_id);
create index on bikes.oauth_states (state);
