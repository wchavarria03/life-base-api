-- Migration 044: attribute groups on medical_records — lets a record's
-- attributes be shown grouped the way the source exam laid them out (e.g.
-- "Serie blanca", "Quimica Sanguinea") instead of one flat list.

alter table medical_records add column attribute_groups jsonb not null default '{}'::jsonb;
