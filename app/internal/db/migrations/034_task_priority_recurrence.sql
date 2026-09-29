-- Migration 030: recurring tasks with a shared weekly/biweekly/monthly/yearly
-- recurrence enum (same as payment_reminders.recurrence_type), plus a
-- 3-level priority scale (minor/major/critical) replacing the old
-- low/medium/high scale. due_date already exists on tasks.tasks (one-off
-- shape) and doubles as the anchor date recurrence_type advances from.

alter table tasks.tasks
    add column recurrence_type text check (recurrence_type in ('weekly','biweekly','monthly','yearly'));

-- Link back to the next auto-created occurrence, mirroring
-- payment_reminders.next_reminder_id (see 016_reminder_transaction_link.sql).
alter table tasks.tasks
    add column next_task_id uuid references tasks.tasks(id) on delete set null;

-- Widen priority from low/medium/high to minor/major/critical. Existing
-- rows map low->minor, medium->major, high->critical.
alter table tasks.tasks drop constraint if exists tasks_priority_check;

update tasks.tasks set priority = case priority
    when 'low' then 'minor'
    when 'medium' then 'major'
    when 'high' then 'critical'
    else 'minor'
end;

alter table tasks.tasks alter column priority set default 'minor';
alter table tasks.tasks add constraint tasks_priority_check
    check (priority in ('minor','major','critical'));

create index on tasks.tasks (next_task_id);
