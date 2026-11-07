-- Home maintenance log (docs/HUB_PLAN_V2.md Phase A).
-- task_logs is an append-only history of work performed: written automatically
-- when a task is completed, or manually for work that was never a task
-- ("replaced the filter"), hence the nullable task_id.

alter table tasks.tasks
    add column area text,
    add column lead_time_days integer check (lead_time_days >= 0);

create table tasks.task_logs (
    id           uuid primary key default gen_random_uuid(),
    user_id      uuid not null references auth.users(id) on delete cascade,
    task_id      uuid references tasks.tasks(id) on delete set null,
    category     text not null check (category in ('household','house','todo')),
    title        text not null,
    area         text,
    performed_at date not null default current_date,
    note         text,
    cost         numeric,
    currency     text,
    created_at   timestamptz not null default now()
);

alter table tasks.task_logs enable row level security;

create policy "task_logs: user owns rows" on tasks.task_logs for all
  using (user_id = auth.uid()) with check (user_id = auth.uid());

grant all on all tables in schema tasks to anon, authenticated, service_role;

create index on tasks.task_logs (user_id, performed_at);
create index on tasks.task_logs (user_id, area);
