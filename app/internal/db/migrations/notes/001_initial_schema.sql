-- Notes domain — initial schema. Standalone CRUD, not synced with any external vault
-- (see docs/HUB_PLAN.md Phase 4 for the reasoning).

create schema if not exists notes;

create table notes.notes (
    id         uuid primary key default gen_random_uuid(),
    user_id    uuid not null references auth.users(id) on delete cascade,
    title      text not null,
    content    text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create or replace function notes.set_updated_at()
returns trigger
language plpgsql
as $$
begin
  new.updated_at = now();
  return new;
end;
$$;

create trigger set_updated_at before update on notes.notes
  for each row execute function notes.set_updated_at();

alter table notes.notes enable row level security;

create policy "notes: user owns rows" on notes.notes for all
  using (user_id = auth.uid()) with check (user_id = auth.uid());

grant usage on schema notes to anon, authenticated, service_role;
grant all on all tables in schema notes to anon, authenticated, service_role;

create index on notes.notes (user_id);
create index on notes.notes (updated_at);
