-- Migration 047: medical attribute definitions — a per-user dictionary
-- mapping an attribute key (e.g. "cholesterol_ldl") to a friendly label and
-- a plain-language description, editable any time from Medical Settings.
-- Shared across all of a user's profiles/records — the same key means the
-- same lab value everywhere.

create table medical_attribute_defs (
    user_id      uuid not null references auth.users(id) on delete cascade,
    attr_key     text not null,
    label        text,
    description  text,
    created_at   timestamptz not null default now(),
    updated_at   timestamptz not null default now(),
    primary key (user_id, attr_key)
);

alter table medical_attribute_defs enable row level security;

create policy "medical_attribute_defs: user owns rows"
on medical_attribute_defs for all
using     (user_id = auth.uid())
with check (user_id = auth.uid());

grant all on table medical_attribute_defs to anon, authenticated, service_role;
