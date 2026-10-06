-- Migration 043: medical profiles — per-person (self, kids, etc.) medical
-- history: exams, vaccines, imaging, notes, with attached files (X-rays,
-- lab PDFs) in a private bucket, structured attributes (e.g. weight,
-- cholesterol) for timeline charting, and sharing via either (a) granting
-- another gmail account viewer/editor access by email, enforced off their
-- JWT email claim — no invite/signup flow needed, matches the moment they
-- log into this app with that Google account — or (b) a public read-only
-- share link (reusing share_links, migration 038) for a profile or a
-- single record.

create table medical_profiles (
    id             uuid primary key default gen_random_uuid(),
    owner_user_id  uuid not null references auth.users(id) on delete cascade,
    name           text not null,
    relationship   text,
    dob            date,
    created_at     timestamptz not null default now()
);

create table medical_profile_access (
    id          uuid primary key default gen_random_uuid(),
    profile_id  uuid not null references medical_profiles(id) on delete cascade,
    email       text not null,
    role        text not null check (role in ('viewer','editor')),
    created_at  timestamptz not null default now(),
    unique (profile_id, email)
);

create table medical_records (
    id           uuid primary key default gen_random_uuid(),
    profile_id   uuid not null references medical_profiles(id) on delete cascade,
    created_by   uuid not null references auth.users(id),
    record_type  text not null check (record_type in ('exam','vaccine','imaging','note')),
    title        text not null,
    record_date  date not null,
    attributes   jsonb not null default '{}'::jsonb,
    notes        text,
    created_at   timestamptz not null default now()
);

create table medical_record_files (
    id            uuid primary key default gen_random_uuid(),
    record_id     uuid not null references medical_records(id) on delete cascade,
    storage_path  text not null,
    file_name     text not null,
    content_type  text,
    created_at    timestamptz not null default now()
);

create index on medical_profile_access (profile_id);
create index on medical_records (profile_id);
create index on medical_records (profile_id, record_date);
create index on medical_record_files (record_id);

-- ── Access helper functions ─────────────────────────────────────────────────
-- security definer so they can read medical_profiles/medical_profile_access
-- regardless of the caller's own RLS visibility into those tables (avoids
-- recursive-policy issues) — same trust-boundary pattern used by this app's
-- other cross-table RLS checks.

create or replace function medical_profile_access_role(p_profile_id uuid)
returns text
language sql
security definer
stable
as $$
    select case
        when exists (
            select 1 from medical_profiles p
            where p.id = p_profile_id and p.owner_user_id = auth.uid()
        ) then 'owner'
        else (
            select a.role from medical_profile_access a
            where a.profile_id = p_profile_id and a.email = (auth.jwt() ->> 'email')
        )
    end
$$;

create or replace function medical_record_access_role(p_record_id uuid)
returns text
language sql
security definer
stable
as $$
    select medical_profile_access_role(
        (select r.profile_id from medical_records r where r.id = p_record_id)
    )
$$;

-- ── RLS ──────────────────────────────────────────────────────────────────

alter table medical_profiles enable row level security;

create policy "medical_profiles: owner or granted can view"
on medical_profiles for select
using (owner_user_id = auth.uid() or medical_profile_access_role(id) is not null);

create policy "medical_profiles: owner or editor can update"
on medical_profiles for update
using     (owner_user_id = auth.uid() or medical_profile_access_role(id) = 'editor')
with check (owner_user_id = auth.uid() or medical_profile_access_role(id) = 'editor');

create policy "medical_profiles: owner creates"
on medical_profiles for insert
with check (owner_user_id = auth.uid());

create policy "medical_profiles: owner deletes"
on medical_profiles for delete
using (owner_user_id = auth.uid());

alter table medical_profile_access enable row level security;

create policy "medical_profile_access: owner manages grants"
on medical_profile_access for all
using     (medical_profile_access_role(profile_id) = 'owner')
with check (medical_profile_access_role(profile_id) = 'owner');

alter table medical_records enable row level security;

create policy "medical_records: owner or granted can view"
on medical_records for select
using (medical_profile_access_role(profile_id) is not null);

create policy "medical_records: owner or editor can write"
on medical_records for all
using     (medical_profile_access_role(profile_id) in ('owner','editor'))
with check (medical_profile_access_role(profile_id) in ('owner','editor'));

alter table medical_record_files enable row level security;

create policy "medical_record_files: owner or granted can view"
on medical_record_files for select
using (medical_record_access_role(record_id) is not null);

create policy "medical_record_files: owner or editor can write"
on medical_record_files for all
using     (medical_record_access_role(record_id) in ('owner','editor'))
with check (medical_record_access_role(record_id) in ('owner','editor'));

grant all on table medical_profiles, medical_profile_access, medical_records, medical_record_files
    to anon, authenticated, service_role;

-- ── Private storage bucket for X-rays, lab PDFs, etc. ───────────────────────
-- Non-load-bearing defense-in-depth, same note as migration 039: the Go
-- backend always uploads/downloads with the service-role key.

insert into storage.buckets (id, name, public)
values ('medical-files', 'medical-files', false)
on conflict (id) do nothing;

create policy "medical files bucket: owner manages own profiles' folder"
on storage.objects for all
using (
    bucket_id = 'medical-files'
    and exists (
        select 1 from medical_profiles p
        where p.id::text = (storage.foldername(name))[1]
        and p.owner_user_id = auth.uid()
    )
)
with check (
    bucket_id = 'medical-files'
    and exists (
        select 1 from medical_profiles p
        where p.id::text = (storage.foldername(name))[1]
        and p.owner_user_id = auth.uid()
    )
);

-- ── Share-link support (migration 038's generic resource) ───────────────────
-- Widen the resource_type check constraint to admit the two new kinds.

alter table share_links drop constraint if exists share_links_resource_type_check;
alter table share_links add constraint share_links_resource_type_check
    check (resource_type in ('note', 'bike', 'medical_profile', 'medical_record'));

insert into page_access (role, page_key, allowed)
values ('admin', 'medical', true), ('member', 'medical', true)
on conflict (role, page_key) do nothing;
