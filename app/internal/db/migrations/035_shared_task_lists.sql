-- Migration 031: tokenized public share links for a task list (household /
-- house / todo). A share link needs no login at all — the token itself is
-- the access credential (see app/internal/services/shared_task_list.go).
-- Lives in the default "public" schema (unlike tasks.tasks) since it's read
-- by an unauthenticated route with the service-role key directly, not
-- through the per-user JWT/RLS path.

create table shared_task_lists (
    id         uuid primary key default gen_random_uuid(),
    user_id    uuid not null references auth.users(id) on delete cascade,
    category   text not null check (category in ('household','house','todo')),
    token      text not null unique,
    created_at timestamptz not null default now(),
    revoked    boolean not null default false
);

create index on shared_task_lists (user_id);
create index on shared_task_lists (token);

alter table shared_task_lists enable row level security;

-- Owners can manage their own share rows via the normal authenticated
-- per-user JWT path. The public /public/shared/:token routes never go
-- through this policy — they use the service-role key, which bypasses RLS
-- entirely (see databases.resolveKeys), and instead scope every query
-- explicitly by user_id/category resolved from the token.
create policy "shared_task_lists: user owns rows" on shared_task_lists for all
    using (user_id = auth.uid()) with check (user_id = auth.uid());

grant all on table shared_task_lists to anon, authenticated, service_role;
