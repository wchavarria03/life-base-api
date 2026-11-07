-- Per-account feature toggles. page_access (role -> page_key) stays the
-- default; a row here overrides it for one account (allowed true = force on,
-- false = force off). No row = follow the role default.
-- Enforced in the API by middleware.FeatureGate, not just hidden in the UI.

create table if not exists user_page_access (
    user_id    uuid not null references auth.users(id) on delete cascade,
    page_key   text not null,
    allowed    boolean not null,
    updated_at timestamptz not null default now(),
    primary key (user_id, page_key)
);

alter table user_page_access enable row level security;

-- A user reads their own overrides (the API resolves them with the caller's
-- JWT); admins read everything.
create policy "user_page_access: read own or admin"
on user_page_access for select
using (user_id = auth.uid() or (auth.jwt() ->> 'user_role') = 'admin');

create policy "user_page_access: writable by admins"
on user_page_access for all
using     ((auth.jwt() ->> 'user_role') = 'admin')
with check ((auth.jwt() ->> 'user_role') = 'admin');

grant all on table user_page_access to anon, authenticated, service_role;
