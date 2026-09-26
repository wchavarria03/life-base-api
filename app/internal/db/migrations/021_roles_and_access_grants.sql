-- Migration 021: grant PostgREST role access to the tables added in 019.
-- Same gap as migration 004 fixed for the original tables — a table
-- created via SQL isn't reachable through PostgREST until granted to
-- anon/authenticated/service_role, regardless of its RLS policies.
-- ("postgrest 42501: permission denied for table page_access")

grant all on table households        to anon, authenticated, service_role;
grant all on table household_members to anon, authenticated, service_role;
grant all on table page_access       to anon, authenticated, service_role;
