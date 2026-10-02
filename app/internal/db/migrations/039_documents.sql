-- Migration 039: document vault — private file storage (insurance,
-- receipts, warranty cards, etc.), unlike social-images which is public.
-- The Go backend always uploads/downloads with the service-role key (see
-- StorageRepository's own doc comment), so these storage.objects policies
-- are defense-in-depth, not load-bearing — app-level ownership is enforced
-- by the documents table's own RLS below, checked before every download.

insert into storage.buckets (id, name, public)
values ('documents', 'documents', false)
on conflict (id) do nothing;

create policy "documents bucket: user manages own folder"
on storage.objects for all
using (
    bucket_id = 'documents'
    and (storage.foldername(name))[1] = auth.uid()::text
)
with check (
    bucket_id = 'documents'
    and (storage.foldername(name))[1] = auth.uid()::text
);

create table documents (
    id            uuid primary key default gen_random_uuid(),
    user_id       uuid not null references auth.users(id) on delete cascade,
    title         text not null,
    category      text,
    file_name     text not null,
    content_type  text not null,
    file_size     bigint not null,
    storage_path  text not null,
    created_at    timestamptz not null default now()
);

create index on documents (user_id);
create index on documents (category);

alter table documents enable row level security;

create policy "documents: user owns rows"
on documents for all
using     (user_id = auth.uid())
with check (user_id = auth.uid());

grant all on table documents to anon, authenticated, service_role;

insert into page_access (role, page_key, allowed)
values ('admin', 'documents', true), ('member', 'documents', true)
on conflict (role, page_key) do nothing;
