-- Migration 055: attach documents/images to other entities (notes, task
-- logs, bikes, components, medical records) and flag sensitive documents.

alter table documents add column if not exists sensitive boolean not null default false;

create table document_links (
    id            uuid primary key default gen_random_uuid(),
    user_id       uuid not null references auth.users(id) on delete cascade,
    document_id   uuid not null references documents(id) on delete cascade,
    entity_type   text not null check (entity_type in ('note', 'task_log', 'bike', 'component', 'medical_record')),
    entity_id     uuid not null,
    created_at    timestamptz not null default now(),
    unique (document_id, entity_type, entity_id)
);

create index on document_links (user_id);
create index on document_links (entity_type, entity_id);

alter table document_links enable row level security;

create policy "document_links: user owns rows"
on document_links for all
using     (user_id = auth.uid())
with check (user_id = auth.uid());

grant all on table document_links to anon, authenticated, service_role;
