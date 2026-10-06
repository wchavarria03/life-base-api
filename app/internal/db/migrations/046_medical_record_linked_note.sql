-- Migration 046: link a medical record to a note — e.g. an exam comment
-- referencing the note where a nutritionist's new plan was written down.
-- set null (not cascade): deleting the note shouldn't delete the record,
-- just drop the link.

alter table medical_records add column linked_note_id uuid references notes.notes(id) on delete set null;

create index on medical_records (linked_note_id);
