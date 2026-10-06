-- Migration 049: medications tracker + allergies/conditions pinned list.

alter table medical_profiles add column allergies text[] not null default '{}';
alter table medical_profiles add column conditions text[] not null default '{}';

create table medical_medications (
    id          uuid primary key default gen_random_uuid(),
    profile_id  uuid not null references medical_profiles(id) on delete cascade,
    name        text not null,
    dose        text,
    schedule    text,
    start_date  date,
    end_date    date,
    notes       text,
    created_at  timestamptz not null default now()
);

create index on medical_medications (profile_id);

alter table medical_medications enable row level security;

create policy "medical_medications: owner or granted can view"
on medical_medications for select
using (medical_profile_access_role(profile_id) is not null);

create policy "medical_medications: owner or editor can write"
on medical_medications for all
using     (medical_profile_access_role(profile_id) in ('owner','editor'))
with check (medical_profile_access_role(profile_id) in ('owner','editor'));

grant all on table medical_medications to anon, authenticated, service_role;
