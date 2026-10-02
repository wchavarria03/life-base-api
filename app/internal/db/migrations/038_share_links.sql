-- Migration 038: generic public share links — read-only, revocable, with an
-- optional expiry, for any resource type (notes and bikes today; the
-- resource_type check constraint is the only place new types get added).
--
-- The public resolver route runs with no user JWT (service-role key, same
-- trust boundary already used for shared_task_lists' public endpoints), so
-- it bypasses RLS entirely — safe here because resource_id was only ever
-- stored after an RLS-scoped ownership check at link-creation time (see
-- ShareLinkService.Create), never taken from an unauthenticated caller.

create table share_links (
    id             uuid primary key default gen_random_uuid(),
    user_id        uuid not null references auth.users(id) on delete cascade,
    resource_type  text not null check (resource_type in ('note', 'bike')),
    resource_id    uuid not null,
    token          text not null unique,
    expires_at     timestamptz,
    revoked        boolean not null default false,
    view_count     integer not null default 0,
    last_viewed_at timestamptz,
    created_at     timestamptz not null default now()
);

create index on share_links (user_id);
create index on share_links (token);

alter table share_links enable row level security;

create policy "share_links: user owns rows"
on share_links for all
using     (user_id = auth.uid())
with check (user_id = auth.uid());

grant all on table share_links to anon, authenticated, service_role;
