-- Tasks domain — initial schema. Generic recurring/one-off task tracker shared by
-- Household, House maintenance, and (later) TODO — see docs/HUB_PLAN.md Phase 3.

create schema if not exists tasks;

create table tasks.tasks (
    id                  uuid primary key default gen_random_uuid(),
    user_id             uuid not null references auth.users(id) on delete cascade,
    category            text not null check (category in ('household','house','todo')),
    title               text not null,
    description         text,
    priority            text not null default 'medium' check (priority in ('low','medium','high')),
    is_recurring        boolean not null default false,

    -- Recurring tasks: due-ness derived from interval_days elapsed since last_completed_date.
    interval_days       integer,
    last_completed_date date,

    -- One-off tasks: status derived from due_date vs completed_at (mirrors Reminder).
    due_date            date,
    completed_at        timestamptz,

    notes               text,
    created_at          timestamptz not null default now(),
    updated_at          timestamptz not null default now()
);

create or replace function tasks.set_updated_at()
returns trigger
language plpgsql
as $$
begin
  new.updated_at = now();
  return new;
end;
$$;

create trigger set_updated_at before update on tasks.tasks
  for each row execute function tasks.set_updated_at();

alter table tasks.tasks enable row level security;

create policy "tasks: user owns rows" on tasks.tasks for all
  using (user_id = auth.uid()) with check (user_id = auth.uid());

grant usage on schema tasks to anon, authenticated, service_role;
grant all on all tables in schema tasks to anon, authenticated, service_role;

create index on tasks.tasks (user_id);
create index on tasks.tasks (category);
create index on tasks.tasks (due_date);
