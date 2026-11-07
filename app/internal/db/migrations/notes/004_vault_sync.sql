-- Notes domain — vault sync (git <-> app). See ledger-api-fe docs/HUB_PLAN_V2.md
-- Phase D and docs/NOTES_SYNC.md. Git is the source of truth for vault
-- notes; notes.notes is a derived, rebuildable index for them.
--
-- Run AFTER notes/001..003. Idempotent where practical.

alter table notes.notes
  add column if not exists source text not null default 'app'
    check (source in ('app', 'vault')),
  add column if not exists vault_path text,
  add column if not exists org_id text,
  add column if not exists content_format text not null default 'md'
    check (content_format in ('md', 'org')),
  add column if not exists vault_commit text,
  -- true = edited in the app, waiting for the homelab worker to write it to git.
  add column if not exists pending_write boolean not null default false,
  -- true = the vault changed the file while an app edit was still pending.
  -- The incoming vault version is archived in notes.note_versions with a
  -- "[vault] " title prefix; the app version stays in notes.notes.
  add column if not exists conflict boolean not null default false,
  -- soft delete: the file was removed from git.
  add column if not exists deleted_at timestamptz,
  add column if not exists sensitive boolean not null default false,
  -- NULL = follow the path/file rules; true/false = the user toggled it in
  -- the app. A flag declared inside the file itself can never be overridden off.
  add column if not exists sensitive_override boolean;

-- One note per vault file per user.
create unique index if not exists notes_user_vault_path_key
  on notes.notes (user_id, vault_path) where vault_path is not null;

create index if not exists notes_org_id_idx
  on notes.notes (user_id, org_id) where org_id is not null;

create index if not exists notes_pending_write_idx
  on notes.notes (user_id) where pending_write;

-- Full-text search over title + content. 'simple' config: the vault mixes
-- English and Spanish, so no stemming. Content is truncated because a
-- tsvector is limited to 1 MB.
alter table notes.notes
  add column if not exists search_vec tsvector
  generated always as (
    to_tsvector('simple', coalesce(title, '') || ' ' || left(coalesce(content, ''), 200000))
  ) stored;

create index if not exists notes_search_vec_idx on notes.notes using gin (search_vec);

-- Links between notes (org-roam [[id:...][label]]) for backlinks / graph.
create table if not exists notes.note_links (
    id           uuid primary key default gen_random_uuid(),
    from_note_id uuid not null references notes.notes(id) on delete cascade,
    to_org_id    text not null,
    -- resolved lazily: NULL until a note with that org_id exists.
    to_note_id   uuid references notes.notes(id) on delete set null,
    label        text,
    created_at   timestamptz not null default now()
);

create index if not exists note_links_from_idx on notes.note_links (from_note_id);
create index if not exists note_links_to_org_idx on notes.note_links (to_org_id);
create index if not exists note_links_to_note_idx on notes.note_links (to_note_id);

alter table notes.note_links enable row level security;

-- Ownership flows through the source note, same pattern as note_versions.
drop policy if exists "note_links: user owns source note" on notes.note_links;
create policy "note_links: user owns source note"
on notes.note_links for all
using (
    exists (select 1 from notes.notes n where n.id = from_note_id and n.user_id = auth.uid())
)
with check (
    exists (select 1 from notes.notes n where n.id = from_note_id and n.user_id = auth.uid())
);

grant all on table notes.note_links to anon, authenticated, service_role;

-- "Sync now" flag the homelab worker polls, plus the last synced commit.
-- One row per user.
create table if not exists notes.vault_sync_state (
    user_id        uuid primary key references auth.users(id) on delete cascade,
    requested_at   timestamptz,
    handled_at     timestamptz,
    last_commit    text,
    last_synced_at timestamptz
);

alter table notes.vault_sync_state enable row level security;

drop policy if exists "vault_sync_state: user owns row" on notes.vault_sync_state;
create policy "vault_sync_state: user owns row" on notes.vault_sync_state for all
  using (user_id = auth.uid()) with check (user_id = auth.uid());

grant all on table notes.vault_sync_state to anon, authenticated, service_role;

-- Search RPC. SECURITY INVOKER (default) so RLS applies to the caller's JWT.
-- Returns a small projection with a highlighted snippet, never the full body.
create or replace function notes.search_notes(q text, lim int default 30)
returns table (
    id uuid,
    title text,
    snippet text,
    source text,
    sensitive boolean,
    vault_path text,
    rank real
)
language sql
stable
as $$
  select n.id,
         n.title,
         ts_headline('simple', left(coalesce(n.content, ''), 20000), websearch_to_tsquery('simple', q),
                     'MaxFragments=1, MaxWords=24, MinWords=8, StartSel=<<, StopSel=>>') as snippet,
         n.source,
         n.sensitive,
         n.vault_path,
         ts_rank(n.search_vec, websearch_to_tsquery('simple', q)) as rank
  from notes.notes n
  where n.deleted_at is null
    and n.search_vec @@ websearch_to_tsquery('simple', q)
  order by rank desc, n.updated_at desc
  limit least(greatest(lim, 1), 100);
$$;

grant execute on function notes.search_notes(text, int) to anon, authenticated, service_role;
