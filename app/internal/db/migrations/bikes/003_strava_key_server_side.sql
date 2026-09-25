-- Migration 003 (bikes): stop passing the Strava token encryption key over
-- the wire on every encrypt/decrypt call.
--
-- Previously encrypt_token_pgp/decrypt_token_pgp took encryption_key as a
-- parameter, so the Go app sent STRAVA_TOKEN_ENCRYPTION_KEY in the RPC
-- request body on every call. It now reads the key from a database-level
-- setting instead, so the key only ever lives in Postgres config — it never
-- transits the PostgREST HTTP boundary.
--
-- REQUIRED MANUAL STEP (run once per environment, not part of this file —
-- the key itself must not be committed to the repo):
--   alter database postgres set app.settings.strava_encryption_key = '<the STRAVA_TOKEN_ENCRYPTION_KEY value>';
-- Then reconnect (existing sessions/pooled connections won't see the new
-- setting until they reconnect).

create or replace function bikes.encrypt_token_pgp(token text)
returns bytea
language sql
security definer
set search_path = public, extensions, pg_temp
as $$
  select extensions.pgp_sym_encrypt(token, current_setting('app.settings.strava_encryption_key'));
$$;

create or replace function bikes.decrypt_token_pgp(encrypted_token bytea)
returns text
language sql
security definer
set search_path = public, extensions, pg_temp
as $$
  select extensions.pgp_sym_decrypt(encrypted_token, current_setting('app.settings.strava_encryption_key'));
$$;

-- Old two-argument signatures are no longer callable by the app; drop them
-- so a stale client can't accidentally pass a key over HTTP again.
drop function if exists bikes.encrypt_token_pgp(text, text);
drop function if exists bikes.decrypt_token_pgp(bytea, text);

grant execute on function bikes.encrypt_token_pgp(text) to anon, authenticated, service_role;
grant execute on function bikes.decrypt_token_pgp(bytea) to anon, authenticated, service_role;
