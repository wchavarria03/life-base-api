-- Migration 032: replace one-row-per-physical-container recipient tracking
-- with a count-based allocation model.
--
-- dog_recipient_types.quantity_total still means "total physical containers
-- owned of this size" but no longer has one row per container. Instead,
-- portioning a batch creates a dog_recipient_allocations row recording how
-- many containers of a type were filled and assigned to a dog (or dog pair,
-- for a container shared between two dogs at one feeding). The "currently
-- empty" count for a type is quantity_total minus the sum of quantity across
-- its allocation rows.
--
-- dog_feed_log records one row per portion actually fed — used both as a
-- feeding history and to reconcile "did you forget to feed?" on the Dogs
-- page (see DogService.FeedReview).
--
-- No real production data exists yet in dog_recipients, so it's dropped
-- rather than migrated.

drop table if exists dog_recipients;

create table dog_recipient_allocations (
    id                uuid primary key default gen_random_uuid(),
    user_id           uuid not null references auth.users(id) on delete cascade,
    recipient_type_id uuid not null references dog_recipient_types(id) on delete cascade,
    dog_id_1          uuid not null references dogs(id) on delete cascade,
    dog_id_2          uuid references dogs(id) on delete set null,
    quantity          int not null check (quantity > 0),
    portioned_at      timestamptz not null default now(),
    created_at        timestamptz not null default now()
);

create table dog_feed_log (
    id                uuid primary key default gen_random_uuid(),
    user_id           uuid not null references auth.users(id) on delete cascade,
    dog_id            uuid not null references dogs(id) on delete cascade,
    recipient_type_id uuid references dog_recipient_types(id) on delete set null,
    fed_at            timestamptz not null default now(),
    created_at        timestamptz not null default now()
);

create index on dog_recipient_allocations (user_id);
create index on dog_recipient_allocations (recipient_type_id);
create index on dog_recipient_allocations (dog_id_1);
create index on dog_recipient_allocations (dog_id_2);
create index on dog_feed_log (user_id);
create index on dog_feed_log (dog_id);
create index on dog_feed_log (fed_at);

-- ── RLS ──────────────────────────────────────────────────────────────────────

alter table dog_recipient_allocations enable row level security;
alter table dog_feed_log              enable row level security;

create policy "dog_recipient_allocations: user owns rows" on dog_recipient_allocations for all
using (user_id = auth.uid()) with check (user_id = auth.uid());

create policy "dog_feed_log: user owns rows" on dog_feed_log for all
using (user_id = auth.uid()) with check (user_id = auth.uid());

grant all on table dog_recipient_allocations to anon, authenticated, service_role;
grant all on table dog_feed_log              to anon, authenticated, service_role;
