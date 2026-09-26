-- Migration 020: fix custom_access_token_hook (019) — it errored at login
-- ("Error running hook URI: pg-functions://postgres/public/custom_access_token_hook")
-- because supabase_auth_admin's default search_path doesn't include
-- "public", so the unqualified reference to household_members couldn't
-- resolve. Also switch to SECURITY DEFINER (owned by postgres, same as the
-- bikes.app_secrets functions in migration bikes/004) so it reads
-- household_members directly rather than depending on RLS granting
-- supabase_auth_admin access.

create or replace function public.custom_access_token_hook(event jsonb)
returns jsonb
language plpgsql
stable
security definer
set search_path = public, pg_temp
as $$
declare
    claims jsonb;
    user_role text;
begin
    select role into user_role
    from public.household_members
    where user_id = (event ->> 'user_id')::uuid;

    claims := event -> 'claims';
    claims := jsonb_set(claims, '{user_role}', to_jsonb(coalesce(user_role, 'member')));

    event := jsonb_set(event, '{claims}', claims);
    return event;
end;
$$;

grant execute on function public.custom_access_token_hook(jsonb) to supabase_auth_admin;
revoke execute on function public.custom_access_token_hook(jsonb) from authenticated, anon, public;
