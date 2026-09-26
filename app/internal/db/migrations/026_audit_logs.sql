-- Migration 026: audit_logs — makes the existing AuditLog() middleware
-- (previously log.Printf to stdout only, see middleware/auditlog.go)
-- queryable instead of living only in Render's log stream. Any
-- authenticated user can insert their own row (the middleware does this
-- automatically on every DELETE request); only admins can read the table.

create table audit_logs (
    id         uuid primary key default gen_random_uuid(),
    user_id    uuid not null references auth.users(id) on delete cascade,
    method     text not null,
    path       text not null,
    status     int not null,
    created_at timestamptz not null default now()
);

create index on audit_logs (created_at desc);

alter table audit_logs enable row level security;

create policy "audit_logs: users insert their own rows"
on audit_logs for insert
with check (user_id = auth.uid());

create policy "audit_logs: admins read all rows"
on audit_logs for select
using ((auth.jwt() ->> 'user_role') = 'admin');

grant all on table audit_logs to anon, authenticated, service_role;
