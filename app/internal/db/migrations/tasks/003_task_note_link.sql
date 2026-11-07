-- Tasks domain — link a task to the note it came from (Notes <-> TODO).
-- A checklist item in a note can be turned into a task; the task keeps the
-- note's id so the note can list its linked tasks and the task can link back.
-- Cross-schema FK is fine: tasks and notes live in the same database (see
-- 046_medical_record_linked_note.sql for the same pattern). set null, not
-- cascade: deleting a note must not delete the tasks, just drop the link.

alter table tasks.tasks add column note_id uuid references notes.notes(id) on delete set null;

create index on tasks.tasks (note_id);
