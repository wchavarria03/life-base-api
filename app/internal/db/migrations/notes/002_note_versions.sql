-- Notes domain — version history. Before every update, the note's current
-- title/content is archived here, so a user can see and reopen prior
-- versions (e.g. each time a nutritionist's plan is updated).

create table notes.note_versions (
    id             uuid primary key default gen_random_uuid(),
    note_id        uuid not null references notes.notes(id) on delete cascade,
    title          text not null,
    content        text,
    version_number int not null,
    created_at     timestamptz not null default now()
);

create index on notes.note_versions (note_id, version_number desc);

alter table notes.note_versions enable row level security;

-- Versions have no user_id of their own — ownership flows through the
-- parent note, same join-based check used by medical_record_files.
create policy "note_versions: user owns parent note"
on notes.note_versions for all
using (
    exists (select 1 from notes.notes n where n.id = note_id and n.user_id = auth.uid())
)
with check (
    exists (select 1 from notes.notes n where n.id = note_id and n.user_id = auth.uid())
);

grant all on table notes.note_versions to anon, authenticated, service_role;
