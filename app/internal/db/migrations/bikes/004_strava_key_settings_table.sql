-- Migration 004 (bikes): replace the current_setting()-based Strava key
-- lookup from migration 003 — it doesn't work on Supabase's managed
-- Postgres, since `alter database ... set app.settings.*` requires real
-- superuser, which the SQL editor role doesn't have there
-- ("ERROR: 42501: permission denied to set parameter").
--
-- Use a locked-down settings table instead: plain CREATE TABLE/INSERT,
-- which works under normal SQL editor privileges. The table has RLS
-- enabled with zero policies and no grants to anon/authenticated/
-- service_role, so PostgREST can never read it — only the SECURITY
-- DEFINER functions below can, since they run as the table owner, which
-- bypasses RLS by default.

create table if not exists bikes.app_secrets (
  key   text primary key,
  value text not null
);

alter table bikes.app_secrets enable row level security;
-- Intentionally no policies and no grants to anon/authenticated/service_role:
-- this table is reachable only from SQL run as its owner (e.g. the
-- SECURITY DEFINER functions below), never via PostgREST.

create or replace function bikes.encrypt_token_pgp(token text)
returns bytea
language sql
security definer
set search_path = public, extensions, pg_temp
as $$
  select extensions.pgp_sym_encrypt(
    token,
    (select value from bikes.app_secrets where key = 'strava_encryption_key')
  );
$$;

create or replace function bikes.decrypt_token_pgp(encrypted_token bytea)
returns text
language sql
security definer
set search_path = public, extensions, pg_temp
as $$
  select extensions.pgp_sym_decrypt(
    encrypted_token,
    (select value from bikes.app_secrets where key = 'strava_encryption_key')
  );
$$;

grant execute on function bikes.encrypt_token_pgp(text) to anon, authenticated, service_role;
grant execute on function bikes.decrypt_token_pgp(bytea) to anon, authenticated, service_role;

-- REQUIRED MANUAL STEP (run once per environment, ordinary INSERT — not
-- part of this file, the key itself must not be committed to the repo):
--   insert into bikes.app_secrets (key, value)
--   values ('strava_encryption_key', '<the key value>')
--   on conflict (key) do update set value = excluded.value;
