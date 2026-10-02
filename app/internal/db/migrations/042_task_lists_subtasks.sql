-- Migration 042: TickTick-parity additions to tasks — custom named lists,
-- subtasks, a due time alongside due_date, and free-text tags.
-- Grants included up front (lesson from migration 019).

create table task_lists (
    id          uuid primary key default gen_random_uuid(),
    user_id     uuid not null references auth.users(id) on delete cascade,
    category    text not null check (category in ('household','house','todo')),
    name        text not null,
    created_at  timestamptz not null default now()
);

alter table task_lists enable row level security;

create policy "task_lists: user owns row"
on task_lists for all
using     (user_id = auth.uid())
with check (user_id = auth.uid());

grant all on table task_lists to anon, authenticated, service_role;

create index on task_lists (user_id, category);

-- set null (not cascade): deleting a list shouldn't delete its tasks, just
-- drop them back to "no list".
alter table tasks.tasks add column list_id uuid references public.task_lists(id) on delete set null;

-- cascade: deleting a parent task removes its subtasks with it.
alter table tasks.tasks add column parent_task_id uuid references tasks.tasks(id) on delete cascade;

alter table tasks.tasks add column due_time text;

alter table tasks.tasks add column tags text[] not null default '{}';

create index on tasks.tasks (list_id);
create index on tasks.tasks (parent_task_id);
